package integration_test

import (
	"net/http"
	"os/exec"

	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/fly/ui"
	"github.com/fatih/color"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("CheckResourceType", func() {
	var (
		flyCmd          *exec.Cmd
		build           atc.Build
		resourceTypes   atc.VersionedResourceTypes
		expectedHeaders ui.TableRow
	)

	BeforeEach(func() {
		build = atc.Build{
			ID:     123,
			Status: "started",
		}

		resourceTypes = atc.VersionedResourceTypes{{
			ResourceType: atc.ResourceType{
				Name: "myresource",
				Type: "myresourcetype",
			},
		}, {
			ResourceType: atc.ResourceType{
				Name: "myresourcetype",
				Type: "mybaseresourcetype",
			},
		}}

		expectedHeaders = ui.TableRow{
			{Contents: "id", Color: color.New(color.Bold)},
			{Contents: "name", Color: color.New(color.Bold)},
			{Contents: "status", Color: color.New(color.Bold)},
			{Contents: "check_error", Color: color.New(color.Bold)},
		}
	})

	Context("when version is specified", func() {
		BeforeEach(func() {
			expectedURL := "/api/v1/teams/main/pipelines/mypipeline/resource-types/myresource/check"
			atcServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", expectedURL),
					ghttp.VerifyJSON(`{"from":{"ref":"fake-ref"}}`),
					ghttp.RespondWithJSONEncoded(http.StatusOK, build),
				),
			)
		})

		It("sends check resource request to ATC", func() {
			Expect(func() {
				flyCmd = exec.Command(flyPath, "-t", targetName, "check-resource-type", "-r", "mypipeline/myresource", "-f", "ref:fake-ref", "--shallow", "-a")
				sess, err := gexec.Start(flyCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(sess).Should(gexec.Exit(0))

				Eventually(sess.Out).Should(PrintTable(ui.Table{
					Headers: expectedHeaders,
					Data: []ui.TableRow{
						{
							{Contents: "123"},
							{Contents: "myresource"},
							{Contents: "started"},
						},
					},
				}))

			}).To(Change(func() int {
				return len(atcServer.ReceivedRequests())
			}).By(2))
		})
	})

	Context("when version is omitted", func() {
		BeforeEach(func() {
			expectedURL := "/api/v1/teams/main/pipelines/mypipeline/resource-types/myresource/check"
			atcServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", expectedURL),
					ghttp.VerifyJSON(`{"from":null}`),
					ghttp.RespondWithJSONEncoded(http.StatusOK, build),
				),
			)
		})

		It("sends check resource request to ATC", func() {
			Expect(func() {
				flyCmd = exec.Command(flyPath, "-t", targetName, "check-resource-type", "-r", "mypipeline/myresource", "--shallow", "-a")
				sess, err := gexec.Start(flyCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(sess).Should(gexec.Exit(0))

				Eventually(sess.Out).Should(PrintTable(ui.Table{
					Headers: expectedHeaders,
					Data: []ui.TableRow{
						{
							{Contents: "123"},
							{Contents: "myresource"},
							{Contents: "started"},
						},
					},
				}))

			}).To(Change(func() int {
				return len(atcServer.ReceivedRequests())
			}).By(2))
		})
	})

	Context("when the check succeed", func() {
		BeforeEach(func() {
			expectedURL := "/api/v1/teams/main/pipelines/mypipeline/resource-types/myresource/check"
			atcServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", expectedURL),
					ghttp.VerifyJSON(`{"from":null}`),
					ghttp.RespondWithJSONEncoded(http.StatusOK, build),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/builds/123"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, atc.Build{
						ID:        123,
						Status:    "succeeded",
						StartTime: 100000000000,
						EndTime:   100000000000,
					}),
				),
			)
		})

		It("sends check resource request to ATC", func() {
			Expect(func() {
				flyCmd = exec.Command(flyPath, "-t", targetName, "check-resource-type", "-r", "mypipeline/myresource", "--shallow")
				sess, err := gexec.Start(flyCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(sess).Should(gexec.Exit(0))

				Eventually(sess.Out).Should(PrintTable(ui.Table{
					Headers: expectedHeaders,
					Data: []ui.TableRow{
						{
							{Contents: "123"},
							{Contents: "myresource"},
							{Contents: "succeeded"},
						},
					},
				}))

			}).To(Change(func() int {
				return len(atcServer.ReceivedRequests())
			}).By(3))
		})
	})

	Context("when the check fail", func() {
		BeforeEach(func() {
			expectedURL := "/api/v1/teams/main/pipelines/mypipeline/resource-types/myresource/check"
			atcServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", expectedURL),
					ghttp.VerifyJSON(`{"from":null}`),
					ghttp.RespondWithJSONEncoded(http.StatusOK, build),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/builds/123"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, atc.Build{
						ID:        123,
						Status:    "errored",
						StartTime: 100000000000,
						EndTime:   100000000000,
					}),
				),
			)
		})

		It("sends check resource request to ATC", func() {
			Expect(func() {
				flyCmd = exec.Command(flyPath, "-t", targetName, "check-resource-type", "-r", "mypipeline/myresource", "--shallow")
				sess, err := gexec.Start(flyCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(sess).Should(gexec.Exit(1))

				Eventually(sess.Out).Should(PrintTable(ui.Table{
					Headers: expectedHeaders,
					Data: []ui.TableRow{
						{
							{Contents: "123"},
							{Contents: "myresource"},
							{Contents: "errored"},
						},
					},
				}))

			}).To(Change(func() int {
				return len(atcServer.ReceivedRequests())
			}).By(3))
		})
	})

	Context("when recursive check succeeds", func() {
		BeforeEach(func() {
			atcServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/teams/main/pipelines/mypipeline/resource-types"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, resourceTypes),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/info"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, atc.Info{Version: atcVersion, WorkerVersion: workerVersion}),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/teams/main/pipelines/mypipeline/resource-types"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, resourceTypes),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/api/v1/teams/main/pipelines/mypipeline/resource-types/myresourcetype/check"),
					ghttp.VerifyJSON(`{"from":null}`),
					ghttp.RespondWithJSONEncoded(http.StatusOK, atc.Build{
						ID:     987,
						Status: "started",
					}),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/builds/987"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, atc.Build{
						ID:        987,
						Status:    "succeeded",
						StartTime: 100000000000,
						EndTime:   100000000000,
					}),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/api/v1/teams/main/pipelines/mypipeline/resource-types/myresource/check"),
					ghttp.VerifyJSON(`{"from":null}`),
					ghttp.RespondWithJSONEncoded(http.StatusOK, build),
				),
			)
		})

		It("sends check resource request to ATC", func() {
			Expect(func() {
				flyCmd = exec.Command(flyPath, "-t", targetName, "check-resource-type", "-r", "mypipeline/myresource", "-a")
				sess, err := gexec.Start(flyCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(sess).Should(gexec.Exit(0))

				Eventually(sess.Out).Should(PrintTable(ui.Table{
					Headers: expectedHeaders,
					Data: []ui.TableRow{
						{
							{Contents: "987"},
							{Contents: "myresourcetype"},
							{Contents: "succeeded"},
						},
					},
				}))

				Eventually(sess.Out).Should(PrintTable(ui.Table{
					Headers: expectedHeaders,
					Data: []ui.TableRow{
						{
							{Contents: "123"},
							{Contents: "myresource"},
							{Contents: "started"},
						},
					},
				}))

			}).To(Change(func() int {
				return len(atcServer.ReceivedRequests())
			}).By(7))
		})
	})

	Context("when recursive check fails", func() {
		BeforeEach(func() {
			atcServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/teams/main/pipelines/mypipeline/resource-types"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, resourceTypes),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/info"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, atc.Info{Version: atcVersion, WorkerVersion: workerVersion}),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/teams/main/pipelines/mypipeline/resource-types"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, resourceTypes),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/api/v1/teams/main/pipelines/mypipeline/resource-types/myresourcetype/check"),
					ghttp.VerifyJSON(`{"from":null}`),
					ghttp.RespondWithJSONEncoded(http.StatusOK, atc.Build{
						ID:     987,
						Status: "started",
					}),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/builds/987"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, atc.Build{
						ID:        987,
						Status:    "errored",
						StartTime: 100000000000,
						EndTime:   100000000000,
					}),
				),
			)
		})

		It("sends check resource request to ATC", func() {
			Expect(func() {
				flyCmd = exec.Command(flyPath, "-t", targetName, "check-resource-type", "-r", "mypipeline/myresource", "-a")
				sess, err := gexec.Start(flyCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())

				Eventually(sess).Should(gexec.Exit(1))

				Eventually(sess.Out).Should(PrintTable(ui.Table{
					Headers: expectedHeaders,
					Data: []ui.TableRow{
						{
							{Contents: "987"},
							{Contents: "myresourcetype"},
							{Contents: "errored"},
						},
					},
				}))

			}).To(Change(func() int {
				return len(atcServer.ReceivedRequests())
			}).By(6))
		})
	})

	Context("when pipeline or resource-type is not found", func() {
		BeforeEach(func() {
			expectedURL := "/api/v1/teams/main/pipelines/mypipeline/resource-types/myresource/check"
			atcServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", expectedURL),
					ghttp.RespondWithJSONEncoded(http.StatusNotFound, ""),
				),
			)
		})

		It("fails with error", func() {
			flyCmd = exec.Command(flyPath, "-t", targetName, "check-resource-type", "-r", "mypipeline/myresource", "--shallow")
			sess, err := gexec.Start(flyCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(sess).Should(gexec.Exit(1))

			Expect(sess.Err).To(gbytes.Say("pipeline 'mypipeline' or resource-type 'myresource' not found"))
		})
	})

	Context("When resource-type check returns internal server error", func() {
		BeforeEach(func() {
			expectedURL := "/api/v1/teams/main/pipelines/mypipeline/resource-types/myresource/check"
			atcServer.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", expectedURL),
					ghttp.RespondWith(http.StatusInternalServerError, "unknown server error"),
				),
			)
		})

		It("outputs error in response body", func() {
			flyCmd = exec.Command(flyPath, "-t", targetName, "check-resource-type", "-r", "mypipeline/myresource", "--shallow")
			sess, err := gexec.Start(flyCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(sess).Should(gexec.Exit(1))

			Expect(sess.Err).To(gbytes.Say("unknown server error"))
		})
	})
})
