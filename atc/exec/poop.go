package exec

import (
	"context"
	"fmt"

	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/atc/runtime"
)

type TaskStep2 struct {
	plan atc.TaskPlan
}

func (step TaskStep2) Run(ctx context.Context, state RunState) error {
	artifacts := state.ArtifactRepository()
	config := step.fetchConfig()

	var image runtime.Artifact
	if step.plan.ImageArtifactName != "" {
		var found bool
		image, found = artifacts.ArtifactFor(step.plan.ImageArtifactName)
		if !found {
			return fmt.Errorf("artifact %s does not exist", step.plan.ImageArtifactName)
		}
	} else if config.ImageResource != nil {
		version := config.ImageResource.Version
		if version == nil {
			// execute the plan, giving it an ID constructed from the ID stored in
			// ctx + the given string
			//
			// this way the ID can be deterministic, allowing for idempotent runs
			err := state.Run(ctx, "image-check", &version, atc.CheckPlan{
				Type:   config.ImageResource.Type,
				Source: config.ImageResource.Source,
			})
		}

		err = state.Run(ctx, "image-get", &image, atc.GetPlan{
			Type:    config.ImageResource.Type,
			Source:  config.ImageResource.Source,
			Version: &version, // XXX(exec2): this really shouldn't be a pointer
			Params:  config.ImageResource.Params,
		})
	}
}

func (step TaskStep2) fetchConfig() atc.TaskConfig {
	return atc.TaskConfig{}
}
