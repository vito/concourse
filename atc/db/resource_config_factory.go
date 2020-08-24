package db

import (
	"database/sql"
	"fmt"
	"strconv"

	sq "github.com/Masterminds/squirrel"
	"github.com/concourse/concourse/atc"
	"github.com/concourse/concourse/atc/db/lock"
	"github.com/lib/pq"
)

type ErrCustomResourceTypeVersionNotFound struct {
	Name string
}

func (e ErrCustomResourceTypeVersionNotFound) Error() string {
	return fmt.Sprintf("custom resource type '%s' version not found", e.Name)
}

//go:generate counterfeiter . ResourceConfigFactory

type ResourceConfigFactory interface {
	FindOrCreateResourceConfigFromBaseType(
		baseType string,
		source atc.Source,
	) (ResourceConfig, error)

	FindOrCreateResourceConfigFromResourceCache(
		resourceCache UsedResourceCache,
		source atc.Source,
	) (ResourceConfig, error)

	FindOrCreateResourceConfig(
		resourceType string,
		source atc.Source,
		resourceTypes atc.VersionedResourceTypes,
	) (ResourceConfig, error)

	FindResourceConfigByID(int) (ResourceConfig, bool, error)

	CleanUnreferencedConfigs() error
}

type resourceConfigFactory struct {
	conn        Conn
	lockFactory lock.LockFactory
}

func NewResourceConfigFactory(conn Conn, lockFactory lock.LockFactory) ResourceConfigFactory {
	return &resourceConfigFactory{
		conn:        conn,
		lockFactory: lockFactory,
	}
}

func (f *resourceConfigFactory) FindResourceConfigByID(resourceConfigID int) (ResourceConfig, bool, error) {
	tx, err := f.conn.Begin()
	if err != nil {
		return nil, false, err
	}
	defer Rollback(tx)

	resourceConfig, found, err := findResourceConfigByID(tx, resourceConfigID, f.lockFactory, f.conn)
	if err != nil {
		return nil, false, err
	}

	if !found {
		return nil, false, nil
	}

	err = tx.Commit()
	if err != nil {
		return nil, false, err
	}

	return resourceConfig, true, nil
}

func (f *resourceConfigFactory) FindOrCreateResourceConfigFromBaseType(
	baseType string,
	source atc.Source,
) (ResourceConfig, error) {
	rc := &resourceConfig{
		lockFactory: f.lockFactory,
		conn:        f.conn,
	}

	tx, err := f.conn.Begin()
	if err != nil {
		return nil, err
	}
	defer Rollback(tx)

	ubrt, found, err := BaseResourceType{Name: baseType}.Find(tx)
	if err != nil {
		return nil, err
	}

	if !found {
		return nil, BaseResourceTypeNotFoundError{Name: baseType}
	}

	rc.createdByBaseResourceType = ubrt

	id, err := f.findOrCreate(tx, "base_resource_type_id", ubrt.ID, source)
	if err != nil {
		return nil, err
	}

	rc.id = id

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return rc, nil
}

func (f *resourceConfigFactory) FindOrCreateResourceConfigFromResourceCache(
	cache UsedResourceCache,
	source atc.Source,
) (ResourceConfig, error) {
	rc := &resourceConfig{
		lockFactory: f.lockFactory,
		conn:        f.conn,
	}

	tx, err := f.conn.Begin()
	if err != nil {
		return nil, err
	}
	defer Rollback(tx)

	rc.createdByResourceCache = cache

	id, err := f.findOrCreate(tx, "resource_cache", cache.ID(), source)
	if err != nil {
		return nil, err
	}

	rc.id = id

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return rc, nil
}

func (f *resourceConfigFactory) findOrCreate(tx Tx, parentColumnName string, parentID int, source atc.Source) (int, error) {
	sourceHash := mapHash(source)

	var id int

	err := psql.Select("id").
		From("resource_configs").
		Where(sq.Eq{
			parentColumnName: parentID,
			"source_hash":    sourceHash,
		}).
		Suffix("FOR SHARE").
		RunWith(tx).
		QueryRow().
		Scan(&id)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}

	if id == 0 {
		err := psql.Insert("resource_configs").
			Columns(
				parentColumnName,
				"source_hash",
			).
			Values(
				parentID,
				sourceHash,
			).
			Suffix(`
				ON CONFLICT (`+parentColumnName+`, source_hash) DO UPDATE SET
					`+parentColumnName+` = ?,
					source_hash = ?
				RETURNING id
			`, parentID, sourceHash).
			RunWith(tx).
			QueryRow().
			Scan(&id)
		if err != nil {
			return 0, err
		}
	}

	return id, nil
}

func (f *resourceConfigFactory) FindOrCreateResourceConfig(
	resourceType string,
	source atc.Source,
	resourceTypes atc.VersionedResourceTypes,
) (ResourceConfig, error) {

	resourceConfigDescriptor, err := constructResourceConfigDescriptor(resourceType, source, resourceTypes)
	if err != nil {
		return nil, err
	}

	tx, err := f.conn.Begin()
	if err != nil {
		return nil, err
	}
	defer Rollback(tx)

	resourceConfig, err := resourceConfigDescriptor.findOrCreate(tx, f.lockFactory, f.conn)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return resourceConfig, nil
}

