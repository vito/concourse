package db

import (
	"fmt"

	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/atc/creds"
)

type SchedulerResource interface {
	StepConfig() atc.StepConfig
	NamedResources() NamedResources
	ResourceTypes() atc.VersionedResourceTypes

	RefreshResourceConfig() (ResourceConfigScope, bool, error)

	CreateBuild() (Build, error)
}

type schedulerResource struct {
	resource Resource
	source   creds.Source
	vrts     creds.VersionedResourceTypes
}

func (sr *schedulerResource) StepConfig() atc.StepConfig {
	return sr.resource.Config().StepConfig()
}

func (sr *schedulerResource) NamedResources() NamedResources {
	return NamedResources{
		{
			Name:   sr.resource.Name(),
			Type:   sr.resource.Type(),
			Source: sr.resource.Source(),
		},
	}
}

func (sr *schedulerResource) ResourceTypes() atc.VersionedResourceTypes {
	vrts := make([]atc.VersionedResourceType, len(sr.vrts))
	for i, vrt := range sr.vrts {
		vrts[i] = vrt.VersionedResourceType
	}

	return vrts
}

func (sr *schedulerResource) RefreshResourceConfig() (ResourceConfigScope, bool, error) {
	vrts, err := sr.vrts.Evaluate()
	if err != nil {
		return nil, false, fmt.Errorf("evaluate resource types: %w", err)
	}

	source, err := sr.source.Evaluate()
	if err != nil {
		return nil, false, fmt.Errorf("evaluate source: %w", err)
	}

	scope, ok, err := sr.resource.SetResourceConfig(source, vrts)
	if err != nil {
		return nil, false, fmt.Errorf("set resource config: %w", err)
	}
	if !ok {
		return nil, false, nil
	}

	return scope, true, nil
}

func (sr *schedulerResource) CreateBuild() (Build, error) {
	return sr.resource.CreateBuild()
}
