package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"code.cloudfoundry.org/lager"
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

	SetCheckSetupError(error) error
}

//go:generate counterfeiter . CheckFactory

type CheckFactory interface {
	Check(int) (Check, bool, error)
	StartedChecks() ([]Check, error)
	CreateCheck(int, bool, atc.Plan, CheckMetadata, SpanContext) (Check, bool, error)
	TryCreateCheck(context.Context, Checkable, ResourceTypes, atc.Version, bool) (Check, bool, error)
	Resources() ([]Resource, error)
	ResourceTypes() ([]ResourceType, error)
	AcquireScanningLock(lager.Logger) (lock.Lock, bool, error)
	NotifyChecker() error
}

type checkFactory struct {
	conn        Conn
	lockFactory lock.LockFactory

	secrets             creds.Secrets
	varSourcePool       creds.VarSourcePool
	defaultCheckTimeout time.Duration
}

func NewCheckFactory(
	conn Conn,
	lockFactory lock.LockFactory,
	secrets creds.Secrets,
	varSourcePool creds.VarSourcePool,
	defaultCheckTimeout time.Duration,
) CheckFactory {
	return &checkFactory{
		conn:        conn,
		lockFactory: lockFactory,

		secrets:             secrets,
		varSourcePool:       varSourcePool,
		defaultCheckTimeout: defaultCheckTimeout,
	}
}

func (c *checkFactory) NotifyChecker() error {
	return c.conn.Bus().Notify(atc.ComponentLidarChecker)
}

func (c *checkFactory) AcquireScanningLock(
	logger lager.Logger,
) (lock.Lock, bool, error) {
	return c.lockFactory.Acquire(
		logger,
		lock.NewResourceScanningLockID(),
	)
}

func (c *checkFactory) Check(id int) (Check, bool, error) {
	check := newEmptyCheck(c.conn, c.lockFactory)
	row := checksQuery.
		Where(sq.Eq{"c.id": id}).
		RunWith(c.conn).
		QueryRow()

	err := scanCheck(check, row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}

	return check, true, nil
}

func (c *checkFactory) StartedChecks() ([]Check, error) {
	rows, err := checksQuery.
		Where(sq.Eq{"status": CheckStatusStarted}).
		OrderBy("c.manually_triggered DESC, c.id").
		RunWith(c.conn).
		Query()
	if err != nil {
		return nil, err
	}

	var checks []Check

	for rows.Next() {
		check := newEmptyCheck(c.conn, c.lockFactory)
		err := scanCheck(check, rows)
		if err != nil {
			return nil, err
		}

		checks = append(checks, check)
	}

	return checks, nil
}

