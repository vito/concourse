package builds

import (
	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/atc/db"
)

type Planner struct {
	planFactory atc.PlanFactory
}

func NewPlanner(planFactory atc.PlanFactory) Planner {
	return Planner{
		planFactory: planFactory,
	}
}

func (planner Planner) Create(
	planConfig atc.StepConfig,
	resources db.SchedulerResources, // TODO: move to atc.PlanResources
	resourceTypes atc.VersionedResourceTypes,
	inputs []db.BuildInput,
) (atc.Plan, error) {
	visitor := &planVisitor{
		planFactory: planner.planFactory,

		resources:     resources,
		resourceTypes: resourceTypes,
		inputs:        inputs,
	}

	err := planConfig.Visit(visitor)
	if err != nil {
		return atc.Plan{}, err
	}

	return visitor.plan, nil
}

type planVisitor struct {
	planFactory atc.PlanFactory

	resources     db.SchedulerResources
	resourceTypes atc.VersionedResourceTypes
	inputs        []db.BuildInput

	plan atc.Plan
}

func (visitor *planVisitor) VisitTask(step *atc.TaskStep) error {
	// the problem: we don't even know what the image is, and we don't know the
	// type, so we don't even know how many plans are needed (resource types
	// can be recursive)
	//
	// if there's a version for the resource type, use Get step directly
	//
	// if there is no version, create a Check and use Get with VersionFrom
	//
	// this is dope, because we just figure it out at plan time, and the Check
	// step will result in the versions being saved regardless
	//
	// check for resource type (which we can know, 'cause we have VRT here!!)
	// get for resource type image
	// check for image
	// get for image
	// ImageArtifactName
	visitor.plan = visitor.planFactory.NewPlan(atc.TaskPlan{
		Name:              step.Name,
		Privileged:        step.Privileged,
		Config:            step.Config,
		ConfigPath:        step.ConfigPath,
		Vars:              step.Vars,
		Tags:              step.Tags,
		Params:            step.Params,
		InputMapping:      step.InputMapping,
		OutputMapping:     step.OutputMapping,
		ImageArtifactName: step.ImageArtifactName,

		VersionedResourceTypes: visitor.resourceTypes,
	})

	return nil
}

func (visitor *planVisitor) VisitCheck(step *atc.CheckStep) error {
	resourceName := step.Resource
	if resourceName == "" {
		resourceName = step.Name
	}

	resource, found := visitor.resources.Lookup(resourceName)
	if !found {
		return UnknownResourceError{resourceName}
	}

	checkPlanConfig := atc.CheckPlan{
		Name: step.Name,

		Resource: resourceName,
		Source:   resource.Source,
		Tags:     step.Tags,

		VersionedResourceTypes: visitor.resourceTypes,
	}

	imagePlan, hasImagePlan := visitor.resourceTypes.FetchType(resource.Type, visitor.planFactory)
	if hasImagePlan {
		checkPlanConfig.TypeFrom = &imagePlan.ID

		visitor.plan = visitor.planFactory.NewPlan(atc.OnSuccessPlan{
			Step: imagePlan,
			Next: visitor.planFactory.NewPlan(checkPlanConfig),
		})
	} else {
		checkPlanConfig.Type = resource.Type

		visitor.plan = visitor.planFactory.NewPlan(checkPlanConfig)
	}

	return nil
}

func (visitor *planVisitor) VisitGet(step *atc.GetStep) error {
	resourceName := step.Resource
	if resourceName == "" {
		resourceName = step.Name
	}

	resource, found := visitor.resources.Lookup(resourceName)
	if !found {
		return UnknownResourceError{resourceName}
	}

	var version atc.Version
	for _, input := range visitor.inputs {
		if input.Name == step.Name {
			version = atc.Version(input.Version)
			break
		}
	}

	if version == nil {
		return VersionNotProvidedError{step.Name}
	}

	getPlanConfig := atc.GetPlan{
		Name: step.Name,

		Resource: resourceName,
		Source:   resource.Source,
		Params:   step.Params,
		Version:  &version,
		Tags:     step.Tags,

		VersionedResourceTypes: visitor.resourceTypes,
	}

	imagePlan, hasImagePlan := visitor.resourceTypes.FetchType(resource.Type, visitor.planFactory)
	if hasImagePlan {
		// XXX: TypeFrom instead?
		getPlanConfig.ImageArtifactName = "type:" + resource.Type

		visitor.plan = visitor.planFactory.NewPlan(atc.OnSuccessPlan{
			Step: imagePlan,
			Next: visitor.planFactory.NewPlan(getPlanConfig),
		})
	} else {
		getPlanConfig.Type = resource.Type

		visitor.plan = visitor.planFactory.NewPlan(getPlanConfig)
	}

	return nil
}

