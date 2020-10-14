package exec

import (
	"context"
	"fmt"
	"io"
	"log"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagerctx"
	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/atc/creds"
	"github.com/concourse/concourse/atc/db"
	"github.com/concourse/concourse/atc/exec/build"
	"github.com/concourse/concourse/atc/resource"
	"github.com/concourse/concourse/atc/runtime"
	"github.com/concourse/concourse/atc/worker"
	"github.com/concourse/concourse/tracing"
	"go.opentelemetry.io/otel/api/trace"
)

type ErrPipelineNotFound struct {
	PipelineName string
}

func (e ErrPipelineNotFound) Error() string {
	return fmt.Sprintf("pipeline '%s' not found", e.PipelineName)
}

type ErrResourceNotFound struct {
	ResourceName string
}

func (e ErrResourceNotFound) Error() string {
	return fmt.Sprintf("resource '%s' not found", e.ResourceName)
}

//go:generate counterfeiter . GetDelegateFactory

type GetDelegateFactory interface {
	GetDelegate(state RunState) GetDelegate
}

//go:generate counterfeiter . GetDelegate

type GetDelegate interface {
	StartSpan(context.Context, string, tracing.Attrs) (context.Context, trace.Span)

	FetchImage(context.Context, RunState, atc.ImageResource) (runtime.Artifact, error)

	Stdout() io.Writer
	Stderr() io.Writer

	Initializing(lager.Logger)
	Starting(lager.Logger)
	Finished(lager.Logger, ExitStatus, runtime.VersionResult)
	SelectedWorker(lager.Logger, string)
	Errored(lager.Logger, string)

	UpdateVersion(lager.Logger, atc.GetPlan, runtime.VersionResult)
}

// GetStep will fetch a version of a resource on a worker that supports the
// resource type.
type GetStep struct {
	planID               atc.PlanID
	plan                 atc.GetPlan
	metadata             StepMetadata
	containerMetadata    db.ContainerMetadata
	resourceFactory      resource.ResourceFactory
	resourceCacheFactory db.ResourceCacheFactory
	strategy             worker.ContainerPlacementStrategy
	workerClient         worker.Client
	delegateFactory      GetDelegateFactory
}

func NewGetStep(
	planID atc.PlanID,
	plan atc.GetPlan,
	metadata StepMetadata,
	containerMetadata db.ContainerMetadata,
	resourceFactory resource.ResourceFactory,
	resourceCacheFactory db.ResourceCacheFactory,
	strategy worker.ContainerPlacementStrategy,
	delegateFactory GetDelegateFactory,
	client worker.Client,
) Step {
	return &GetStep{
		planID:               planID,
		plan:                 plan,
		metadata:             metadata,
		containerMetadata:    containerMetadata,
		resourceFactory:      resourceFactory,
		resourceCacheFactory: resourceCacheFactory,
		strategy:             strategy,
		delegateFactory:      delegateFactory,
		workerClient:         client,
	}
}

func (step *GetStep) Run(ctx context.Context, state RunState) (bool, error) {
	delegate := step.delegateFactory.GetDelegate(state)
	ctx, span := delegate.StartSpan(ctx, "get", tracing.Attrs{
		"name":     step.plan.Name,
		"resource": step.plan.Resource,
	})

	ok, err := step.run(ctx, state, delegate)
	tracing.End(span, err)

	return ok, err
}

func (step *GetStep) run(ctx context.Context, state RunState, delegate GetDelegate) (bool, error) {
	logger := lagerctx.FromContext(ctx)
	logger = logger.Session("get-step", lager.Data{
		"step-name": step.plan.Name,
	})

	delegate.Initializing(logger)

	source, err := creds.NewSource(state, step.plan.Source).Evaluate()
	if err != nil {
		return false, err
	}

	params, err := creds.NewParams(state, step.plan.Params).Evaluate()
	if err != nil {
		return false, err
	}

	version, err := NewVersionSourceFromPlan(&step.plan).Version(state)
	if err != nil {
		return false, err
	}

	workerSpec := worker.WorkerSpec{
		Tags:   step.plan.Tags,
		TeamID: step.metadata.TeamID,
	}

	var imageSpec worker.ImageSpec
	resourceType, found := step.plan.VersionedResourceTypes.Lookup(step.plan.Type)
	if found {
		artifact, err := delegate.FetchImage(ctx, state, atc.ImageResource{
			Type:                   resourceType.Type,
			Source:                 resourceType.Source,
			Version:                resourceType.Version,
			VersionedResourceTypes: step.plan.VersionedResourceTypes,
		})
		if err != nil {
			return false, fmt.Errorf("fetch image: %w", err)
		}

		imageSpec = worker.ImageSpec{
			ImageArtifact: artifact,
		}
	} else {
		imageSpec = worker.ImageSpec{
			ResourceType: step.plan.Type,
		}

		workerSpec.ResourceType = step.plan.Type
	}

	log.Println("!!!!!!!!!!!!! IMAGE SPEC:", imageSpec)

	containerSpec := worker.ContainerSpec{
		ImageSpec: imageSpec,
		TeamID:    step.metadata.TeamID,
		Env:       step.metadata.Env(),
	}
	tracing.Inject(ctx, &containerSpec)

	// XXX(substeps): would be cool to build this off of the fetched image resource cache
	resourceTypes, err := creds.NewVersionedResourceTypes(state, step.plan.VersionedResourceTypes).Evaluate()
	if err != nil {
		return false, err
	}

	resourceCache, err := step.resourceCacheFactory.FindOrCreateResourceCache(
		db.ForBuild(step.metadata.BuildID),
		step.plan.Type,
		version,
		source,
		params,
		resourceTypes,
	)
	if err != nil {
		logger.Error("failed-to-create-resource-cache", err)
		return false, err
	}

	processSpec := runtime.ProcessSpec{
		Path:         "/opt/resource/in",
		Args:         []string{resource.ResourcesDir("get")},
		StdoutWriter: delegate.Stdout(),
		StderrWriter: delegate.Stderr(),
	}

	resourceToGet := step.resourceFactory.NewResource(
		source,
		params,
		version,
	)

	containerOwner := db.NewBuildStepContainerOwner(step.metadata.BuildID, step.planID, step.metadata.TeamID)

	getResult, err := step.workerClient.RunGetStep(
		ctx,
		logger,
		containerOwner,
		containerSpec,
		workerSpec,
		step.strategy,
		step.containerMetadata,
		worker.ImageFetcherSpec{
			Delegate: worker.NoopImageFetchingDelegate{},
		},
		processSpec,
		delegate,
		resourceCache,
		resourceToGet,
	)
	if err != nil {
		return false, err
	}

	var succeeded bool
	if getResult.ExitStatus == 0 {
		state.StoreResult(step.planID, resourceCache)

		state.ArtifactRepository().RegisterArtifact(
			build.ArtifactName(step.plan.Name),
			getResult.GetArtifact,
		)

		if step.plan.Resource != "" {
			delegate.UpdateVersion(logger, step.plan, getResult.VersionResult)
		}

		succeeded = true
	}

	delegate.Finished(
		logger,
		ExitStatus(getResult.ExitStatus),
		getResult.VersionResult,
	)

	return succeeded, nil
}