type ParentTypeHasNoVersionError struct {
	Type string
}

func (err ParentTypeHasNoVersionError) Error() string {
	return fmt.Sprintf("parent type '%s' does not have a version yet", err.Type)
}

// constructResourceConfig cannot be called for constructing a resource type's
// resource config while also containing the same resource type in the list of
// resource types, because that results in a circular dependency.
//
// XXX: dealing with a missing version in the resource types is awkward; can we
// reduce the number of call sites for this? e.g. Build.SaveOutput
func constructResourceConfigDescriptor(
	resourceTypeName string,
	source atc.Source,
	resourceTypes atc.VersionedResourceTypes,
) (ResourceConfigDescriptor, error) {
	resourceConfigDescriptor := ResourceConfigDescriptor{
		Source: source,
	}

	customType, found := resourceTypes.Lookup(resourceTypeName)
	if found {
		// XXX: prevent this from constructing if version is null
		//
		// seems like nothing else checks for that. where did that used to happen?
		if customType.Version == nil {
			return ResourceConfigDescriptor{}, ParentTypeHasNoVersionError{
				Type: customType.Name,
			}
		}

		customTypeResourceConfig, err := constructResourceConfigDescriptor(
			customType.Type,
			customType.Source,
			resourceTypes.Without(customType.Name),
		)
		if err != nil {
			return ResourceConfigDescriptor{}, err
		}

		resourceConfigDescriptor.CreatedByResourceCache = &ResourceCacheDescriptor{
			ResourceConfigDescriptor: customTypeResourceConfig,
			Version:                  customType.Version,
		}
	} else {
		resourceConfigDescriptor.CreatedByBaseResourceType = &BaseResourceType{
			Name: resourceTypeName,
		}
	}

	return resourceConfigDescriptor, nil
}

func (f *resourceConfigFactory) CleanUnreferencedConfigs() error {
	usedByResourceConfigCheckSessionIds, _, err := sq.
		Select("resource_config_id").
		From("resource_config_check_sessions").
		ToSql()
	if err != nil {
		return err
	}

	usedByResourceCachesIds, _, err := sq.
		Select("resource_config_id").
		From("resource_caches").
		ToSql()
	if err != nil {
		return err
	}

	usedByResourceIds, _, err := sq.
		Select("resource_config_id").
		From("resources").
		Where("resource_config_id IS NOT NULL").
		ToSql()
	if err != nil {
		return err
	}

	usedByResourceTypesIds, _, err := sq.
		Select("resource_config_id").
		From("resource_types").
		Where("resource_config_id IS NOT NULL").
		ToSql()
	if err != nil {
		return err
	}

	_, err = psql.Delete("resource_configs").
		Where("id NOT IN (" + usedByResourceConfigCheckSessionIds + " UNION " + usedByResourceCachesIds + " UNION " + usedByResourceIds + " UNION " + usedByResourceTypesIds + ")").
		PlaceholderFormat(sq.Dollar).
		RunWith(f.conn).Exec()
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code.Name() == pqFKeyViolationErrCode {
			// this can happen if a use or resource cache is created referencing the
			// config; as the subqueries above are not atomic
			return nil
		}

		return err
	}

	return nil
}

func findResourceConfigByID(tx Tx, resourceConfigID int, lockFactory lock.LockFactory, conn Conn) (ResourceConfig, bool, error) {
	var brtIDString, cacheIDString sql.NullString

	err := psql.Select("base_resource_type_id", "resource_cache_id").
		From("resource_configs").
		Where(sq.Eq{"id": resourceConfigID}).
		RunWith(tx).
		QueryRow().
		Scan(&brtIDString, &cacheIDString)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}

	rc := &resourceConfig{
		id:          resourceConfigID,
		lockFactory: lockFactory,
		conn:        conn,
	}

	if brtIDString.Valid {
		var brtName string
		var unique bool
		brtID, err := strconv.Atoi(brtIDString.String)
		if err != nil {
			return nil, false, err
		}

		err = psql.Select("name, unique_version_history").
			From("base_resource_types").
			Where(sq.Eq{"id": brtID}).
			RunWith(tx).
			QueryRow().
			Scan(&brtName, &unique)
		if err != nil {
			return nil, false, err
		}

		rc.createdByBaseResourceType = &UsedBaseResourceType{brtID, brtName, unique}

	} else if cacheIDString.Valid {
		cacheID, err := strconv.Atoi(cacheIDString.String)
		if err != nil {
			return nil, false, err
		}

		usedByResourceCache, found, err := findResourceCacheByID(tx, cacheID, lockFactory, conn)
		if err != nil {
			return nil, false, err
		}

		if !found {
			return nil, false, nil
		}

		rc.createdByResourceCache = usedByResourceCache
	}

	return rc, true, nil
}
