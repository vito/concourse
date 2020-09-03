package db_test

import (
	"time"

	"code.cloudfoundry.org/clock/fakeclock"
	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/atc/db"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CheckableCounter", func() {
	var (
		fakeClock       *fakeclock.FakeClock
		refreshInterval time.Duration

		counter *db.CheckableCounter
	)

	BeforeEach(func() {
		fakeClock = fakeclock.NewFakeClock(time.Now())
		refreshInterval = time.Minute

		counter = db.NewCheckableCounter(dbConn, fakeClock, refreshInterval)
	})

	FIt("returns the number of scopes, but only refreshes after the interval elapses", func() {
		count, err := counter.CheckableCount()
		Expect(err).ToNot(HaveOccurred())
		Expect(count).To(Equal(0))

		config, err := resourceConfigFactory.FindOrCreateResourceConfig(
			defaultWorkerResourceType.Type,
			atc.Source{"some": "source"},
			atc.VersionedResourceTypes{},
		)
		Expect(err).ToNot(HaveOccurred())

		_, err = config.FindOrCreateScope(nil)
		Expect(err).ToNot(HaveOccurred())

		count, err = counter.CheckableCount()
		Expect(err).ToNot(HaveOccurred())
		Expect(count).To(Equal(0))

		fakeClock.Increment(refreshInterval)

		count, err = counter.CheckableCount()
		Expect(err).ToNot(HaveOccurred())
		Expect(count).To(Equal(1))

		for i := 0; i < 10; i++ {
			config, err := resourceConfigFactory.FindOrCreateResourceConfig(
				defaultWorkerResourceType.Type,
				atc.Source{"some": "source", "unique": i},
				atc.VersionedResourceTypes{},
			)
			Expect(err).ToNot(HaveOccurred())

			_, err = config.FindOrCreateScope(nil)
			Expect(err).ToNot(HaveOccurred())

			count, err = counter.CheckableCount()
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(1))
		}

		fakeClock.Increment(refreshInterval)

		count, err = counter.CheckableCount()
		Expect(err).ToNot(HaveOccurred())
		Expect(count).To(Equal(11))
	})
})
