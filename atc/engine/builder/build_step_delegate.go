package builder

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/atc/db"
	"github.com/concourse/concourse/atc/event"
	"github.com/concourse/concourse/atc/exec"
	"github.com/concourse/concourse/atc/runtime"
	"github.com/concourse/concourse/tracing"
	"go.opentelemetry.io/otel/api/trace"
)

type buildStepDelegate struct {
	build  db.Build
	planID atc.PlanID
	clock  clock.Clock
	state  exec.RunState
	stderr io.Writer
	stdout io.Writer
}

func NewBuildStepDelegate(
	build db.Build,
	planID atc.PlanID,
	state exec.RunState,
	clock clock.Clock,
) *buildStepDelegate {
	return &buildStepDelegate{
		build:  build,
		planID: planID,
		clock:  clock,
		state:  state,
		stdout: nil,
		stderr: nil,
	}
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

func (delegate *buildStepDelegate) Stdout() io.Writer {
	if delegate.stdout != nil {
		return delegate.stdout
	}

	if delegate.state.RedactionEnabled() {
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
	return delegate.stdout
}

func (delegate *buildStepDelegate) Stderr() io.Writer {
	if delegate.stderr != nil {
		return delegate.stderr
	}

	if delegate.state.RedactionEnabled() {
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

// Name of the artifact fetched when using image_resource. Note that this only
// exists within a local scope, so it doesn't pollute the build state.
const imageArtifactName = "image"

func (delegate *buildStepDelegate) FetchImage(ctx context.Context, state exec.RunState, image atc.ImageResource) (runtime.Artifact, error) {
	fetchState := state.NewLocalScope()

	version := image.Version
	if version == nil {
		checkID := delegate.planID + "/image-check"

		checkPlan := atc.Plan{
			ID: checkID,
			Check: &atc.CheckPlan{
				Name:                   imageArtifactName,
				Type:                   image.Type,
				VersionedResourceTypes: image.VersionedResourceTypes,
				Source:                 image.Source,
			},
		}

		err := delegate.build.SaveEvent(event.ImageCheck{
			Time: delegate.clock.Now().Unix(),
			Origin: event.Origin{
				ID: event.OriginID(delegate.planID),
			},
			Plan: checkPlan,
		})
		if err != nil {
			return nil, fmt.Errorf("save image check event: %w", err)
		}

		ok, err := fetchState.Run(ctx, checkPlan)
		if err != nil {
			return nil, err
		}

		if !ok {
			return nil, fmt.Errorf("image check failed")
		}

		if !fetchState.Result(checkID, &version) {
			return nil, fmt.Errorf("check did not return a version")
		}
	}

	getID := delegate.planID + "/image-get"

	getPlan := atc.Plan{
		ID: getID,
		Get: &atc.GetPlan{
			Name:                   imageArtifactName,
			Type:                   image.Type,
			VersionedResourceTypes: image.VersionedResourceTypes,
			Source:                 image.Source,
			Version:                &version,
			Params:                 image.Params,
		},
	}

	err := delegate.build.SaveEvent(event.ImageGet{
		Time: delegate.clock.Now().Unix(),
		Origin: event.Origin{
			ID: event.OriginID(delegate.planID),
		},
		Plan: getPlan,
	})
	if err != nil {
		return nil, fmt.Errorf("save image get event: %w", err)
	}

	ok, err := fetchState.Run(ctx, getPlan)
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, fmt.Errorf("image fetching failed")
	}

	var cache db.UsedResourceCache
	if !fetchState.Result(getID, &cache) {
		return nil, fmt.Errorf("get did not return a cache")
	}

	err = delegate.build.SaveImageResourceVersion(cache)
	if err != nil {
		return nil, fmt.Errorf("save image version: %w", err)
	}

	art, found := fetchState.ArtifactRepository().ArtifactFor(imageArtifactName)
	if !found {
		return nil, fmt.Errorf("fetched artifact not found")
	}

	log.Println("!!!!!!!!!!!!! ARTIFACT:", art)

	return art, nil
}

func (delegate *buildStepDelegate) redactImageSource(source atc.Source) (atc.Source, error) {
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

func (delegate *buildStepDelegate) buildOutputFilter(str string) string {
	it := &credVarsIterator{line: str}
	delegate.state.IterateInterpolatedCreds(it)
	return it.line
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
