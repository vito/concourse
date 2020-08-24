package commands

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/fly/commands/internal/flaghelpers"
	"github.com/concourse/concourse/fly/rc"
	"github.com/concourse/concourse/fly/ui"
	"github.com/fatih/color"
)

type CheckResourceTypeCommand struct {
	ResourceType flaghelpers.ResourceFlag `short:"r" long:"resource-type" required:"true" value-name:"PIPELINE/RESOURCE-TYPE" description:"Name of a resource-type to check"`
	Version      *atc.Version             `short:"f" long:"from"                     value-name:"VERSION"           description:"Version of the resource type to check from, e.g. digest:sha256@..."`
	Async        bool                     `short:"a" long:"async"                    value-name:"ASYNC"             description:"Return the check without waiting for its result"`
	Shallow      bool                     `long:"shallow"                          value-name:"SHALLOW"         description:"Check the resource type itself only"`
}

func (command *CheckResourceTypeCommand) Execute(args []string) error {

	target, err := rc.LoadTarget(Fly.Target, Fly.Verbose)
	if err != nil {
		return err
	}

	err = target.Validate()
	if err != nil {
		return err
	}

	var version atc.Version
	if command.Version != nil {
		version = *command.Version
	}

	if !command.Shallow {
		err = command.checkParent(target)
		if err != nil {
			return err
		}
	}

	build, found, err := target.Team().CheckResourceType(command.ResourceType.PipelineName, command.ResourceType.ResourceName, version)
	if err != nil {
		return err
	}

	if !found {
		return fmt.Errorf("pipeline '%s' or resource-type '%s' not found\n", command.ResourceType.PipelineName, command.ResourceType.ResourceName)
	}

	var buildID = strconv.Itoa(build.ID)

	if !command.Async {
		for build.Status == "started" {
			time.Sleep(time.Second)

			build, found, err = target.Client().Build(buildID)
			if err != nil {
				return err
			}

			if !found {
				return fmt.Errorf("check '%s' not found\n", buildID)
			}
		}
	}

	table := ui.Table{
		Headers: ui.TableRow{
			{Contents: "id", Color: color.New(color.Bold)},
			{Contents: "name", Color: color.New(color.Bold)},
			{Contents: "status", Color: color.New(color.Bold)},
			// {Contents: "check_error", Color: color.New(color.Bold)},
		},
	}

	table.Data = append(table.Data, []ui.TableCell{
		{Contents: buildID},
		{Contents: command.ResourceType.ResourceName},
		{Contents: build.Status},
		// {Contents: build.CheckError}, // XXX(check-refactor): show output?
	})

	if err = table.Render(os.Stdout, Fly.PrintTableHeaders); err != nil {
		return err
	}

	if build.Status == "errored" {
		os.Exit(1)
	}

	return nil
}

func (command *CheckResourceTypeCommand) checkParent(target rc.Target) error {
	resourceTypes, found, err := target.Team().VersionedResourceTypes(command.ResourceType.PipelineName)
	if err != nil {
		return err
	}

	if !found {
		return fmt.Errorf("pipeline '%s' not found\n", command.ResourceType.PipelineName)
	}

	resourceType, found := resourceTypes.Lookup(command.ResourceType.ResourceName)
	if !found {
		return fmt.Errorf("resource type '%s' not found\n", command.ResourceType.ResourceName)
	}

	parentType, found := command.findParent(resourceType.ResourceType, resourceTypes)
	if !found {
		return nil
	}

	cmd := &CheckResourceTypeCommand{
		ResourceType: flaghelpers.ResourceFlag{
			ResourceName: parentType.Name,
			PipelineName: command.ResourceType.PipelineName,
		},
	}

	return cmd.Execute(nil)
}

func (command *CheckResourceTypeCommand) findParent(resourceType atc.ResourceType, resourceTypes atc.VersionedResourceTypes) (atc.VersionedResourceType, bool) {
	for _, t := range resourceTypes {
		if t.Name != resourceType.Name && t.Name == resourceType.Type {
			return t, true
		}
	}
	return atc.VersionedResourceType{}, false
}
