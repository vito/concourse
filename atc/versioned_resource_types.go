package atc

type VersionedResourceType struct {
	ResourceType

	Version Version `json:"version"`
}

type VersionedResourceTypes []VersionedResourceType

func (types VersionedResourceTypes) Lookup(name string) (VersionedResourceType, bool) {
	for _, t := range types {
		if t.Name == name {
			return t, true
		}
	}

	return VersionedResourceType{}, false
}

func (types VersionedResourceTypes) Without(name string) VersionedResourceTypes {
	newTypes := VersionedResourceTypes{}
	for _, t := range types {
		if t.Name != name {
			newTypes = append(newTypes, t)
		}
	}

	return newTypes
}

// XXX: how to integrate image policy checking? perform at plan time with
// unevaluated creds?
func (types VersionedResourceTypes) FetchType(
	name string,
	planFactory PlanFactory,
) (Plan, bool) {
	resourceType, found := types.Lookup(name)
	if !found {
		return Plan{}, false
	}

	getPlanConfig := GetPlan{
		Name:   "type:" + resourceType.Name,
		Source: resourceType.Source,
		Params: resourceType.Params,
	}

	parentType, hasParent := types.FetchType(resourceType.Type, planFactory)
	if hasParent {
		getPlanConfig.ImageArtifactName = "type:" + resourceType.Type
	} else {
		getPlanConfig.Type = resourceType.Type
	}

	// XXX: check plan will decide on its own whether it needs to run
	checkPlan := planFactory.NewPlan(CheckPlan{
		Name:        resourceType.Name,
		Source:      resourceType.Source,
		FromVersion: resourceType.Version,
	})

	getPlanConfig.VersionFrom = &checkPlan.ID

	getPlan := planFactory.NewPlan(OnSuccessPlan{
		Step: checkPlan,
		Next: planFactory.NewPlan(getPlanConfig),
	})

	if hasParent {
		getPlan = planFactory.NewPlan(OnSuccessPlan{
			Step: parentType,
			Next: getPlan,
		})
	}

	return getPlan, true
}
