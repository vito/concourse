package builder

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagerctx"
	"go.opentelemetry.io/otel/api/trace"

	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/atc/db"
	"github.com/concourse/concourse/atc/db/lock"
	"github.com/concourse/concourse/atc/event"
	"github.com/concourse/concourse/atc/exec"
	"github.com/concourse/concourse/atc/runtime"
	"github.com/concourse/concourse/tracing"
	"github.com/concourse/concourse/vars"
)

func NewDelegateFactory() *delegateFactory {
	return &delegateFactory{}
}

type delegateFactory struct{}

func (delegate *delegateFactory) GetDelegate(build db.Build, planID atc.PlanID, buildVars *vars.BuildVariables) exec.GetDelegate {
	return NewGetDelegate(build, planID, buildVars, clock.NewClock())
}

func (delegate *delegateFactory) PutDelegate(build db.Build, planID atc.PlanID, buildVars *vars.BuildVariables) exec.PutDelegate {
	return NewPutDelegate(build, planID, buildVars, clock.NewClock())
}

func (delegate *delegateFactory) TaskDelegate(build db.Build, planID atc.PlanID, buildVars *vars.BuildVariables) exec.TaskDelegate {
	return NewTaskDelegate(build, planID, buildVars, clock.NewClock())
}

func (delegate *delegateFactory) CheckDelegate(build db.Build, plan atc.Plan, buildVars *vars.BuildVariables) exec.CheckDelegate {
	return NewCheckDelegate(build, plan, buildVars, clock.NewClock())
}

func (delegate *delegateFactory) BuildStepDelegate(build db.Build, planID atc.PlanID, buildVars *vars.BuildVariables) exec.BuildStepDelegate {
	return NewBuildStepDelegate(build, planID, buildVars, clock.NewClock())
}

func NewGetDelegate(build db.Build, planID atc.PlanID, buildVars *vars.BuildVariables, clock clock.Clock) exec.GetDelegate {
	return &getDelegate{
		BuildStepDelegate: NewBuildStepDelegate(build, planID, buildVars, clock),

		eventOrigin: event.Origin{ID: event.OriginID(planID)},
		build:       build,
		clock:       clock,
	}
}

type getDelegate struct {
	exec.BuildStepDelegate

	build       db.Build
	eventOrigin event.Origin
	clock       clock.Clock
}

func (d *getDelegate) Initializing(logger lager.Logger) {
	err := d.build.SaveEvent(event.InitializeGet{
		Origin: d.eventOrigin,
		Time:   time.Now().Unix(),
	})
	if err != nil {
		logger.Error("failed-to-save-initialize-get-event", err)
		return
	}

	logger.Info("initializing")
}

func (d *getDelegate) Starting(logger lager.Logger) {
	err := d.build.SaveEvent(event.StartGet{
		Time:   time.Now().Unix(),
		Origin: d.eventOrigin,
	})
	if err != nil {
		logger.Error("failed-to-save-start-get-event", err)
		return
	}

	logger.Info("starting")
}

func (d *getDelegate) Finished(logger lager.Logger, exitStatus exec.ExitStatus, info runtime.VersionResult) {
	// PR#4398: close to flush stdout and stderr
	d.Stdout().(io.Closer).Close()
	d.Stderr().(io.Closer).Close()

	err := d.build.SaveEvent(event.FinishGet{
		Origin:          d.eventOrigin,
		Time:            d.clock.Now().Unix(),
		ExitStatus:      int(exitStatus),
		FetchedVersion:  info.Version,
		FetchedMetadata: info.Metadata,
	})
	if err != nil {
		logger.Error("failed-to-save-finish-get-event", err)
		return
	}

	logger.Info("finished", lager.Data{"exit-status": exitStatus})
}

