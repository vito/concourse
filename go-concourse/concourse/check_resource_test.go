package concourse_test

import (
	"net/http"

	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/go-concourse/concourse"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("CheckResource", func() {
	Context("when ATC request succeeds", func() {
		var expectedCheck atc.Build

		BeforeEach(func() {
			expectedCheck = atc.Build{
				ID:        123,
				Status:    "started",
				StartTime: 100000000000,
				EndTime:   100000000000,
			}

			expectedURL := "/api/v1/teams/some-team/pipelines/mypipeline/resources/myresource/check"
			atcServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", expectedURL),
					ghttp.VerifyJSON(`{"from":{"ref":"fake-ref"}}`),
					ghttp.RespondWithJSONEncoded(http.StatusOK, expectedCheck),
				),
			)
		})

		It("sends check resource request to ATC", func() {
			check, found, err := team.CheckResource("mypipeline", "myresource", atc.Version{"ref": "fake-ref"})
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(check).To(Equal(expectedCheck))

			Expect(atcServer.ReceivedRequests()).To(HaveLen(1))
		})
	})

	Context("when pipeline or resource does not exist", func() {
		BeforeEach(func() {
			expectedURL := "/api/v1/teams/some-team/pipelines/mypipeline/resources/myresource/check"
			atcServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", expectedURL),
					ghttp.VerifyJSON(`{"from":{"ref":"fake-ref"}}`),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, ""),
				),
			)
		})

		It("returns a ResourceNotFoundError", func() {
			_, found, err := team.CheckResource("mypipeline", "myresource", atc.Version{"ref": "fake-ref"})
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeFalse())
		})
	})

	Context("when ATC responds with an error", func() {
		BeforeEach(func() {
			expectedURL := "/api/v1/teams/some-team/pipelines/mypipeline/resources/myresource/check"

			atcServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", expectedURL),
					ghttp.VerifyJSON(`{"from":{"ref":"fake-ref"}}`),
					ghttp.RespondWith(http.StatusBadRequest, "bad request"),
				),
			)
		})

		It("returns an error", func() {
			_, _, err := team.CheckResource("mypipeline", "myresource", atc.Version{"ref": "fake-ref"})
			Expect(err).To(HaveOccurred())

			Expect(err.Error()).To(ContainSubstring("bad request"))
		})
	})

	Context("when ATC responds with an internal server error", func() {
		BeforeEach(func() {
			expectedURL := "/api/v1/teams/some-team/pipelines/mypipeline/resources/myresource/check"

			atcServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", expectedURL),
					ghttp.VerifyJSON(`{"from":{"ref":"fake-ref"}}`),
					ghttp.RespondWith(http.StatusInternalServerError, "unknown server error"),
				),
			)
		})

		It("returns an error with body", func() {
			_, _, err := team.CheckResource("mypipeline", "myresource", atc.Version{"ref": "fake-ref"})
			Expect(err).To(HaveOccurred())

			cre, ok := err.(concourse.GenericError)
			Expect(ok).To(BeTrue())
			Expect(cre.Error()).To(Equal("unknown server error"))
		})
	})
})
