package exec_test

import (
	"context"
	"errors"
	"time"

	. "github.com/concourse/concourse/atc/exec"
	"github.com/concourse/concourse/atc/exec/build"
	"github.com/concourse/concourse/atc/exec/execfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Timeout Step", func() {
	var (
		ctx    context.Context
		cancel func()

		fakeStep *execfakes.FakeStep

		repo  *build.Repository
		state *execfakes.FakeRunState

		step Step

		timeoutDuration string

		stepOk  bool
		stepErr error
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())

		fakeStep = new(execfakes.FakeStep)

		repo = build.NewRepository()
		state = new(execfakes.FakeRunState)
		state.ArtifactRepositoryReturns(repo)

		timeoutDuration = "1h"
	})

	JustBeforeEach(func() {
		step = Timeout(fakeStep, timeoutDuration)
		stepOk, stepErr = step.Run(ctx, state)
	})

	Context("when the duration is valid", func() {
		It("runs the step with a deadline", func() {
			runCtx, _ := fakeStep.RunArgsForCall(0)
			deadline, ok := runCtx.Deadline()
			Expect(ok).To(BeTrue())
			Expect(deadline).To(BeTemporally("~", time.Now().Add(time.Hour), 10*time.Second))
		})

		Context("when the step returns an error", func() {
			var someError error

			BeforeEach(func() {
				someError = errors.New("some error")
				fakeStep.RunReturns(false, someError)
			})

			It("returns the error", func() {
				Expect(stepErr).NotTo(BeNil())
				Expect(stepErr).To(Equal(someError))
			})
		})

		Context("when the step exceeds the timeout", func() {
			BeforeEach(func() {
				fakeStep.RunReturns(true, context.DeadlineExceeded)
			})

			It("returns no error", func() {
				Expect(stepErr).ToNot(HaveOccurred())
			})

			It("is not successful", func() {
				Expect(stepOk).To(BeFalse())
			})
		})

		Describe("canceling", func() {
			BeforeEach(func() {
				cancel()
			})

			It("forwards the context down", func() {
				runCtx, _ := fakeStep.RunArgsForCall(0)
				Expect(runCtx.Err()).To(Equal(context.Canceled))
			})

			It("is not successful", func() {
				Expect(stepOk).To(BeFalse())
			})
		})

		Context("when the step is successful", func() {
			BeforeEach(func() {
				fakeStep.RunReturns(true, nil)
			})

			It("is successful", func() {
				Expect(stepOk).To(BeTrue())
			})
		})

		Context("when the step fails", func() {
			BeforeEach(func() {
				fakeStep.RunReturns(false, nil)
			})

			It("is not successful", func() {
				Expect(stepOk).To(BeFalse())
			})
		})
	})

	Context("when the duration is invalid", func() {
		BeforeEach(func() {
			timeoutDuration = "nope"
		})

		It("errors immediately without running the step", func() {
			Expect(stepErr).To(HaveOccurred())
			Expect(fakeStep.RunCallCount()).To(BeZero())
		})
	})
})