func (c *checkFactory) TryCreateCheck(ctx context.Context, checkable Checkable, resourceTypes ResourceTypes, fromVersion atc.Version, manuallyTriggered bool) (Check, bool, error) {
	// XXX: removed
	return nil, false, errors.New("stubbed")

	// logger := lagerctx.FromContext(ctx)

	// var err error

	// parentType, found := resourceTypes.Parent(checkable)
	// if found {
	// 	if parentType.Version() == nil {
	// 		return nil, false, fmt.Errorf("resource type '%s' has no version", parentType.Name())
	// 	}
	// }

	// timeout := c.defaultCheckTimeout
	// if to := checkable.CheckTimeout(); to != "" {
	// 	timeout, err = time.ParseDuration(to)
	// 	if err != nil {
	// 		return nil, false, err
	// 	}
	// }

	// pp, found, err := checkable.Pipeline()
	// if err != nil {
	// 	return nil, false, fmt.Errorf("failed to reload pipeline: %s", err.Error())
	// }
	// if !found {
	// 	return nil, false, fmt.Errorf("pipeline not found")
	// }

	// varss, err := pp.Variables(logger, c.secrets, c.varSourcePool)
	// if err != nil {
	// 	return nil, false, err
	// }

	// source, err := creds.NewSource(varss, checkable.Source()).Evaluate()
	// if err != nil {
	// 	return nil, false, err
	// }

	// filteredTypes := resourceTypes.Filter(checkable).Deserialize()
	// versionedResourceTypes, err := creds.NewVersionedResourceTypes(varss, filteredTypes).Evaluate()
	// if err != nil {
	// 	return nil, false, err
	// }

	// This could have changed based on new variable interpolation so update it
	// resourceConfigScope, err := checkable.SetResourceConfig(source, versionedResourceTypes)
	// if err != nil {
	// 	return nil, false, err
	// }

	// if fromVersion == nil {
	// 	rcv, found, err := resourceConfigScope.LatestVersion()
	// 	if err != nil {
	// 		return nil, false, err
	// 	}

	// 	if found {
	// 		fromVersion = atc.Version(rcv.Version())
	// 	}
	// }

	// 	checkPlanConfig := atc.CheckPlan{
	// 		Name:        checkable.Name(),
	// 		Source:      checkable.Source(),
	// 		Tags:        checkable.Tags(),
	// 		Timeout:     timeout.String(),
	// 		FromVersion: fromVersion,

	// 		VersionedResourceTypes: filteredTypes,
	// 	}

	// 	// XXX
	// 	planFactory := atc.NewPlanFactory(0)

	// 	var plan atc.Plan
	// 	imagePlan, hasImage := versionedResourceTypes.FetchType(checkable.Type(), planFactory)
	// 	if hasImage {
	// 		checkPlanConfig.TypeFrom = &imagePlan.ID
	// 		plan = planFactory.NewPlan(atc.OnSuccessPlan{
	// 			Step: imagePlan,
	// 			Next: planFactory.NewPlan(checkPlanConfig),
	// 		})
	// 	} else {
	// 		checkPlanConfig.Type = checkable.Type()
	// 		plan = planFactory.NewPlan(checkPlanConfig)
	// 	}

	// 	meta := CheckMetadata{
	// 		TeamID:       checkable.TeamID(),
	// 		TeamName:     checkable.TeamName(),
	// 		PipelineName: checkable.PipelineName(),
	// 		PipelineID:   checkable.PipelineID(),
	// ResourceConfigID:   resourceConfigScope.ResourceConfig().ID(),
	// BaseResourceTypeID: resourceConfigScope.ResourceConfig().OriginBaseResourceType().ID,
	// }

	// check, created, err := c.CreateCheck(
	// 	resourceConfigScope.ID(),
	// 	manuallyTriggered,
	// 	plan,
	// 	meta,
	// 	NewSpanContext(ctx),
	// )
	// if err != nil {
	// 	return nil, false, err
	// }

	// return check, created, nil
}

func (c *checkFactory) CreateCheck(
	resourceConfigScopeID int,
	manuallyTriggered bool,
	plan atc.Plan,
	meta CheckMetadata,
	sc SpanContext,
) (Check, bool, error) {
	tx, err := c.conn.Begin()
	if err != nil {
		return nil, false, err
	}

	defer Rollback(tx)

	planPayload, err := json.Marshal(plan)
	if err != nil {
		return nil, false, err
	}

	es := c.conn.EncryptionStrategy()
	encryptedPayload, nonce, err := es.Encrypt(planPayload)
	if err != nil {
		return nil, false, err
	}

	metadata, err := json.Marshal(meta)
	if err != nil {
		return nil, false, err
	}

	spanContext, err := json.Marshal(sc)
	if err != nil {
		return nil, false, err
	}

	var id int
	var createTime time.Time
	err = psql.Insert("checks").
		Columns(
			"resource_config_scope_id",
			"schema",
			"status",
			"manually_triggered",
			"plan",
			"nonce",
			"metadata",
			"span_context",
		).
		Values(
			resourceConfigScopeID,
			schema,
			CheckStatusStarted,
			manuallyTriggered,
			encryptedPayload,
			nonce,
			metadata,
			spanContext,
		).
		Suffix(`
			ON CONFLICT DO NOTHING
			RETURNING id, create_time
		`).
		RunWith(tx).
		QueryRow().
		Scan(&id, &createTime)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, false, err
	}

	return &check{
		id:                    id,
		resourceConfigScopeID: resourceConfigScopeID,
		schema:                schema,
		status:                CheckStatusStarted,
		plan:                  plan,
		createTime:            createTime,
		metadata:              meta,

		pipelineRef: pipelineRef{
			conn:         c.conn,
			lockFactory:  c.lockFactory,
			pipelineID:   meta.PipelineID,
			pipelineName: meta.PipelineName,
		},

		spanContext: sc,
	}, true, err
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
