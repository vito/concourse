package db

import (
	"context"
	"fmt"
	"time"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagerctx"
	sq "github.com/Masterminds/squirrel"
	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/atc/creds"
	"github.com/concourse/concourse/atc/db/lock"
)

//go:generate counterfeiter . Checkable

type Checkable interface {
	PipelineRef

	Name() string
	TeamID() int
	ResourceConfigScopeID() int
	TeamName() string
	Type() string
	Source() atc.Source
	Tags() atc.Tags
	CheckEvery() string
	CheckTimeout() string
	LastCheckEndTime() time.Time
	CurrentPinnedVersion() atc.Version

	HasWebhook() bool

	SetResourceConfig(
		atc.Source,
		atc.VersionedResourceTypes,
	) (ResourceConfigScope, error)

	CheckPlan(atc.Version, time.Duration, time.Duration, atc.VersionedResourceTypes) atc.CheckPlan

	CreateBuild(bool) (Build, bool, error)

	SetCheckSetupError(error) error
}

//go:generate counterfeiter . CheckFactory

type CheckFactory interface {
	TryCreateCheck(context.Context, Checkable, ResourceTypes, atc.Version, bool) (Build, bool, error)
	Resources() ([]Resource, error)
	ResourceTypes() ([]ResourceType, error)
	AcquireScanningLock(lager.Logger) (lock.Lock, bool, error)
	NotifyChecker() error
}

type checkFactory struct {
	conn        Conn
	lockFactory lock.LockFactory

	secrets       creds.Secrets
	varSourcePool creds.VarSourcePool

	defaultCheckTimeout             time.Duration
	defaultCheckInterval            time.Duration
	defaultWithWebhookCheckInterval time.Duration
}

func NewCheckFactory(
	conn Conn,
	lockFactory lock.LockFactory,
	secrets creds.Secrets,
	varSourcePool creds.VarSourcePool,
	defaultCheckTimeout time.Duration,
	defaultCheckInterval time.Duration,
	defaultWithWebhookCheckInterval time.Duration,
) CheckFactory {
	return &checkFactory{
		conn:        conn,
		lockFactory: lockFactory,

		secrets:       secrets,
		varSourcePool: varSourcePool,

		defaultCheckTimeout:             defaultCheckTimeout,
		defaultCheckInterval:            defaultCheckInterval,
		defaultWithWebhookCheckInterval: defaultWithWebhookCheckInterval,
	}
}

func (c *checkFactory) NotifyChecker() error {
	return c.conn.Bus().Notify(atc.ComponentLidarChecker)
}

func (c *checkFactory) AcquireScanningLock(logger lager.Logger) (lock.Lock, bool, error) {
	return c.lockFactory.Acquire(
		logger,
		lock.NewResourceScanningLockID(),
	)
}

func (c *checkFactory) TryCreateCheck(ctx context.Context, leaf Checkable, resourceTypes ResourceTypes, fromVersion atc.Version, manuallyTriggered bool) (Build, bool, error) {
	logger := lagerctx.FromContext(ctx)

	ancestry := resourceTypes.Filter(leaf)
	checkables := make([]Checkable, len(ancestry))
	for i, a := range ancestry {
		checkables[i] = a
	}

	checkables = append(checkables, leaf)

	pf := atc.NewPlanFactory(0) // XXX(check-refactor): what ID to use?
	var checkPlans []atc.Plan
	for _, checkable := range checkables {
		parentType, found := ancestry.Parent(checkable)
		if found {
			if parentType.Version() == nil {
				// XXX(check-refactor): this used to error - now we just stop
				break
			}
		}

		var err error

		interval := c.defaultCheckInterval
		if checkable.HasWebhook() {
			interval = c.defaultWithWebhookCheckInterval
		}
		if every := checkable.CheckEvery(); every != "" {
			interval, err = time.ParseDuration(every)
			if err != nil {
				return nil, false, fmt.Errorf("check interval: %s", err)
			}
		}

		if !manuallyTriggered && time.Now().Before(checkable.LastCheckEndTime().Add(interval)) {
			// skip creating the check if its interval hasn't elapsed yet
			continue
		}

		timeout := c.defaultCheckTimeout
		if to := checkable.CheckTimeout(); to != "" {
			timeout, err = time.ParseDuration(to)
			if err != nil {
				return nil, false, fmt.Errorf("check timeout: %s", err)
			}
		}

		filteredTypes := resourceTypes.Filter(checkable).Deserialize()

		checkPlans = append(
			checkPlans,
			pf.NewPlan(checkable.CheckPlan(fromVersion, interval, timeout, filteredTypes)),
		)
	}

	if len(checkPlans) == 0 {
		// no checks queued; no intervals elapsed?
		return nil, false, nil
	}

	checkPlan := pf.NewPlan(atc.InParallelPlan{
		Steps: checkPlans,
	})

	build, created, err := leaf.CreateBuild(manuallyTriggered)
	if err != nil {
		return nil, false, fmt.Errorf("create build: %w", err)
	}

	if !created {
		return nil, false, nil
	}

	started, err := build.Start(checkPlan)
	if err != nil {
		return nil, false, fmt.Errorf("start build: %w", err)
	}

	logger.Info("created-build", lager.Data{
		"plan":    checkPlan,
		"started": started,
	})

	_, err = build.Reload()
	if err != nil {
		return nil, false, fmt.Errorf("reload build: %w", err)
	}

	return build, true, nil
}

func (c *checkFactory) Resources() ([]Resource, error) {
	var resources []Resource

	rows, err := resourcesQuery.
		Where(sq.Eq{"p.paused": false}).
		RunWith(c.conn).
		Query()

	if err != nil {
		return nil, err
	}

	defer Close(rows)

	for rows.Next() {
		r := newEmptyResource(c.conn, c.lockFactory)
		err = scanResource(r, rows)
		if err != nil {
			return nil, err
		}

		resources = append(resources, r)
	}

	return resources, nil
}

func (c *checkFactory) ResourceTypes() ([]ResourceType, error) {
	var resourceTypes []ResourceType

	rows, err := resourceTypesQuery.
		RunWith(c.conn).
		Query()

	if err != nil {
		return nil, err
	}

	defer Close(rows)

	for rows.Next() {
		r := newEmptyResourceType(c.conn, c.lockFactory)
		err = scanResourceType(r, rows)
		if err != nil {
			return nil, err
		}

		resourceTypes = append(resourceTypes, r)
	}

	return resourceTypes, nil
}