func (d *getDelegate) UpdateVersion(log lager.Logger, plan atc.GetPlan, info runtime.VersionResult) {
	logger := log.WithData(lager.Data{
		"pipeline-name": d.build.PipelineName(),
		"pipeline-id":   d.build.PipelineID()},
	)

	pipeline, found, err := d.build.Pipeline()
	if err != nil {
		logger.Error("failed-to-find-pipeline", err)
		return
	}

	if !found {
		logger.Debug("pipeline-not-found")
		return
	}

	resource, found, err := pipeline.Resource(plan.Resource)
	if err != nil {
		logger.Error("failed-to-find-resource", err)
		return
	}

	if !found {
		logger.Debug("resource-not-found")
		return
	}

	_, err = resource.UpdateMetadata(
		info.Version,
		db.NewResourceConfigMetadataFields(info.Metadata),
	)
	if err != nil {
		logger.Error("failed-to-save-resource-config-version-metadata", err)
		return
	}
}

func NewPutDelegate(build db.Build, planID atc.PlanID, buildVars *vars.BuildVariables, clock clock.Clock) exec.PutDelegate {
	return &putDelegate{
		BuildStepDelegate: NewBuildStepDelegate(build, planID, buildVars, clock),

		eventOrigin: event.Origin{ID: event.OriginID(planID)},
		build:       build,
		clock:       clock,
	}
}

type putDelegate struct {
	exec.BuildStepDelegate

	build       db.Build
	eventOrigin event.Origin
	clock       clock.Clock
}

func (d *putDelegate) Initializing(logger lager.Logger) {
	err := d.build.SaveEvent(event.InitializePut{
		Origin: d.eventOrigin,
		Time:   time.Now().Unix(),
	})
	if err != nil {
		logger.Error("failed-to-save-initialize-put-event", err)
		return
	}

	logger.Info("initializing")
}

func (d *putDelegate) Starting(logger lager.Logger) {
	err := d.build.SaveEvent(event.StartPut{
		Time:   time.Now().Unix(),
		Origin: d.eventOrigin,
	})
	if err != nil {
		logger.Error("failed-to-save-start-put-event", err)
		return
	}

	logger.Info("starting")
}

func (d *putDelegate) Finished(logger lager.Logger, exitStatus exec.ExitStatus, info runtime.VersionResult) {
	// PR#4398: close to flush stdout and stderr
	d.Stdout().(io.Closer).Close()
	d.Stderr().(io.Closer).Close()

	err := d.build.SaveEvent(event.FinishPut{
		Origin:          d.eventOrigin,
		Time:            d.clock.Now().Unix(),
		ExitStatus:      int(exitStatus),
		CreatedVersion:  info.Version,
		CreatedMetadata: info.Metadata,
	})
	if err != nil {
		logger.Error("failed-to-save-finish-put-event", err)
		return
	}

	logger.Info("finished", lager.Data{"exit-status": exitStatus, "version-info": info})
}

func (d *putDelegate) SaveOutput(log lager.Logger, plan atc.PutPlan, source atc.Source, resourceTypes atc.VersionedResourceTypes, info runtime.VersionResult) {
	logger := log.WithData(lager.Data{
		"step":          plan.Name,
		"resource":      plan.Resource,
		"resource-type": plan.Type,
		"version":       info.Version,
	})

	err := d.build.SaveOutput(
		plan.Type,
		source,
		resourceTypes,
		info.Version,
		db.NewResourceConfigMetadataFields(info.Metadata),
		plan.Name,
		plan.Resource,
	)
	if err != nil {
		logger.Error("failed-to-save-output", err)
		return
	}
}

func NewTaskDelegate(build db.Build, planID atc.PlanID, buildVars *vars.BuildVariables, clock clock.Clock) exec.TaskDelegate {
	return &taskDelegate{
		BuildStepDelegate: NewBuildStepDelegate(build, planID, buildVars, clock),

		eventOrigin: event.Origin{ID: event.OriginID(planID)},
		build:       build,
	}
}

type taskDelegate struct {
	exec.BuildStepDelegate
	config      atc.TaskConfig
	build       db.Build
	eventOrigin event.Origin
}

func (d *taskDelegate) SetTaskConfig(config atc.TaskConfig) {
	d.config = config
}

