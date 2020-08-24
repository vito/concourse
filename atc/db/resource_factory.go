package db

import (
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/atc/creds"
	"github.com/concourse/concourse/atc/db/lock"
	"github.com/concourse/concourse/vars"
)

//go:generate counterfeiter . ResourceFactory

type ResourceFactory interface {
	Resource(int) (Resource, bool, error)

	VisibleResources([]string) ([]Resource, error)
	AllResources() ([]Resource, error)

	ResourcesToSchedule(time.Duration, time.Duration) ([]SchedulerResource, error)
}

type NamedResources []NamedResource

type NamedResource struct {
	Name   string
	Type   string
	Source atc.Source
}

func (resources NamedResources) Lookup(name string) (NamedResource, bool) {
	for _, resource := range resources {
		if resource.Name == name {
			return resource, true
		}
	}

	return NamedResource{}, false
}

type resourceFactory struct {
	conn          Conn
	lockFactory   lock.LockFactory
	globalSecrets creds.Secrets
	varSourcePool creds.VarSourcePool
}

func NewResourceFactory(
	conn Conn,
	lockFactory lock.LockFactory,
	globalSecrets creds.Secrets,
	varSourcePool creds.VarSourcePool,
) ResourceFactory {
	return &resourceFactory{
		conn:          conn,
		lockFactory:   lockFactory,
		globalSecrets: globalSecrets,
		varSourcePool: varSourcePool,
	}
}

func (r *resourceFactory) Resource(resourceID int) (Resource, bool, error) {
	resource := newEmptyResource(r.conn, r.lockFactory)
	row := resourcesQuery.
		Where(sq.Eq{"r.id": resourceID}).
		RunWith(r.conn).
		QueryRow()

	err := scanResource(resource, row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}

	return resource, true, nil
}

func (r *resourceFactory) VisibleResources(teamNames []string) ([]Resource, error) {
	rows, err := resourcesQuery.
		Where(sq.Or{
			sq.Eq{"t.name": teamNames},
			sq.And{
				sq.NotEq{"t.name": teamNames},
				sq.Eq{"p.public": true},
			},
		}).
		OrderBy("r.id ASC").
		RunWith(r.conn).
		Query()
	if err != nil {
		return nil, err
	}

	return scanResources(rows, r.conn, r.lockFactory)
}

func (r *resourceFactory) AllResources() ([]Resource, error) {
	rows, err := resourcesQuery.
		OrderBy("r.id ASC").
		RunWith(r.conn).
		Query()
	if err != nil {
		return nil, err
	}

	return scanResources(rows, r.conn, r.lockFactory)
}

func (r *resourceFactory) ResourcesToSchedule(
	defaultInterval, defaultWebhookInterval time.Duration,
) ([]SchedulerResource, error) {
	rows, err := resourcesQuery.
		OrderBy("r.id ASC").
		Where(sq.Eq{"p.paused": false}).
		RunWith(r.conn).
		Query()
	if err != nil {
		return nil, err
	}

	resources, err := scanResources(rows, r.conn, r.lockFactory)
	if err != nil {
		return nil, err
	}

	pipelines := make(map[int]Pipeline)
	pipelineResourceTypes := make(map[Pipeline]ResourceTypes)
	pipelineVariables := make(map[Pipeline]vars.Variables)
	var srs []SchedulerResource
	for _, res := range resources {
		interval := defaultInterval
		if res.HasWebhook() {
			interval = defaultWebhookInterval
		}

		// TODO: it would be a lot nicer to push this down into the query; can we
		// add a check_every interval type column?
		if every := res.CheckEvery(); every != "" {
			interval, err = time.ParseDuration(every)
			if err != nil {
				return nil, err
			}
		}

		if time.Now().Before(res.LastCheckEndTime().Add(interval)) {
			// doesn't need to be checked yet
			continue
		}

		pipeline, found := pipelines[res.PipelineID()]
		if !found {
			pipeline, found, err = res.Pipeline()
			if err != nil {
				return nil, fmt.Errorf("get pipeline: %w", err)
			}
		}

		if !found {
			// pipeline removed?
			continue
		}

		rts, found := pipelineResourceTypes[pipeline]
		if !found {
			rts, err := pipeline.ResourceTypes()
			if err != nil {
				return nil, fmt.Errorf("get pipeline resource types: %w", err)
			}

			pipelineResourceTypes[pipeline] = rts
		}

		vars, found := pipelineVariables[pipeline]
		if !found {
			vars, err = pipeline.Variables(nil, r.globalSecrets, r.varSourcePool)
			if err != nil {
				return nil, fmt.Errorf("construct pipeline variables: %w", err)
			}
		}

		srs = append(srs, &schedulerResource{
			resource: res,
			source:   creds.NewSource(vars, res.Source()),
			vrts:     creds.NewVersionedResourceTypes(vars, rts.Filter(res).Deserialize()),
		})
	}

	return srs, nil
}

func scanResources(resourceRows *sql.Rows, conn Conn, lockFactory lock.LockFactory) ([]Resource, error) {
	var resources []Resource

	for resourceRows.Next() {
		resource := newEmptyResource(conn, lockFactory)
		err := scanResource(resource, resourceRows)
		if err != nil {
			return nil, err
		}

		resources = append(resources, resource)
	}

	return resources, nil
}
