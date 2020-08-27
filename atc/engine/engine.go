package engine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagerctx"
	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/atc/creds"
	"github.com/concourse/concourse/atc/db"
	"github.com/concourse/concourse/atc/exec"
	"github.com/concourse/concourse/atc/metric"
	"github.com/concourse/concourse/atc/policy"
	"github.com/concourse/concourse/tracing"
)

//go:generate counterfeiter . Engine

type Engine interface {
	NewBuild(db.Build) Runnable

	Drain(context.Context)
}

//go:generate counterfeiter . Runnable

type Runnable interface {
	Run(context.Context)
}

//go:generate counterfeiter . StepBuilder

type StepBuilder interface {
	BuildStep(lager.Logger, db.Build) (exec.Step, error)

	BuildStepErrored(lager.Logger, db.Build, error)
}

func NewEngine(builder StepBuilder) Engine {
	return &engine{
		builder:       builder,
		release:       make(chan bool),
		trackedStates: new(sync.Map),
		waitGroup:     new(sync.WaitGroup),
	}
}

type engine struct {
	builder       StepBuilder
	release       chan bool
	trackedStates *sync.Map
	waitGroup     *sync.WaitGroup
}

func (engine *engine) Drain(ctx context.Context) {
	logger := lagerctx.FromContext(ctx)

	logger.Info("start")
	defer logger.Info("done")

	close(engine.release)

	logger.Info("waiting")

	engine.waitGroup.Wait()
}

func (engine *engine) NewBuild(build db.Build) Runnable {
	return NewBuild(
		build,
		engine.builder,
		engine.release,
		engine.trackedStates,
		engine.waitGroup,
	)
}

func NewBuild(
	build db.Build,
	builder StepBuilder,
	release chan bool,
	trackedStates *sync.Map,
	waitGroup *sync.WaitGroup,
) Runnable {
	return &engineBuild{
		build:   build,
		builder: builder,

		release:       release,
		trackedStates: trackedStates,
		waitGroup:     waitGroup,
	}
}

type engineBuild struct {
	build   db.Build
	builder StepBuilder

	release       chan bool
	trackedStates *sync.Map
	waitGroup     *sync.WaitGroup

	pipelineCredMgrs []creds.Manager
}

func (b *engineBuild) Run(ctx context.Context) {
	b.waitGroup.Add(1)
	defer b.waitGroup.Done()

	logger := lagerctx.FromContext(ctx).WithData(b.build.LagerData())

	lock, acquired, err := b.build.AcquireTrackingLock(logger, time.Minute)
	if err != nil {
		logger.Error("failed-to-get-lock", err)
		return
	}

	if !acquired {
		logger.Debug("build-already-tracked")
		return
	}

	defer lock.Release()

	found, err := b.build.Reload()
	if err != nil {
		logger.Error("failed-to-load-build-from-db", err)
		return
	}

	if !found {
		logger.Info("build-not-found")
		return
	}

	if !b.build.IsRunning() {
		logger.Info("build-already-finished")
		return
	}

	notifier, err := b.build.AbortNotifier()
	if err != nil {
		logger.Error("failed-to-listen-for-aborts", err)
		return
	}

	defer notifier.Close()

	ctx, span := tracing.StartSpanFollowing(ctx, b.build, "build", b.build.TracingAttrs())
	defer span.End()

	step, err := b.builder.BuildStep(logger, b.build)
	if err != nil {
		logger.Error("failed-to-build-step", err)

		// Fails the build if BuildStep returned error. Because some unrecoverable error,
		// like pipeline var_source is wrong, will cause a build to never start
		// to run.
		b.builder.BuildStepErrored(logger, b.build, err)
		b.finish(logger.Session("finish"), err, false)

		return
	}
	b.trackStarted(logger)
	defer b.trackFinished(logger)

	logger.Info("running")

	state := b.runState()
	defer b.clearRunState()

	ctx, cancel := context.WithCancel(ctx)

	noleak := make(chan bool)
	defer close(noleak)

	go func() {
		select {
		case <-noleak:
		case <-notifier.Notify():
			logger.Info("aborting")
			cancel()
		}
	}()

	done := make(chan error)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic in engine build step run %d: %v", b.build.ID(), r)

				fmt.Fprintf(os.Stderr, "%s\n %s\n", err.Error(), string(debug.Stack()))
				logger.Error("panic-in-engine-build-step-run", err)

				done <- err
			}
		}()

		ctx := lagerctx.NewContext(ctx, logger)
		ctx = policy.RecordTeamAndPipeline(ctx, b.build.TeamName(), b.build.PipelineName())
		done <- step.Run(ctx, state)
	}()

	select {
	case <-b.release:
		logger.Info("releasing")

	case err = <-done:
		logger.Debug("engine-build-done")
		if err != nil {
			if ok := errors.As(err, &exec.Retriable{}); ok {
				return
			}
		}
		b.finish(logger.Session("finish"), err, step.Succeeded())
	}
}

func (b *engineBuild) finish(logger lager.Logger, err error, succeeded bool) {
	if errors.Is(err, context.Canceled) {
		b.saveStatus(logger, atc.StatusAborted)
		logger.Info("aborted")

	} else if err != nil {
		b.saveStatus(logger, atc.StatusErrored)
		logger.Info("errored", lager.Data{"error": err.Error()})

	} else if succeeded {
		b.saveStatus(logger, atc.StatusSucceeded)
		logger.Info("succeeded")

	} else {
		b.saveStatus(logger, atc.StatusFailed)
		logger.Info("failed")
	}
}

func (b *engineBuild) saveStatus(logger lager.Logger, status atc.BuildStatus) {
	if err := b.build.Finish(db.BuildStatus(status)); err != nil {
		logger.Error("failed-to-finish-build", err)
	}
}

func (b *engineBuild) trackStarted(logger lager.Logger) {
	metric.BuildStarted{
		PipelineName: b.build.PipelineName(),
		JobName:      b.build.JobName(),
		BuildName:    b.build.Name(),
		BuildID:      b.build.ID(),
		TeamName:     b.build.TeamName(),
	}.Emit(logger)
}

func (b *engineBuild) trackFinished(logger lager.Logger) {
	found, err := b.build.Reload()
	if err != nil {
		logger.Error("failed-to-load-build-from-db", err)
		return
	}

	if !found {
		logger.Info("build-removed")
		return
	}

	if !b.build.IsRunning() {
		metric.BuildFinished{
			PipelineName:  b.build.PipelineName(),
			JobName:       b.build.JobName(),
			BuildName:     b.build.Name(),
			BuildID:       b.build.ID(),
			BuildStatus:   b.build.Status(),
			BuildDuration: b.build.EndTime().Sub(b.build.StartTime()),
			TeamName:      b.build.TeamName(),
		}.Emit(logger)
	}
}

func (b *engineBuild) runState() exec.RunState {
	id := fmt.Sprintf("build:%v", b.build.ID())
	existingState, _ := b.trackedStates.LoadOrStore(id, exec.NewRunState())
	return existingState.(exec.RunState)
}

func (b *engineBuild) clearRunState() {
	id := fmt.Sprintf("build:%v", b.build.ID())
	b.trackedStates.Delete(id)
}