func (visitor *planVisitor) VisitPut(step *atc.PutStep) error {
	logicalName := step.Name

	resourceName := step.Resource
	if resourceName == "" {
		resourceName = logicalName
	}

	resource, found := visitor.resources.Lookup(resourceName)
	if !found {
		return UnknownResourceError{resourceName}
	}

	putPlanConfig := atc.PutPlan{
		Name:     logicalName,
		Resource: resourceName,
		Source:   resource.Source,
		Params:   step.Params,
		Tags:     step.Tags,
		Inputs:   step.Inputs,

		VersionedResourceTypes: visitor.resourceTypes,
	}

	var putPlan atc.Plan
	imagePlan, hasImagePlan := visitor.resourceTypes.FetchType(resource.Type, visitor.planFactory)
	if hasImagePlan {
		putPlanConfig.ImageArtifactName = "type:" + resource.Type

		putPlan = visitor.planFactory.NewPlan(atc.OnSuccessPlan{
			Step: imagePlan,
			Next: visitor.planFactory.NewPlan(putPlanConfig),
		})
	} else {
		putPlanConfig.Type = resource.Type

		putPlan = visitor.planFactory.NewPlan(putPlanConfig)
	}

	dependentGetPlan := visitor.planFactory.NewPlan(atc.GetPlan{
		Type:        resource.Type,
		Name:        logicalName,
		Resource:    resourceName,
		VersionFrom: &putPlan.ID,

		Params: step.GetParams,
		Tags:   step.Tags,
		Source: resource.Source,

		VersionedResourceTypes: visitor.resourceTypes,
	})

	visitor.plan = visitor.planFactory.NewPlan(atc.OnSuccessPlan{
		Step: putPlan,
		Next: dependentGetPlan,
	})

	return nil
}

func (visitor *planVisitor) VisitDo(step *atc.DoStep) error {
	do := atc.DoPlan{}

	for _, step := range step.Steps {
		err := step.Config.Visit(visitor)
		if err != nil {
			return err
		}

		do = append(do, visitor.plan)
	}

	visitor.plan = visitor.planFactory.NewPlan(do)

	return nil
}

func (visitor *planVisitor) VisitAggregate(step *atc.AggregateStep) error {
	do := atc.AggregatePlan{}

	for _, sub := range step.Steps {
		err := sub.Config.Visit(visitor)
		if err != nil {
			return err
		}

		do = append(do, visitor.plan)
	}

	visitor.plan = visitor.planFactory.NewPlan(do)

	return nil
}

func (visitor *planVisitor) VisitInParallel(step *atc.InParallelStep) error {
	var steps []atc.Plan

	for _, sub := range step.Config.Steps {
		err := sub.Config.Visit(visitor)
		if err != nil {
			return err
		}

		steps = append(steps, visitor.plan)
	}

	visitor.plan = visitor.planFactory.NewPlan(atc.InParallelPlan{
		Steps:    steps,
		Limit:    step.Config.Limit,
		FailFast: step.Config.FailFast,
	})

	return nil
}

func (visitor *planVisitor) VisitAcross(step *atc.AcrossStep) error {
	vars := make([]atc.AcrossVar, len(step.Vars))
	for i, v := range step.Vars {
		maxInFlight := 1
		if v.MaxInFlight != nil {
			maxInFlight = v.MaxInFlight.Limit
			if v.MaxInFlight.All {
				maxInFlight = len(v.Values)
			}
		}
		vars[i] = atc.AcrossVar{
			Var:         v.Var,
			Values:      v.Values,
			MaxInFlight: maxInFlight,
		}
	}

	acrossPlan := atc.AcrossPlan{
		Vars:        vars,
		Steps:       []atc.VarScopedPlan{},
		FailFast:    step.FailFast,
	}
	for _, vals := range cartesianProduct(step.Vars) {
		err := step.Step.Visit(visitor)
		if err != nil {
			return err
		}
		acrossPlan.Steps = append(acrossPlan.Steps, atc.VarScopedPlan{
			Step:  visitor.plan,
			Values: vals,
		})
	}

	visitor.plan = visitor.planFactory.NewPlan(acrossPlan)

	return nil
}

