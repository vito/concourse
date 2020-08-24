package db_test

import (
	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/atc/db"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("Job Factory", func() {
	var jobFactory db.JobFactory

	BeforeEach(func() {
		jobFactory = db.NewJobFactory(dbConn, lockFactory)
	})

	Context("when there are public and private pipelines", func() {
		var publicPipeline db.Pipeline

		BeforeEach(func() {
			otherTeam, err := teamFactory.CreateTeam(atc.Team{Name: "other-team"})
			Expect(err).NotTo(HaveOccurred())

			publicPipeline, _, err = otherTeam.SavePipeline("public-pipeline", atc.Config{
				Jobs: atc.JobConfigs{
					{
						Name: "public-pipeline-job-1",
						PlanSequence: []atc.Step{
							{
								Config: &atc.GetStep{
									Name: "some-resource",
								},
							},
							{
								Config: &atc.GetStep{
									Name: "some-other-resource",
								},
							},
							{
								Config: &atc.PutStep{
									Name: "some-resource",
								},
							},
						},
					},
					{
						Name: "public-pipeline-job-2",
						PlanSequence: []atc.Step{
							{
								Config: &atc.GetStep{
									Name:   "some-resource",
									Passed: []string{"public-pipeline-job-1"},
								},
							},
							{
								Config: &atc.GetStep{
									Name:   "some-other-resource",
									Passed: []string{"public-pipeline-job-1"},
								},
							},
							{
								Config: &atc.GetStep{
									Name:     "resource",
									Resource: "some-resource",
								},
							},
							{
								Config: &atc.PutStep{
									Name:     "resource",
									Resource: "some-resource",
								},
							},
							{
								Config: &atc.PutStep{
									Name: "some-resource",
								},
							},
						},
					},
					{
						Name: "public-pipeline-job-3",
						PlanSequence: []atc.Step{
							{
								Config: &atc.GetStep{
									Name:   "some-resource",
									Passed: []string{"public-pipeline-job-1", "public-pipeline-job-2"},
								},
							},
						},
					},
				},
				Resources: atc.ResourceConfigs{
					{
						Name: "some-resource",
						Type: "some-type",
					},
					{
						Name: "some-other-resource",
						Type: "some-type",
					},
				},
			}, db.ConfigVersion(0), false)
			Expect(err).ToNot(HaveOccurred())
			Expect(publicPipeline.Expose()).To(Succeed())

			_, _, err = otherTeam.SavePipeline("private-pipeline", atc.Config{
				Jobs: atc.JobConfigs{
					{
						Name: "private-pipeline-job",
						PlanSequence: []atc.Step{
							{
								Config: &atc.GetStep{
									Name: "some-resource",
								},
							},
							{
								Config: &atc.PutStep{
									Name: "some-resource",
								},
							},
						},
					},
				},
				Resources: atc.ResourceConfigs{
					{
						Name: "some-resource",
						Type: "some-type",
					},
				},
			}, db.ConfigVersion(0), false)
			Expect(err).ToNot(HaveOccurred())
		})

		Describe("VisibleJobs", func() {
			It("returns jobs in the provided teams and jobs in public pipelines", func() {
				visibleJobs, err := jobFactory.VisibleJobs([]string{"default-team"})
				Expect(err).ToNot(HaveOccurred())

				Expect(len(visibleJobs)).To(Equal(4))
				Expect(visibleJobs[0].Name).To(Equal("some-job"))
				Expect(visibleJobs[1].Name).To(Equal("public-pipeline-job-1"))
				Expect(visibleJobs[2].Name).To(Equal("public-pipeline-job-2"))
				Expect(visibleJobs[3].Name).To(Equal("public-pipeline-job-3"))

				Expect(visibleJobs[0].Inputs).To(BeNil())
				Expect(visibleJobs[1].Inputs).To(Equal([]atc.DashboardJobInput{
					atc.DashboardJobInput{
						Name:     "some-other-resource",
						Resource: "some-other-resource",
					},
					atc.DashboardJobInput{
						Name:     "some-resource",
						Resource: "some-resource",
					},
				}))
				Expect(visibleJobs[2].Inputs).To(Equal([]atc.DashboardJobInput{
					atc.DashboardJobInput{
						Name:     "resource",
						Resource: "some-resource",
					},
					atc.DashboardJobInput{
						Name:     "some-other-resource",
						Resource: "some-other-resource",
						Passed:   []string{"public-pipeline-job-1"},
					},
					atc.DashboardJobInput{
						Name:     "some-resource",
						Resource: "some-resource",
						Passed:   []string{"public-pipeline-job-1"},
					},
				}))
				Expect(visibleJobs[3].Inputs).To(Equal([]atc.DashboardJobInput{
					atc.DashboardJobInput{
						Name:     "some-resource",
						Resource: "some-resource",
						Passed:   []string{"public-pipeline-job-1", "public-pipeline-job-2"},
					},
				}))

				Expect(visibleJobs[0].Outputs).To(BeNil())
				Expect(visibleJobs[1].Outputs).To(Equal([]atc.JobOutput{
					atc.JobOutput{
						Name:     "some-resource",
						Resource: "some-resource",
					},
				}))
				Expect(visibleJobs[2].Outputs).To(Equal([]atc.JobOutput{
					atc.JobOutput{
						Name:     "resource",
						Resource: "some-resource",
					},
					atc.JobOutput{
						Name:     "some-resource",
						Resource: "some-resource",
					},
				}))
				Expect(visibleJobs[3].Outputs).To(BeNil())
			})

			It("returns next build, latest completed build, and transition build for each job", func() {
				job, found, err := defaultPipeline.Job("some-job")
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeTrue())

				transitionBuild, err := job.CreateBuild()
				Expect(err).ToNot(HaveOccurred())

				err = transitionBuild.Finish(db.BuildStatusSucceeded)
				Expect(err).ToNot(HaveOccurred())

				found, err = transitionBuild.Reload()
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeTrue())

				finishedBuild, err := job.CreateBuild()
				Expect(err).ToNot(HaveOccurred())

				err = finishedBuild.Finish(db.BuildStatusSucceeded)
				Expect(err).ToNot(HaveOccurred())

				found, err = finishedBuild.Reload()
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeTrue())

				nextBuild, err := job.CreateBuild()
				Expect(err).ToNot(HaveOccurred())

				visibleJobs, err := jobFactory.VisibleJobs([]string{"default-team"})
				Expect(err).ToNot(HaveOccurred())

				Expect(visibleJobs[0].Name).To(Equal("some-job"))
				Expect(visibleJobs[0].NextBuild.ID).To(Equal(nextBuild.ID()))
				Expect(visibleJobs[0].NextBuild.Name).To(Equal(nextBuild.Name()))
				Expect(visibleJobs[0].NextBuild.JobName).To(Equal(nextBuild.JobName()))
				Expect(visibleJobs[0].NextBuild.PipelineName).To(Equal(nextBuild.PipelineName()))
				Expect(visibleJobs[0].NextBuild.TeamName).To(Equal(nextBuild.TeamName()))
				Expect(visibleJobs[0].NextBuild.Status).To(Equal(string(nextBuild.Status())))
				Expect(visibleJobs[0].NextBuild.StartTime).To(Equal(nextBuild.StartTime()))
				Expect(visibleJobs[0].NextBuild.EndTime).To(Equal(nextBuild.EndTime()))

				Expect(visibleJobs[0].FinishedBuild.ID).To(Equal(finishedBuild.ID()))
				Expect(visibleJobs[0].FinishedBuild.Name).To(Equal(finishedBuild.Name()))
				Expect(visibleJobs[0].FinishedBuild.JobName).To(Equal(finishedBuild.JobName()))
				Expect(visibleJobs[0].FinishedBuild.PipelineName).To(Equal(finishedBuild.PipelineName()))
				Expect(visibleJobs[0].FinishedBuild.TeamName).To(Equal(finishedBuild.TeamName()))
				Expect(visibleJobs[0].FinishedBuild.Status).To(Equal(string(finishedBuild.Status())))
				Expect(visibleJobs[0].FinishedBuild.StartTime).To(Equal(finishedBuild.StartTime()))
				Expect(visibleJobs[0].FinishedBuild.EndTime).To(Equal(finishedBuild.EndTime()))

				Expect(visibleJobs[0].TransitionBuild.ID).To(Equal(transitionBuild.ID()))
				Expect(visibleJobs[0].TransitionBuild.Name).To(Equal(transitionBuild.Name()))
				Expect(visibleJobs[0].TransitionBuild.JobName).To(Equal(transitionBuild.JobName()))
				Expect(visibleJobs[0].TransitionBuild.PipelineName).To(Equal(transitionBuild.PipelineName()))
				Expect(visibleJobs[0].TransitionBuild.TeamName).To(Equal(transitionBuild.TeamName()))
				Expect(visibleJobs[0].TransitionBuild.Status).To(Equal(string(transitionBuild.Status())))
				Expect(visibleJobs[0].TransitionBuild.StartTime).To(Equal(transitionBuild.StartTime()))
				Expect(visibleJobs[0].TransitionBuild.EndTime).To(Equal(transitionBuild.EndTime()))
			})
		})

		Describe("AllActiveJobs", func() {
			It("return all private and public pipelines", func() {
				allJobs, err := jobFactory.AllActiveJobs()
				Expect(err).ToNot(HaveOccurred())

				Expect(len(allJobs)).To(Equal(5))
				Expect(allJobs[0].Name).To(Equal("some-job"))
				Expect(allJobs[1].Name).To(Equal("public-pipeline-job-1"))
				Expect(allJobs[2].Name).To(Equal("public-pipeline-job-2"))
				Expect(allJobs[3].Name).To(Equal("public-pipeline-job-3"))
				Expect(allJobs[4].Name).To(Equal("private-pipeline-job"))

				Expect(allJobs[0].Inputs).To(BeNil())
				Expect(allJobs[1].Inputs).To(Equal([]atc.DashboardJobInput{
					atc.DashboardJobInput{
						Name:     "some-other-resource",
						Resource: "some-other-resource",
					},
					atc.DashboardJobInput{
						Name:     "some-resource",
						Resource: "some-resource",
					},
				}))
				Expect(allJobs[2].Inputs).To(Equal([]atc.DashboardJobInput{
					atc.DashboardJobInput{
						Name:     "resource",
						Resource: "some-resource",
					},
					atc.DashboardJobInput{
						Name:     "some-other-resource",
						Resource: "some-other-resource",
						Passed:   []string{"public-pipeline-job-1"},
					},
					atc.DashboardJobInput{
						Name:     "some-resource",
						Resource: "some-resource",
						Passed:   []string{"public-pipeline-job-1"},
					},
				}))
				Expect(allJobs[3].Inputs).To(Equal([]atc.DashboardJobInput{
					atc.DashboardJobInput{
						Name:     "some-resource",
						Resource: "some-resource",
						Passed:   []string{"public-pipeline-job-1", "public-pipeline-job-2"},
					},
				}))
				Expect(allJobs[4].Inputs).To(Equal([]atc.DashboardJobInput{
					atc.DashboardJobInput{
						Name:     "some-resource",
						Resource: "some-resource",
					},
				}))

				Expect(allJobs[0].Outputs).To(BeNil())
				Expect(allJobs[1].Outputs).To(Equal([]atc.JobOutput{
					atc.JobOutput{
						Name:     "some-resource",
						Resource: "some-resource",
					},
				}))
				Expect(allJobs[2].Outputs).To(Equal([]atc.JobOutput{
					atc.JobOutput{
						Name:     "resource",
						Resource: "some-resource",
					},
					atc.JobOutput{
						Name:     "some-resource",
						Resource: "some-resource",
					},
				}))
				Expect(allJobs[3].Outputs).To(BeNil())
				Expect(allJobs[4].Outputs).To(Equal([]atc.JobOutput{
					atc.JobOutput{
						Name:     "some-resource",
						Resource: "some-resource",
					},
				}))
			})
		})
	})

	Describe("JobsToSchedule", func() {
		var (
			job1 db.Job
			job2 db.Job
			job3 db.Job
		)

		BeforeEach(func() {
			err := defaultPipeline.Destroy()
			Expect(err).ToNot(HaveOccurred())
		})

		Context("when the job has a requested schedule time later than the last scheduled", func() {
			BeforeEach(func() {
				pipeline1, _, err := defaultTeam.SavePipeline("fake-pipeline", atc.Config{
					Jobs: atc.JobConfigs{
						{Name: "job-name"},
					},
				}, db.ConfigVersion(1), false)
				Expect(err).ToNot(HaveOccurred())

				var found bool
				job1, found, err = pipeline1.Job("job-name")
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeTrue())

				err = job1.RequestSchedule()
				Expect(err).ToNot(HaveOccurred())
			})

			It("fetches that job", func() {
				jobs, err := jobFactory.JobsToSchedule()
				Expect(err).ToNot(HaveOccurred())
				Expect(len(jobs)).To(Equal(1))
				Expect(jobs[0].Name()).To(Equal(job1.Name()))
			})
		})

		Context("when the job has a requested schedule time earlier than the last scheduled", func() {
			BeforeEach(func() {
				pipeline1, _, err := defaultTeam.SavePipeline("fake-pipeline", atc.Config{
					Jobs: atc.JobConfigs{
						{Name: "job-name"},
					},
				}, db.ConfigVersion(1), false)
				Expect(err).ToNot(HaveOccurred())

				var found bool
				job1, found, err = pipeline1.Job("job-name")
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeTrue())

				_, err = dbConn.Exec("UPDATE jobs SET last_scheduled = now() WHERE id = $1;", job1.ID())
				Expect(err).ToNot(HaveOccurred())
			})

			It("does not fetch that job", func() {
				jobs, err := jobFactory.JobsToSchedule()
				Expect(err).ToNot(HaveOccurred())
				Expect(len(jobs)).To(Equal(0))
			})
		})

		Context("when the job has a requested schedule time is the same as the last scheduled", func() {
			BeforeEach(func() {
				pipeline1, _, err := defaultTeam.SavePipeline("fake-pipeline", atc.Config{
					Jobs: atc.JobConfigs{
						{Name: "job-name"},
					},
				}, db.ConfigVersion(1), false)
				Expect(err).ToNot(HaveOccurred())

				var found bool
				job1, found, err = pipeline1.Job("job-name")
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeTrue())

				err = job1.RequestSchedule()
				Expect(err).ToNot(HaveOccurred())

				found, err = job1.Reload()
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeTrue())

				err = job1.UpdateLastScheduled(job1.ScheduleRequestedTime())
				Expect(err).ToNot(HaveOccurred())
			})

			It("does not fetch that job", func() {
				jobs, err := jobFactory.JobsToSchedule()
				Expect(err).ToNot(HaveOccurred())
				Expect(len(jobs)).To(Equal(0))
			})
		})

		Context("when there are multiple jobs with different times", func() {
			BeforeEach(func() {
				pipeline1, _, err := defaultTeam.SavePipeline("fake-pipeline", atc.Config{
					Jobs: atc.JobConfigs{
						{Name: "job-name"},
					},
				}, db.ConfigVersion(1), false)
				Expect(err).ToNot(HaveOccurred())

				var found bool
				job1, found, err = pipeline1.Job("job-name")
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeTrue())

				err = job1.RequestSchedule()
				Expect(err).ToNot(HaveOccurred())

				team, err := teamFactory.CreateTeam(atc.Team{Name: "some-team"})
				Expect(err).ToNot(HaveOccurred())

				pipeline2, _, err := team.SavePipeline("fake-pipeline-two", atc.Config{
					Jobs: atc.JobConfigs{
						{Name: "job-fake"},
					},
				}, db.ConfigVersion(1), false)
				Expect(err).ToNot(HaveOccurred())

				job2, found, err = pipeline2.Job("job-fake")
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeTrue())

				pipeline3, _, err := team.SavePipeline("fake-pipeline-three", atc.Config{
					Jobs: atc.JobConfigs{
						{Name: "job-fake-two"},
					},
				}, db.ConfigVersion(1), false)
				Expect(err).ToNot(HaveOccurred())

				job3, found, err = pipeline3.Job("job-fake-two")
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeTrue())

				_, err = dbConn.Exec("UPDATE jobs SET last_scheduled = now() WHERE id = $1;", job2.ID())
				Expect(err).ToNot(HaveOccurred())

				err = job3.RequestSchedule()
				Expect(err).ToNot(HaveOccurred())
			})

			It("fetches the jobs that have a requested schedule time later than it's last scheduled", func() {
				jobs, err := jobFactory.JobsToSchedule()
				Expect(err).ToNot(HaveOccurred())
				Expect(len(jobs)).To(Equal(2))
				jobNames := []string{jobs[0].Name(), jobs[1].Name()}
				Expect(jobNames).To(ConsistOf(job1.Name(), job3.Name()))
			})
		})

		Context("when the job is paused but has a later schedule requested time", func() {
			BeforeEach(func() {
				pipeline1, _, err := defaultTeam.SavePipeline("fake-pipeline", atc.Config{
					Jobs: atc.JobConfigs{
						{Name: "job-name"},
					},
				}, db.ConfigVersion(1), false)
				Expect(err).ToNot(HaveOccurred())

				var found bool
				job1, found, err = pipeline1.Job("job-name")
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeTrue())

				err = job1.RequestSchedule()
				Expect(err).ToNot(HaveOccurred())

				err = job1.Pause()
				Expect(err).ToNot(HaveOccurred())
			})

			It("does not fetch that job", func() {
				jobs, err := jobFactory.JobsToSchedule()
				Expect(err).ToNot(HaveOccurred())
				Expect(len(jobs)).To(Equal(0))
			})
		})

		Context("when the job is inactive but has a later schedule requested time", func() {
			BeforeEach(func() {
				pipeline1, _, err := defaultTeam.SavePipeline("fake-pipeline", atc.Config{
					Jobs: atc.JobConfigs{
						{Name: "job-name"},
					},
				}, db.ConfigVersion(1), false)
				Expect(err).ToNot(HaveOccurred())

				var found bool
				job1, found, err = pipeline1.Job("job-name")
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeTrue())

				err = job1.RequestSchedule()
				Expect(err).ToNot(HaveOccurred())

				_, _, err = defaultTeam.SavePipeline("fake-pipeline", atc.Config{}, pipeline1.ConfigVersion(), false)
				Expect(err).ToNot(HaveOccurred())
			})

			It("does not fetch that job", func() {
				jobs, err := jobFactory.JobsToSchedule()
				Expect(err).ToNot(HaveOccurred())
				Expect(len(jobs)).To(Equal(0))
			})
		})

		Context("when the pipeline is paused but it's job has a later schedule requested time", func() {
			BeforeEach(func() {
				pipeline1, _, err := defaultTeam.SavePipeline("fake-pipeline", atc.Config{
					Jobs: atc.JobConfigs{
						{Name: "job-name"},
					},
				}, db.ConfigVersion(1), false)
				Expect(err).ToNot(HaveOccurred())

				var found bool
				job1, found, err = pipeline1.Job("job-name")
				Expect(err).ToNot(HaveOccurred())
				Expect(found).To(BeTrue())

				err = job1.RequestSchedule()
				Expect(err).ToNot(HaveOccurred())

				err = pipeline1.Pause()
				Expect(err).ToNot(HaveOccurred())
			})

			It("does not fetch that job", func() {
				jobs, err := jobFactory.JobsToSchedule()
				Expect(err).ToNot(HaveOccurred())
				Expect(len(jobs)).To(Equal(0))
			})
		})

		Describe("scheduler jobs resources", func() {
			Context("when the job needed to be schedule has no resources", func() {
				BeforeEach(func() {
					pipeline1, _, err := defaultTeam.SavePipeline("fake-pipeline", atc.Config{
						Jobs: atc.JobConfigs{
							{Name: "job-name"},
						},
					}, db.ConfigVersion(1), false)
					Expect(err).ToNot(HaveOccurred())

					var found bool
					job1, found, err = pipeline1.Job("job-name")
					Expect(err).ToNot(HaveOccurred())
					Expect(found).To(BeTrue())

					err = job1.RequestSchedule()
					Expect(err).ToNot(HaveOccurred())
				})

				It("fetches that job and no resources", func() {
					jobs, err := jobFactory.JobsToSchedule()
					Expect(err).ToNot(HaveOccurred())
					Expect(len(jobs)).To(Equal(1))
					Expect(jobs[0].Name()).To(Equal(job1.Name()))
					Expect(jobs[0].Resources).To(BeNil())
				})
			})

			Context("when the job needed to be schedule uses resources", func() {
				BeforeEach(func() {
					pipeline1, _, err := defaultTeam.SavePipeline("fake-pipeline", atc.Config{
						Jobs: atc.JobConfigs{
							{
								Name: "job-name",
								PlanSequence: []atc.Step{
									{
										Config: &atc.GetStep{
											Name: "some-resource",
										},
									},
								},
							},
						},

						Resources: atc.ResourceConfigs{
							{
								Name: "some-resource",
								Type: "some-type",
								Source: atc.Source{
									"some": "source",
								},
							},
							{
								Name: "unused-resource",
							},
						},
					}, db.ConfigVersion(1), false)
					Expect(err).ToNot(HaveOccurred())

					var found bool
					job1, found, err = pipeline1.Job("job-name")
					Expect(err).ToNot(HaveOccurred())
					Expect(found).To(BeTrue())

					err = job1.RequestSchedule()
					Expect(err).ToNot(HaveOccurred())
				})

				It("fetches that job and the used resource", func() {
					jobs, err := jobFactory.JobsToSchedule()
					Expect(err).ToNot(HaveOccurred())
					Expect(len(jobs)).To(Equal(1))
					Expect(jobs[0].Name()).To(Equal(job1.Name()))
					Expect(jobs[0].Resources).To(HaveLen(1))
					Expect(jobs[0].Resources).To(ConsistOf(
						db.NamedResource{
							Name:   "some-resource",
							Type:   "some-type",
							Source: atc.Source{"some": "source"},
						},
					))
				})
			})

			Context("when multiple jobs needed to be schedule uses resources", func() {
				BeforeEach(func() {
					pipeline1, _, err := defaultTeam.SavePipeline("fake-pipeline", atc.Config{
						Jobs: atc.JobConfigs{
							{
								Name: "job-1",
								PlanSequence: []atc.Step{
									{
										Config: &atc.GetStep{
											Name: "some-resource",
										},
									},
								},
							},
							{
								Name: "job-2",
								PlanSequence: []atc.Step{
									{
										Config: &atc.GetStep{
											Name: "some-resource",
										},
									},
									{
										Config: &atc.GetStep{
											Name: "other-resource",
										},
									},
								},
							},
						},

						Resources: atc.ResourceConfigs{
							{
								Name: "some-resource",
								Type: "some-type",
							},
							{
								Name: "other-resource",
								Type: "some-type",
							},
							{
								Name: "unused-resource",
								Type: "some-type",
							},
						},
					}, db.ConfigVersion(1), false)
					Expect(err).ToNot(HaveOccurred())

					pipeline2, _, err := defaultTeam.SavePipeline("fake-pipeline-2", atc.Config{
						Jobs: atc.JobConfigs{
							{
								Name: "job-3",
								PlanSequence: []atc.Step{
									{
										Config: &atc.GetStep{
											Name: "some-resource",
										},
									},
									{
										Config: &atc.GetStep{
											Name: "some-resource-2",
										},
									},
								},
							},
						},

						Resources: atc.ResourceConfigs{
							{
								Name: "some-resource",
								Type: "other-type",
							},
							{
								Name: "some-resource-2",
								Type: "other-type",
							},
						},
					}, db.ConfigVersion(1), false)
					Expect(err).ToNot(HaveOccurred())

					var found bool
					job1, found, err = pipeline1.Job("job-1")
					Expect(err).ToNot(HaveOccurred())
					Expect(found).To(BeTrue())

					job2, found, err = pipeline1.Job("job-2")
					Expect(err).ToNot(HaveOccurred())
					Expect(found).To(BeTrue())

					job3, found, err = pipeline2.Job("job-3")
					Expect(err).ToNot(HaveOccurred())
					Expect(found).To(BeTrue())

					err = job1.RequestSchedule()
					Expect(err).ToNot(HaveOccurred())

					err = job2.RequestSchedule()
					Expect(err).ToNot(HaveOccurred())

					err = job3.RequestSchedule()
					Expect(err).ToNot(HaveOccurred())
				})

				It("fetches that job and the used resource", func() {
					jobs, err := jobFactory.JobsToSchedule()
					Expect(err).ToNot(HaveOccurred())

					jobResources := make(map[string]db.NamedResources)
					for _, job := range jobs {
						jobResources[job.Name()] = job.Resources
					}

					Expect(jobResources).To(MatchAllKeys(Keys{
						job1.Name(): ConsistOf(
							db.NamedResource{
								Name: "some-resource",
								Type: "some-type",
							},
						),
						job2.Name(): ConsistOf(
							db.NamedResource{
								Name: "some-resource",
								Type: "some-type",
							},
							db.NamedResource{
								Name: "other-resource",
								Type: "some-type",
							}),
						job3.Name(): ConsistOf(
							db.NamedResource{
								Name: "some-resource",
								Type: "other-type",
							},
							db.NamedResource{
								Name: "some-resource-2",
								Type: "other-type",
							}),
					}))
				})
			})

			Context("when the job needed to be schedule uses resources as puts", func() {
				BeforeEach(func() {
					pipeline1, _, err := defaultTeam.SavePipeline("fake-pipeline", atc.Config{
						Jobs: atc.JobConfigs{
							{
								Name: "job-name",
								PlanSequence: []atc.Step{
									{
										Config: &atc.PutStep{
											Name: "some-resource",
										},
									},
								},
							},
						},

						Resources: atc.ResourceConfigs{
							{
								Name: "some-resource",
								Type: "some-type",
								Source: atc.Source{
									"some": "source",
								},
							},
							{
								Name: "unused-resource",
							},
						},
					}, db.ConfigVersion(1), false)
					Expect(err).ToNot(HaveOccurred())

					var found bool
					job1, found, err = pipeline1.Job("job-name")
					Expect(err).ToNot(HaveOccurred())
					Expect(found).To(BeTrue())

					err = job1.RequestSchedule()
					Expect(err).ToNot(HaveOccurred())
				})

				It("fetches that job and the used resource", func() {
					jobs, err := jobFactory.JobsToSchedule()
					Expect(err).ToNot(HaveOccurred())
					Expect(len(jobs)).To(Equal(1))
					Expect(jobs[0].Name()).To(Equal(job1.Name()))
					Expect(jobs[0].Resources).To(ConsistOf(
						db.NamedResource{
							Name:   "some-resource",
							Type:   "some-type",
							Source: atc.Source{"some": "source"},
						},
					))
				})
			})

			Context("when the job needed to be schedule uses the resource as a put and a get", func() {
				BeforeEach(func() {
					pipeline1, _, err := defaultTeam.SavePipeline("fake-pipeline", atc.Config{
						Jobs: atc.JobConfigs{
							{
								Name: "job-name",
								PlanSequence: []atc.Step{
									{
										Config: &atc.GetStep{
											Name: "some-resource",
										},
									},
									{
										Config: &atc.PutStep{
											Name: "some-resource",
										},
									},
								},
							},
						},

						Resources: atc.ResourceConfigs{
							{
								Name: "some-resource",
								Type: "some-type",
								Source: atc.Source{
									"some": "source",
								},
							},
							{
								Name: "unused-resource",
							},
						},
					}, db.ConfigVersion(1), false)
					Expect(err).ToNot(HaveOccurred())

					var found bool
					job1, found, err = pipeline1.Job("job-name")
					Expect(err).ToNot(HaveOccurred())
					Expect(found).To(BeTrue())

					err = job1.RequestSchedule()
					Expect(err).ToNot(HaveOccurred())
				})

				It("fetches that job and the used resource", func() {
					jobs, err := jobFactory.JobsToSchedule()
					Expect(err).ToNot(HaveOccurred())
					Expect(len(jobs)).To(Equal(1))
					Expect(jobs[0].Name()).To(Equal(job1.Name()))
					Expect(jobs[0].Resources).To(ConsistOf(
						db.NamedResource{
							Name:   "some-resource",
							Type:   "some-type",
							Source: atc.Source{"some": "source"},
						},
					))
				})
			})
		})

		Describe("schedule jobs resource types", func() {
			Context("when the pipeline for the job needed to be scheduled uses custom resource types", func() {
				BeforeEach(func() {
					pipeline1, _, err := defaultTeam.SavePipeline("fake-pipeline", atc.Config{
						Jobs: atc.JobConfigs{
							{Name: "job-name"},
						},
						ResourceTypes: atc.ResourceTypes{
							{
								Name: "some-type",
								Type: "other-type",
							},
						},
					}, db.ConfigVersion(1), false)
					Expect(err).ToNot(HaveOccurred())

					var found bool
					job1, found, err = pipeline1.Job("job-name")
					Expect(err).ToNot(HaveOccurred())
					Expect(found).To(BeTrue())

					err = job1.RequestSchedule()
					Expect(err).ToNot(HaveOccurred())
				})

				It("fetches that job and resource type", func() {
					jobs, err := jobFactory.JobsToSchedule()
					Expect(err).ToNot(HaveOccurred())
					Expect(len(jobs)).To(Equal(1))
					Expect(jobs[0].Name()).To(Equal(job1.Name()))
					Expect(jobs[0].ResourceTypes).To(ConsistOf(
						atc.VersionedResourceType{
							ResourceType: atc.ResourceType{
								Name: "some-type",
								Type: "other-type",
							},
						},
					))
				})
			})

			Context("when multiple job from different pipelines uses custom resource types", func() {
				BeforeEach(func() {
					pipeline1, _, err := defaultTeam.SavePipeline("fake-pipeline", atc.Config{
						Jobs: atc.JobConfigs{
							{Name: "job-1"},
							{Name: "job-2"},
						},
						ResourceTypes: atc.ResourceTypes{
							{
								Name: "some-type",
								Type: "other-type",
							},
						},
					}, db.ConfigVersion(1), false)
					Expect(err).ToNot(HaveOccurred())

					pipeline2, _, err := defaultTeam.SavePipeline("fake-pipeline-2", atc.Config{
						Jobs: atc.JobConfigs{
							{Name: "job-3"},
						},
						ResourceTypes: atc.ResourceTypes{
							{
								Name: "some-type-1",
								Type: "other-type-1",
							},
							{
								Name: "some-type-2",
								Type: "other-type-2",
							},
						},
					}, db.ConfigVersion(1), false)
					Expect(err).ToNot(HaveOccurred())

					var found bool
					job1, found, err = pipeline1.Job("job-1")
					Expect(err).ToNot(HaveOccurred())
					Expect(found).To(BeTrue())

					err = job1.RequestSchedule()
					Expect(err).ToNot(HaveOccurred())

					job2, found, err = pipeline1.Job("job-2")
					Expect(err).ToNot(HaveOccurred())
					Expect(found).To(BeTrue())

					err = job2.RequestSchedule()
					Expect(err).ToNot(HaveOccurred())

					job3, found, err = pipeline2.Job("job-3")
					Expect(err).ToNot(HaveOccurred())
					Expect(found).To(BeTrue())

					err = job3.RequestSchedule()
					Expect(err).ToNot(HaveOccurred())
				})

				It("fetches all jobs and resource types", func() {
					jobs, err := jobFactory.JobsToSchedule()
					Expect(err).ToNot(HaveOccurred())

					jobResourceTypes := make(map[string]atc.VersionedResourceTypes)
					for _, job := range jobs {
						jobResourceTypes[job.Name()] = job.ResourceTypes
					}

					Expect(jobResourceTypes).To(MatchAllKeys(Keys{
						job1.Name(): ConsistOf(
							atc.VersionedResourceType{
								ResourceType: atc.ResourceType{
									Name: "some-type",
									Type: "other-type",
								},
							},
						),
						job2.Name(): ConsistOf(
							atc.VersionedResourceType{
								ResourceType: atc.ResourceType{
									Name: "some-type",
									Type: "other-type",
								},
							},
						),
						job3.Name(): ConsistOf(
							atc.VersionedResourceType{
								ResourceType: atc.ResourceType{
									Name: "some-type-1",
									Type: "other-type-1",
								},
							},
							atc.VersionedResourceType{
								ResourceType: atc.ResourceType{
									Name: "some-type-2",
									Type: "other-type-2",
								},
							},
						),
					}))
				})
			})
		})
	})
})
