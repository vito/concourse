package exec

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/hashicorp/go-multierror"
)

// AggregateStep is a step of steps to run in parallel.
type AggregateStep []Step

// Run executes all steps in parallel. It will indicate that it's ready when
// all of its steps are ready, and propagate any signal received to all running
// steps.
//
// It will wait for all steps to exit, even if one step fails or errors. After
// all steps finish, their errors (if any) will be aggregated and returned as a
// single error.
func (step AggregateStep) Run(ctx context.Context, state RunState) error {
	errs := make(chan error, len(step))

	for _, s := range step {
		s := s
		go func() {
			defer func() {
				if r := recover(); r != nil {
					err := fmt.Errorf("panic in aggregate step: %v", r)

					fmt.Fprintf(os.Stderr, "%s\n %s\n", err.Error(), string(debug.Stack()))
					errs <- err
				}
			}()

			errs <- s.Run(ctx, state)
		}()
	}

	var result error
	for i := 0; i < len(step); i++ {
		err := <-errs
		if err != nil {
			result = multierror.Append(result, err)
		}
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	if result != nil {
		return result
	}

	return nil
}

// Succeeded is true if all of the steps' Succeeded is true
func (step AggregateStep) Succeeded() bool {
	succeeded := true

	for _, step := range step {
		if !step.Succeeded() {
			succeeded = false
		}
	}

	return succeeded
}