func cartesianProduct(vars []atc.AcrossVarConfig) [][]interface{} {
	if len(vars) == 0 {
		return make([][]interface{}, 1)
	}
	var product [][]interface{}
	subProduct := cartesianProduct(vars[:len(vars)-1])
	for _, vec := range subProduct {
		for _, val := range vars[len(vars)-1].Values {
			product = append(product, append(vec, val))
		}
	}
	return product
}

func (visitor *planVisitor) VisitSetPipeline(step *atc.SetPipelineStep) error {
	visitor.plan = visitor.planFactory.NewPlan(atc.SetPipelinePlan{
		Name:     step.Name,
		File:     step.File,
		Team:     step.Team,
		Vars:     step.Vars,
		VarFiles: step.VarFiles,
	})

	return nil
}

func (visitor *planVisitor) VisitLoadVar(step *atc.LoadVarStep) error {
	visitor.plan = visitor.planFactory.NewPlan(atc.LoadVarPlan{
		Name:   step.Name,
		File:   step.File,
		Format: step.Format,
		Reveal: step.Reveal,
	})

	return nil
}

func (visitor *planVisitor) VisitTry(step *atc.TryStep) error {
	err := step.Step.Config.Visit(visitor)
	if err != nil {
		return err
	}

	visitor.plan = visitor.planFactory.NewPlan(atc.TryPlan{
		Step: visitor.plan,
	})

	return nil
}

func (visitor *planVisitor) VisitTimeout(step *atc.TimeoutStep) error {
	err := step.Step.Visit(visitor)
	if err != nil {
		return err
	}

	visitor.plan = visitor.planFactory.NewPlan(atc.TimeoutPlan{
		Duration: step.Duration,
		Step:     visitor.plan,
	})

	return nil
}

func (visitor *planVisitor) VisitRetry(step *atc.RetryStep) error {
	retryStep := make(atc.RetryPlan, step.Attempts)

	for i := 0; i < step.Attempts; i++ {
		err := step.Step.Visit(visitor)
		if err != nil {
			return err
		}

		retryStep[i] = visitor.plan
	}

	visitor.plan = visitor.planFactory.NewPlan(retryStep)

	return nil
}

func (visitor *planVisitor) VisitOnSuccess(step *atc.OnSuccessStep) error {
	plan := atc.OnSuccessPlan{}

	err := step.Step.Visit(visitor)
	if err != nil {
		return err
	}

	plan.Step = visitor.plan

	err = step.Hook.Config.Visit(visitor)
	if err != nil {
		return err
	}

	plan.Next = visitor.plan

	visitor.plan = visitor.planFactory.NewPlan(plan)

	return nil
}

func (visitor *planVisitor) VisitOnFailure(step *atc.OnFailureStep) error {
	plan := atc.OnFailurePlan{}

	err := step.Step.Visit(visitor)
	if err != nil {
		return err
	}

	plan.Step = visitor.plan

	err = step.Hook.Config.Visit(visitor)
	if err != nil {
		return err
	}

	plan.Next = visitor.plan

	visitor.plan = visitor.planFactory.NewPlan(plan)

	return nil
}

func (visitor *planVisitor) VisitOnAbort(step *atc.OnAbortStep) error {
	plan := atc.OnAbortPlan{}

	err := step.Step.Visit(visitor)
	if err != nil {
		return err
	}

	plan.Step = visitor.plan

	err = step.Hook.Config.Visit(visitor)
	if err != nil {
		return err
	}

	plan.Next = visitor.plan

	visitor.plan = visitor.planFactory.NewPlan(plan)

	return nil
}

func (visitor *planVisitor) VisitOnError(step *atc.OnErrorStep) error {
	plan := atc.OnErrorPlan{}

	err := step.Step.Visit(visitor)
	if err != nil {
		return err
	}

	plan.Step = visitor.plan

	err = step.Hook.Config.Visit(visitor)
	if err != nil {
		return err
	}

	plan.Next = visitor.plan

	visitor.plan = visitor.planFactory.NewPlan(plan)

	return nil
}
func (visitor *planVisitor) VisitEnsure(step *atc.EnsureStep) error {
	plan := atc.EnsurePlan{}

	err := step.Step.Visit(visitor)
	if err != nil {
		return err
	}

	plan.Step = visitor.plan

	err = step.Hook.Config.Visit(visitor)
	if err != nil {
		return err
	}

	plan.Next = visitor.plan

	visitor.plan = visitor.planFactory.NewPlan(plan)

	return nil
}
