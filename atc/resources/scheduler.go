package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/atc/db"
)

type Factory interface {
	// returns all pipeline resources which are to be scheduled
	ResourcesToSchedule(time.Duration, time.Duration) ([]db.SchedulerResource, error)
}

type Planner interface {
	Create(
		atc.StepConfig,
		db.NamedResources,
		atc.VersionedResourceTypes,
		[]db.BuildInput,
	) (atc.Plan, error)
}

type IntervalConfig struct {
	DefaultCheckInterval        time.Duration
	DefaultWebhookCheckInterval time.Duration
}

type BuildScheduler struct {
	Factory Factory
	Planner Planner

	IntervalConfig
}

func (scheduler BuildScheduler) Run(context.Context) error {
	resources, err := scheduler.Factory.ResourcesToSchedule(
		scheduler.DefaultCheckInterval,
		scheduler.DefaultWebhookCheckInterval,
	)
	if err != nil {
		return fmt.Errorf("get all resources: %w", err)
	}

	// keep track of scope IDs which have already been checked
	alreadyChecked := make(map[int]bool)

	for _, resource := range resources {
		// evaluate creds to come up with a resource config and scope
		//
		// if resource config scope is different from current one, AND has been
		// checked, update resource_config_id and resource_config_scope_id
		//
		// if value changes, request scheduling for downstream jobs
		scope, ok, err := resource.RefreshResourceConfig()
		if err != nil {
			// XXX: expect to get here if an ancestor type has no version?
			return fmt.Errorf("update resource config: %w", err)
		}

		if ok {
			// make sure we haven't already queued a check for the same scope
			if alreadyChecked[scope.ID()] {
				continue
			} else {
				alreadyChecked[scope.ID()] = true
			}
		} else {
			// scope cannot be created; parent type must not have a version yet.
			//
			// the plan we construct will have a check and get for the parent type's
			// image, and the resource's scope be set by the check step instead
		}

		// create a build plan with unevaluated creds
		//
		// check step will evaluate creds to come up with its own resource config
		// and scope
		//
		// check step will save versions to scope at runtime
		//
		// check step will update associated resource or resource type's
		// resource_config_id and resource_config_scope_id if needed (i.e. cred
		// rotation)
		plan, err := scheduler.Planner.Create(
			resource.StepConfig(),
			resource.NamedResources(), // NOTE: unevaluated creds
			resource.ResourceTypes(),  // NOTE: unevaluated creds
			[]db.BuildInput{},
		)
		if err != nil {
			// XXX: probably shouldn't bail on the whole thing
			return fmt.Errorf("create build plan: %w", err)
		}

		build, err := resource.CreateBuild()
		if err != nil {
			return fmt.Errorf("create build: %w", err)
		}

		started, err := build.Start(plan)
		if err != nil {
			return fmt.Errorf("start build: %w", err)
		}

		if !started {
			// XXX: is this possible?
		}
	}

	return nil
}
