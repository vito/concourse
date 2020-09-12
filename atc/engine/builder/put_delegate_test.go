package builder_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/clock/fakeclock"
	"code.cloudfoundry.org/lager/lagertest"
	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/atc/db"
	"github.com/concourse/concourse/atc/db/dbfakes"
	"github.com/concourse/concourse/atc/engine/builder"
	"github.com/concourse/concourse/atc/event"
	"github.com/concourse/concourse/atc/exec"
	"github.com/concourse/concourse/atc/runtime"
	"github.com/concourse/concourse/vars"
)

var _ = Describe("PutDelegate", func() {
	var (
		logger    *lagertest.TestLogger
		fakeBuild *dbfakes.FakeBuild
		fakeClock *fakeclock.FakeClock
		buildVars *vars.BuildVariables

		now = time.Date(1991, 6, 3, 5, 30, 0, 0, time.UTC)

		delegate   exec.PutDelegate
		info       runtime.VersionResult
		exitStatus exec.ExitStatus
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")

		fakeBuild = new(dbfakes.FakeBuild)
		fakeClock = fakeclock.NewFakeClock(now)
		credVars := vars.StaticVariables{
			"source-param": "super-secret-source",
			"git-key":      "{\n123\n456\n789\n}\n",
		}
		buildVars = vars.NewBuildVariables(credVars, true)

		info = runtime.VersionResult{
			Version:  atc.Version{"foo": "bar"},
			Metadata: []atc.MetadataField{{Name: "baz", Value: "shmaz"}},
		}

		delegate = builder.NewPutDelegate(fakeBuild, "some-plan-id", buildVars, fakeClock)
	})

	Describe("Finished", func() {
		JustBeforeEach(func() {
			delegate.Finished(logger, exitStatus, info)
		})

		It("saves an event", func() {
			Expect(fakeBuild.SaveEventCallCount()).To(Equal(1))
			Expect(fakeBuild.SaveEventArgsForCall(0)).To(Equal(event.FinishPut{
				Origin:          event.Origin{ID: event.OriginID("some-plan-id")},
				Time:            now.Unix(),
				ExitStatus:      int(exitStatus),
				CreatedVersion:  info.Version,
				CreatedMetadata: info.Metadata,
			}))
		})
	})

	Describe("SaveOutput", func() {
		var plan atc.PutPlan
		var source atc.Source
		var resourceTypes atc.VersionedResourceTypes

		JustBeforeEach(func() {
			plan = atc.PutPlan{
				Name:     "some-name",
				Type:     "some-type",
				Resource: "some-resource",
			}
			source = atc.Source{"some": "source"}
			resourceTypes = atc.VersionedResourceTypes{}

			delegate.SaveOutput(logger, plan, source, resourceTypes, info)
		})

		It("saves the build output", func() {
			Expect(fakeBuild.SaveOutputCallCount()).To(Equal(1))
			resourceType, sourceArg, resourceTypesArg, version, metadata, name, resource := fakeBuild.SaveOutputArgsForCall(0)
			Expect(resourceType).To(Equal(plan.Type))
			Expect(sourceArg).To(Equal(source))
			Expect(resourceTypesArg).To(Equal(resourceTypes))
			Expect(version).To(Equal(info.Version))
			Expect(metadata).To(Equal(db.NewResourceConfigMetadataFields(info.Metadata)))
			Expect(name).To(Equal(plan.Name))
			Expect(resource).To(Equal(plan.Resource))
		})
	})
})
