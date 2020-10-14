package exec_test

import (
	"context"
	"errors"
	"sync"

	"github.com/concourse/concourse/atc/exec"
	. "github.com/concourse/concourse/atc/exec"
	"github.com/concourse/concourse/atc/exec/build"
	"github.com/concourse/concourse/atc/exec/execfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Aggregate", func() {
	var (
		ctx    context.Context
		cancel func()

		fakeStepA *execfakes.FakeStep
		fakeStepB *execfakes.FakeStep

		repo  *build.Repository
		state *execfakes.FakeRunState

		step    Step
		stepOk  bool
		stepErr error
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())

		fakeStepA = new(execfakes.FakeStep)
		fakeStepB = new(execfakes.FakeStep)

		step = AggregateStep{
			fakeStepA,
			fakeStepB,
		}

		repo = build.NewRepository()
		state = new(execfakes.FakeRunState)
		state.ArtifactRepositoryReturns(repo)
	})

	AfterEach(func() {
		cancel()
	})

	JustBeforeEach(func() {
		stepOk, stepErr = step.Run(ctx, state)
	})

	It("succeeds", func() {
		Expect(stepErr).ToNot(HaveOccurred())
	})

	It("passes the artifact repo to all steps", func() {
		Expect(fakeStepA.RunCallCount()).To(Equal(1))
		_, repo := fakeStepA.RunArgsForCall(0)
		Expect(repo).To(Equal(repo))

		Expect(fakeStepB.RunCallCount()).To(Equal(1))
		_, repo = fakeStepB.RunArgsForCall(0)
		Expect(repo).To(Equal(repo))
	})

	Describe("executing each source", func() {
		BeforeEach(func() {
			wg := new(sync.WaitGroup)
			wg.Add(2)

			fakeStepA.RunStub = func(context.Context, RunState) (bool, error) {
				wg.Done()
				wg.Wait()
				return true, nil
			}

			fakeStepB.RunStub = func(context.Context, RunState) (bool, error) {
				wg.Done()
				wg.Wait()
				return true, nil
			}
		})

		It("happens concurrently", func() {
			Expect(fakeStepA.RunCallCount()).To(Equal(1))
			Expect(fakeStepB.RunCallCount()).To(Equal(1))
		})
	})

	Describe("canceling", func() {
		BeforeEach(func() {
			cancel()
		})

		It("cancels each substep", func() {
			ctx, _ := fakeStepA.RunArgsForCall(0)
			Expect(ctx.Err()).To(Equal(context.Canceled))
			ctx, _ = fakeStepB.RunArgsForCall(0)
			Expect(ctx.Err()).To(Equal(context.Canceled))
		})

		It("returns ctx.Err()", func() {
			Expect(stepErr).To(Equal(context.Canceled))
		})
	})

	Context("when sources fail", func() {
		disasterA := errors.New("nope A")
		disasterB := errors.New("nope B")

		BeforeEach(func() {
			fakeStepA.RunReturns(false, disasterA)
			fakeStepB.RunReturns(false, disasterB)
		})

		It("exits with an error including the original message", func() {
			Expect(stepErr.Error()).To(ContainSubstring("nope A"))
			Expect(stepErr.Error()).To(ContainSubstring("nope B"))
		})
	})

	Context("when all sources are successful", func() {
		BeforeEach(func() {
			fakeStepA.RunReturns(true, nil)
			fakeStepB.RunReturns(true, nil)
		})

		It("succeeds", func() {
			Expect(stepOk).To(BeTrue())
		})
	})

	Context("and some branches are not successful", func() {
		BeforeEach(func() {
			fakeStepA.RunReturns(true, nil)
			fakeStepB.RunReturns(false, nil)
		})

		It("fails", func() {
			Expect(stepOk).To(BeFalse())
		})
	})

	Context("when no branches indicate success", func() {
		BeforeEach(func() {
			fakeStepA.RunReturns(false, nil)
			fakeStepB.RunReturns(false, nil)
		})

		It("fails", func() {
			Expect(stepOk).To(BeFalse())
		})
	})

	Context("when there are no branches", func() {
		BeforeEach(func() {
			step = AggregateStep{}
		})

		It("returns true", func() {
			Expect(stepOk).To(BeTrue())
		})
	})

	Describe("Panic", func() {
		Context("when one step panic", func() {
			BeforeEach(func() {
				fakeStepB.RunStub = func(_ context.Context, _ exec.RunState) (bool, error) {
					panic("something terrible")
				}
			})

			It("recover from panic and yields false", func() {
				Expect(stepOk).To(BeFalse())
				Expect(stepErr.Error()).To(ContainSubstring("something terrible"))
			})
		})
	})
})
