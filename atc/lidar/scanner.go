package lidar

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"strconv"
	"sync"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagerctx"
	"github.com/concourse/concourse/atc/creds"
	"github.com/concourse/concourse/atc/db"
	"github.com/concourse/concourse/atc/metric"
	"github.com/concourse/concourse/tracing"
)

func NewScanner(
	logger lager.Logger,
	checkFactory db.CheckFactory,
	secrets creds.Secrets,
) *scanner {
	return &scanner{
		logger:       logger,
		checkFactory: checkFactory,
		secrets:      secrets,
	}
}

type scanner struct {
	logger lager.Logger

	checkFactory db.CheckFactory
	secrets      creds.Secrets
}

func (s *scanner) Run(ctx context.Context) error {
	spanCtx, span := tracing.StartSpan(ctx, "scanner.Run", nil)
	s.logger.Info("start")
	defer span.End()
	defer s.logger.Info("end")

	resources, err := s.checkFactory.Resources()
	if err != nil {
		s.logger.Error("failed-to-get-resources", err)
		return err
	}

	resourceTypes, err := s.checkFactory.ResourceTypes()
	if err != nil {
		s.logger.Error("failed-to-get-resource-types", err)
		return err
	}

	waitGroup := new(sync.WaitGroup)
	resourceTypesChecked := &sync.Map{}

	for _, resource := range resources {
		waitGroup.Add(1)

		go func(resource db.Resource, resourceTypes db.ResourceTypes) {
			loggerData := lager.Data{
				"resource_id":   strconv.Itoa(resource.ID()),
				"resource_name": resource.Name(),
				"pipeline_name": resource.PipelineName(),
				"team_name":     resource.TeamName(),
			}
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("panic in scanner run %s: %v", loggerData, r)

					fmt.Fprintf(os.Stderr, "%s\n %s\n", err.Error(), string(debug.Stack()))
					s.logger.Error("panic-in-scanner-run", err)

					s.setCheckError(s.logger, resource, err)
				}
			}()
			defer waitGroup.Done()

			err := s.check(spanCtx, resource, resourceTypes, resourceTypesChecked)
			s.setCheckError(s.logger, resource, err)

		}(resource, resourceTypes)
	}

	waitGroup.Wait()

	return s.checkFactory.NotifyChecker()
}

func (s *scanner) check(ctx context.Context, resource db.Resource, resourceTypes db.ResourceTypes, resourceTypesChecked *sync.Map) error {

	var err error

	spanCtx, span := tracing.StartSpan(ctx, "scanner.check", tracing.Attrs{
		"team":                     resource.TeamName(),
		"pipeline":                 resource.PipelineName(),
		"resource":                 resource.Name(),
		"type":                     resource.Type(),
		"resource_config_scope_id": strconv.Itoa(resource.ResourceConfigScopeID()),
	})
	defer span.End()

	// XXX(check-refactor): don't forget this: check from pinned version if set
	version := resource.CurrentPinnedVersion()

	_, created, err := s.checkFactory.TryCreateCheck(lagerctx.NewContext(spanCtx, s.logger), resource, resourceTypes, version, false)
	if err != nil {
		s.logger.Error("failed-to-create-check", err)
		return err
	}

	if !created {
		s.logger.Debug("check-already-exists")
	}

	metric.Metrics.ChecksEnqueued.Inc()

	return nil
}

func (s *scanner) setCheckError(logger lager.Logger, checkable db.Checkable, err error) {
	setErr := checkable.SetCheckSetupError(err)
	if setErr != nil {
		logger.Error("failed-to-set-check-error", setErr)
	}
}