func (d *taskDelegate) Initializing(logger lager.Logger) {
	err := d.build.SaveEvent(event.InitializeTask{
		Origin:     d.eventOrigin,
		Time:       time.Now().Unix(),
		TaskConfig: event.ShadowTaskConfig(d.config),
	})
	if err != nil {
		logger.Error("failed-to-save-initialize-task-event", err)
		return
	}

	logger.Info("initializing")
}

func (d *taskDelegate) Starting(logger lager.Logger) {
	err := d.build.SaveEvent(event.StartTask{
		Origin:     d.eventOrigin,
		Time:       time.Now().Unix(),
		TaskConfig: event.ShadowTaskConfig(d.config),
	})
	if err != nil {
		logger.Error("failed-to-save-initialize-task-event", err)
		return
	}

	logger.Debug("starting")
}

func (d *taskDelegate) Finished(logger lager.Logger, exitStatus exec.ExitStatus) {
	// PR#4398: close to flush stdout and stderr
	d.Stdout().(io.Closer).Close()
	d.Stderr().(io.Closer).Close()

	err := d.build.SaveEvent(event.FinishTask{
		ExitStatus: int(exitStatus),
		Time:       time.Now().Unix(),
		Origin:     d.eventOrigin,
	})
	if err != nil {
		logger.Error("failed-to-save-finish-event", err)
		return
	}

	logger.Info("finished", lager.Data{"exit-status": exitStatus})
}

func NewCheckDelegate(build db.Build, plan atc.Plan, buildVars *vars.BuildVariables, clock clock.Clock) exec.CheckDelegate {
	return &checkDelegate{
		BuildStepDelegate: NewBuildStepDelegate(build, plan.ID, buildVars, clock),

		build:       build,
		plan:        plan.Check,
		eventOrigin: event.Origin{ID: event.OriginID(plan.ID)},
		clock:       clock,
	}
}

type checkDelegate struct {
	exec.BuildStepDelegate

	build       db.Build
	plan        *atc.CheckPlan
	eventOrigin event.Origin
	clock       clock.Clock

	// stashed away just so we don't have to query them multiple times
	cachedPipeline     db.Pipeline
	cachedResource     db.Resource
	cachedResourceType db.ResourceType
}

func (d *checkDelegate) FindOrCreateScope(config db.ResourceConfig) (db.ResourceConfigScope, error) {
	resource, _, err := d.resource()
	if err != nil {
		return nil, fmt.Errorf("get resource: %w", err)
	}

	scope, err := config.FindOrCreateScope(resource) // ignore found, nil is ok
	if err != nil {
		return nil, fmt.Errorf("find or create scope: %w", err)
	}

	return scope, nil
}

func (d *checkDelegate) WaitAndRun(ctx context.Context, scope db.ResourceConfigScope) (lock.Lock, bool, error) {
	logger := lagerctx.FromContext(ctx)

	var err error

	var interval time.Duration
	if d.plan.Interval != "" {
		interval, err = time.ParseDuration(d.plan.Interval)
		if err != nil {
			return nil, false, err
		}
	}

	var lock lock.Lock
	for {
		var acquired bool
		lock, acquired, err = scope.AcquireResourceCheckingLock(logger)
		if err != nil {
			return nil, false, fmt.Errorf("acquire lock: %w", err)
		}

		if acquired {
			break
		}

		d.clock.Sleep(time.Second)
	}

	// XXX(check-refactor): one interesting thing we could do here is literally
	// wait until the interval elapses.
	//
	// then all of checking would be modeled as rate limiting. we'd just make
	// sure a build was running for each resource and keep queueing up another
	// when the last one finishes.

	end, err := scope.LastCheckEndTime()
	if err != nil {
		if releaseErr := lock.Release(); releaseErr != nil {
			logger.Error("failed-to-release-lock", releaseErr)
		}

		return nil, false, fmt.Errorf("get last check end time: %w", err)
	}

	runAt := end.Add(interval)

	shouldRun := false
	if d.build.IsManuallyTriggered() {
		// do not delay manually triggered checks (or builds)
		shouldRun = true
	} else if !d.clock.Now().Before(runAt) {
		// run if we're past the last check end time
		shouldRun = true
	} else {
		// XXX(check-refactor): we could potentially sleep here until runAt is
		// reached.
		//
		// then the check build queueing logic is to just make sure there's a build
		// running for every resource, without having to check if intervals have
		// elapsed.
		//
		// this could be expanded upon to short-circuit the waiting with events
		// triggered webhooks so that webhooks are super responsive - rather than
		// queueing a build, it would just wake up a goroutine
	}

	if !shouldRun {
		err := lock.Release()
		if err != nil {
			return nil, false, fmt.Errorf("release lock: %w", err)
		}

		return nil, false, nil
	}

	// XXX(check-refactor): enforce global rate limiting

	return lock, true, nil
}

func (d *checkDelegate) PointToSavedVersions(scope db.ResourceConfigScope) error {
	resource, found, err := d.resource()
	if err != nil {
		return fmt.Errorf("get resource: %w", err)
	}

	if found {
		err := resource.SetResourceConfigScope(scope)
		if err != nil {
			return fmt.Errorf("set resource scope: %w", err)
		}
	}

	resourceType, found, err := d.resourceType()
	if err != nil {
		return fmt.Errorf("get resource type: %w", err)
	}

	if found {
		err := resourceType.SetResourceConfigScope(scope)
		if err != nil {
			return fmt.Errorf("set resource type scope: %w", err)
		}
	}

	return nil
}

func (d *checkDelegate) pipeline() (db.Pipeline, error) {
	if d.cachedPipeline != nil {
		return d.cachedPipeline, nil
	}

	pipeline, found, err := d.build.Pipeline()
	if err != nil {
		return nil, fmt.Errorf("get build pipeline: %w", err)
	}

	if !found {
		return nil, fmt.Errorf("pipeline not found")
	}

	d.cachedPipeline = pipeline

	return d.cachedPipeline, nil
}

func (d *checkDelegate) resource() (db.Resource, bool, error) {
	if d.plan.Resource == "" {
		return nil, false, nil
	}

	if d.cachedResource != nil {
		return d.cachedResource, true, nil
	}

	pipeline, err := d.pipeline()
	if err != nil {
		return nil, false, err
	}

	resource, found, err := pipeline.Resource(d.plan.Resource)
	if err != nil {
		return nil, false, fmt.Errorf("get pipeline resource: %w", err)
	}

	if !found {
		return nil, false, fmt.Errorf("resource '%s' deleted", d.plan.Resource)
	}

	d.cachedResource = resource

	return d.cachedResource, true, nil
}

func (d *checkDelegate) resourceType() (db.ResourceType, bool, error) {
	if d.plan.ResourceType == "" {
		return nil, false, nil
	}

	if d.cachedResourceType != nil {
		return d.cachedResourceType, true, nil
	}

	pipeline, err := d.pipeline()
	if err != nil {
		return nil, false, err
	}

	resourceType, found, err := pipeline.ResourceType(d.plan.ResourceType)
	if err != nil {
		return nil, false, fmt.Errorf("get pipeline resource type: %w", err)
	}

	if !found {
		return nil, false, fmt.Errorf("resource type '%s' deleted", d.plan.ResourceType)
	}

	d.cachedResourceType = resourceType

	return d.cachedResourceType, true, nil
}

func NewBuildStepDelegate(
	build db.Build,
	planID atc.PlanID,
	buildVars *vars.BuildVariables,
	clock clock.Clock,
) *buildStepDelegate {
	return &buildStepDelegate{
		build:     build,
		planID:    planID,
		clock:     clock,
		buildVars: buildVars,
		stdout:    nil,
		stderr:    nil,
	}
}

type buildStepDelegate struct {
	build     db.Build
	planID    atc.PlanID
	clock     clock.Clock
	buildVars *vars.BuildVariables
	stderr    io.Writer
	stdout    io.Writer
}

func (delegate *buildStepDelegate) StartSpan(
	ctx context.Context,
	component string,
	extraAttrs tracing.Attrs,
) (context.Context, trace.Span) {
	attrs := delegate.build.TracingAttrs()
	for k, v := range extraAttrs {
		attrs[k] = v
	}

	return tracing.StartSpan(ctx, component, attrs)
}

func (delegate *buildStepDelegate) Variables() *vars.BuildVariables {
	return delegate.buildVars
}

func (delegate *buildStepDelegate) ImageVersionDetermined(resourceCache db.UsedResourceCache) error {
	return delegate.build.SaveImageResourceVersion(resourceCache)
}

type credVarsIterator struct {
	line string
}

func (it *credVarsIterator) YieldCred(name, value string) {
	for _, lineValue := range strings.Split(value, "\n") {
		lineValue = strings.TrimSpace(lineValue)
		// Don't consider a single char as a secret.
		if len(lineValue) > 1 {
			it.line = strings.Replace(it.line, lineValue, "((redacted))", -1)
		}
	}
}

func (delegate *buildStepDelegate) buildOutputFilter(str string) string {
	it := &credVarsIterator{line: str}
	delegate.buildVars.IterateInterpolatedCreds(it)
	return it.line
}

func (delegate *buildStepDelegate) RedactImageSource(source atc.Source) (atc.Source, error) {
	b, err := json.Marshal(&source)
	if err != nil {
		return source, err
	}
	s := delegate.buildOutputFilter(string(b))
	newSource := atc.Source{}
	err = json.Unmarshal([]byte(s), &newSource)
	if err != nil {
		return source, err
	}
	return newSource, nil
}

func (delegate *buildStepDelegate) Stdout() io.Writer {
	if delegate.stdout == nil {
		if delegate.buildVars.RedactionEnabled() {
			delegate.stdout = newDBEventWriterWithSecretRedaction(
				delegate.build,
				event.Origin{
					Source: event.OriginSourceStdout,
					ID:     event.OriginID(delegate.planID),
				},
				delegate.clock,
				delegate.buildOutputFilter,
			)
		} else {
			delegate.stdout = newDBEventWriter(
				delegate.build,
				event.Origin{
					Source: event.OriginSourceStdout,
					ID:     event.OriginID(delegate.planID),
				},
				delegate.clock,
			)
		}
	}
	return delegate.stdout
}

func (delegate *buildStepDelegate) Stderr() io.Writer {
	if delegate.stderr == nil {
		if delegate.buildVars.RedactionEnabled() {
			delegate.stderr = newDBEventWriterWithSecretRedaction(
				delegate.build,
				event.Origin{
					Source: event.OriginSourceStderr,
					ID:     event.OriginID(delegate.planID),
				},
				delegate.clock,
				delegate.buildOutputFilter,
			)
		} else {
			delegate.stderr = newDBEventWriter(
				delegate.build,
				event.Origin{
					Source: event.OriginSourceStderr,
					ID:     event.OriginID(delegate.planID),
				},
				delegate.clock,
			)
		}
	}
	return delegate.stderr
}

func (delegate *buildStepDelegate) Initializing(logger lager.Logger) {
	err := delegate.build.SaveEvent(event.Initialize{
		Origin: event.Origin{
			ID: event.OriginID(delegate.planID),
		},
		Time: time.Now().Unix(),
	})
	if err != nil {
		logger.Error("failed-to-save-initialize-event", err)
		return
	}

	logger.Info("initializing")
}

func (delegate *buildStepDelegate) Starting(logger lager.Logger) {
	err := delegate.build.SaveEvent(event.Start{
		Origin: event.Origin{
			ID: event.OriginID(delegate.planID),
		},
		Time: time.Now().Unix(),
	})
	if err != nil {
		logger.Error("failed-to-save-start-event", err)
		return
	}

	logger.Debug("starting")
}

func (delegate *buildStepDelegate) Finished(logger lager.Logger, succeeded bool) {
	// PR#4398: close to flush stdout and stderr
	delegate.Stdout().(io.Closer).Close()
	delegate.Stderr().(io.Closer).Close()

	err := delegate.build.SaveEvent(event.Finish{
		Origin: event.Origin{
			ID: event.OriginID(delegate.planID),
		},
		Time:      time.Now().Unix(),
		Succeeded: succeeded,
	})
	if err != nil {
		logger.Error("failed-to-save-finish-event", err)
		return
	}

	logger.Info("finished")
}

func (delegate *buildStepDelegate) SelectedWorker(logger lager.Logger, workerName string) {
	err := delegate.build.SaveEvent(event.SelectedWorker{
		Time: time.Now().Unix(),
		Origin: event.Origin{
			ID: event.OriginID(delegate.planID),
		},
		WorkerName: workerName,
	})
	if err != nil {
		logger.Error("failed-to-save-selected-worker-event", err)
		return
	}
}

func (delegate *buildStepDelegate) Errored(logger lager.Logger, message string) {
	err := delegate.build.SaveEvent(event.Error{
		Message: message,
		Origin: event.Origin{
			ID: event.OriginID(delegate.planID),
		},
		Time: delegate.clock.Now().Unix(),
	})
	if err != nil {
		logger.Error("failed-to-save-error-event", err)
	}
}

func newDBEventWriter(build db.Build, origin event.Origin, clock clock.Clock) io.WriteCloser {
	return &dbEventWriter{
		build:  build,
		origin: origin,
		clock:  clock,
	}
}

type dbEventWriter struct {
	build    db.Build
	origin   event.Origin
	clock    clock.Clock
	dangling []byte
}

func (writer *dbEventWriter) Write(data []byte) (int, error) {
	text := writer.writeDangling(data)
	if text == nil {
		return len(data), nil
	}

	err := writer.saveLog(string(text))
	if err != nil {
		return 0, err
	}

	return len(data), nil
}

func (writer *dbEventWriter) writeDangling(data []byte) []byte {
	text := append(writer.dangling, data...)

	checkEncoding, _ := utf8.DecodeLastRune(text)
	if checkEncoding == utf8.RuneError {
		writer.dangling = text
		return nil
	}

	writer.dangling = nil
	return text
}

func (writer *dbEventWriter) saveLog(text string) error {
	return writer.build.SaveEvent(event.Log{
		Time:    writer.clock.Now().Unix(),
		Payload: text,
		Origin:  writer.origin,
	})
}

func (writer *dbEventWriter) Close() error {
	return nil
}

func newDBEventWriterWithSecretRedaction(build db.Build, origin event.Origin, clock clock.Clock, filter exec.BuildOutputFilter) io.Writer {
	return &dbEventWriterWithSecretRedaction{
		dbEventWriter: dbEventWriter{
			build:  build,
			origin: origin,
			clock:  clock,
		},
		filter: filter,
	}
}

type dbEventWriterWithSecretRedaction struct {
	dbEventWriter
	filter exec.BuildOutputFilter
}

func (writer *dbEventWriterWithSecretRedaction) Write(data []byte) (int, error) {
	var text []byte

	if data != nil {
		text = writer.writeDangling(data)
		if text == nil {
			return len(data), nil
		}
	} else {
		if writer.dangling == nil || len(writer.dangling) == 0 {
			return 0, nil
		}
		text = writer.dangling
	}

	payload := string(text)
	if data != nil {
		idx := strings.LastIndex(payload, "\n")
		if idx >= 0 && idx < len(payload) {
			// Cache content after the last new-line, and proceed contents
			// before the last new-line.
			writer.dangling = ([]byte)(payload[idx+1:])
			payload = payload[:idx+1]
		} else {
			// No new-line found, then cache the log.
			writer.dangling = text
			return len(data), nil
		}
	}

	payload = writer.filter(payload)
	err := writer.saveLog(payload)
	if err != nil {
		return 0, err
	}

	return len(data), nil
}

func (writer *dbEventWriterWithSecretRedaction) Close() error {
	writer.Write(nil)
	return nil
}
