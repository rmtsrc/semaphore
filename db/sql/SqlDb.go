package sql

import (
	"database/sql"
	"embed"
	"fmt"
	"github.com/Masterminds/squirrel"
	"github.com/go-gorp/gorp/v3"
	_ "github.com/go-sql-driver/mysql" // imports mysql driver
	_ "github.com/lib/pq"
	"github.com/semaphoreui/semaphore/db"
	"github.com/semaphoreui/semaphore/pkg/task_logger"
	"github.com/semaphoreui/semaphore/util"
	log "github.com/sirupsen/logrus"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type SqlDb struct {
	sql *gorp.DbMap
}

var initialSQL = `
create table ` + "`migrations`" + ` (
	` + "`version`" + ` varchar(255) not null primary key,
	` + "`upgraded_date`" + ` datetime null,
	` + "`notes`" + ` text null
);
`

//go:embed migrations/*.sql
var dbAssets embed.FS

func getQueryForParams(q squirrel.SelectBuilder, prefix string, props db.ObjectProps, params db.RetrieveQueryParams) (res squirrel.SelectBuilder, err error) {

	pp, err := params.Validate(props)
	if err != nil {
		return
	}

	orderDirection := "ASC"
	if pp.SortInverted {
		orderDirection = "DESC"
	}

	var orderColumn string
	if pp.SortBy == "" {
		orderColumn = props.DefaultSortingColumn
		if props.SortInverted {
			orderDirection = "DESC"
		}
	} else {
		orderColumn = pp.SortBy
	}

	if orderColumn != "" {
		q = q.OrderBy(prefix + orderColumn + " " + orderDirection)
	}

	if pp.Count > 0 {
		q = q.Limit(uint64(pp.Count))
	}

	if pp.Offset > 0 {
		q = q.Offset(uint64(pp.Offset))
	}

	res = q
	return
}

func handleRollbackError(err error) {
	if err != nil {
		log.Warn(err.Error())
	}
}

var (
	identifierQuoteRE = regexp.MustCompile("`")
)

// validateMutationResult checks the success of the update query
func validateMutationResult(res sql.Result, err error) error {
	if err != nil {
		if strings.Contains(err.Error(), "foreign key") {
			err = db.ErrInvalidOperation
		}
		return err
	}

	return nil
}

func (d *SqlDb) prepareQueryWithDialect(query string, dialect gorp.Dialect) string {
	switch dialect.(type) {
	case gorp.PostgresDialect:
		var queryBuilder strings.Builder
		argNum := 1
		for _, r := range query {
			switch r {
			case '?':
				queryBuilder.WriteString("$" + strconv.Itoa(argNum))
				argNum++
			case '`':
				queryBuilder.WriteRune('"')
			default:
				queryBuilder.WriteRune(r)
			}
		}
		query = queryBuilder.String()
	}
	return query
}

func (d *SqlDb) PrepareQuery(query string) string {
	return d.prepareQueryWithDialect(query, d.sql.Dialect)
}

func (d *SqlDb) insert(primaryKeyColumnName string, query string, args ...interface{}) (int, error) {
	var insertId int64

	switch d.sql.Dialect.(type) {
	case gorp.PostgresDialect:
		query += " returning " + primaryKeyColumnName

		err := d.sql.QueryRow(d.PrepareQuery(query), args...).Scan(&insertId)

		if err != nil {
			return 0, err
		}
	default:
		res, err := d.exec(query, args...)

		if err != nil {
			return 0, err
		}

		insertId, err = res.LastInsertId()

		if err != nil {
			return 0, err
		}
	}

	return int(insertId), nil
}

func (d *SqlDb) exec(query string, args ...interface{}) (sql.Result, error) {
	q := d.PrepareQuery(query)
	return d.sql.Exec(q, args...)
}

func (d *SqlDb) selectOne(holder interface{}, query string, args ...interface{}) error {
	return d.sql.SelectOne(holder, d.PrepareQuery(query), args...)
}

func (d *SqlDb) selectAll(i interface{}, query string, args ...interface{}) ([]interface{}, error) {
	q := d.PrepareQuery(query)
	return d.sql.Select(i, q, args...)
}

func connect() (*sql.DB, error) {
	cfg, err := util.Config.GetDBConfig()
	if err != nil {
		return nil, err
	}

	connectionString, err := cfg.GetConnectionString(true)
	if err != nil {
		return nil, err
	}

	dialect := cfg.Dialect
	return sql.Open(dialect, connectionString)
}

func createDb() error {
	cfg, err := util.Config.GetDBConfig()
	if err != nil {
		return err
	}

	if !cfg.HasSupportMultipleDatabases() {
		return nil
	}

	connectionString, err := cfg.GetConnectionString(false)
	if err != nil {
		return err
	}

	conn, err := sql.Open(cfg.Dialect, connectionString)
	if err != nil {
		return err
	}

	defer conn.Close()

	_, err = conn.Exec("create database " + cfg.GetDbName())

	if err != nil {
		log.Warn(err.Error())
	}

	return nil
}

func (d *SqlDb) getObject(projectID int, props db.ObjectProps, objectID int, object interface{}) (err error) {
	q := squirrel.Select("*").
		From(props.TableName).
		Where("id=?", objectID)

	if props.IsGlobal {
		q = q.Where("project_id is null")
	} else {
		q = q.Where("project_id=?", projectID)
	}

	query, args, err := q.ToSql()

	if err != nil {
		return
	}

	err = d.selectOne(object, query, args...)

	if err == sql.ErrNoRows {
		err = db.ErrNotFound
	}

	return
}

func (d *SqlDb) makeObjectsQuery(projectID int, props db.ObjectProps, params db.RetrieveQueryParams) (q squirrel.SelectBuilder, err error) {

	columns := []string{"*"}
	if len(props.SelectColumns) > 0 {
		columns = props.SortableColumns
	}

	q = squirrel.Select(columns...).From("`" + props.TableName + "` pe")

	if !props.IsGlobal {
		q = q.Where("pe.project_id=?", projectID)
	}

	if len(props.Ownerships) > 0 {
		for _, ownership := range props.Ownerships {
			if params.Ownership.WithoutOwnerOnly {
				q = q.Where(squirrel.Eq{
					"pe." + string(ownership.ReferringColumnSuffix): nil,
				})
			} else {
				ownerID := params.Ownership.GetOwnerID(*ownership)
				if ownerID != nil {
					q = q.Where(squirrel.Eq{
						"pe." + string(ownership.ReferringColumnSuffix): *ownerID,
					})
				}
			}
		}
	}

	q, err = getQueryForParams(q, "pe.", props, params)

	//if params.Count > 0 {
	//	q = q.Limit(uint64(params.Count))
	//}
	//
	//if params.Offset > 0 {
	//	q = q.Offset(uint64(params.Offset))
	//}

	return
}

func (d *SqlDb) getObjects(
	projectID int,
	props db.ObjectProps,
	params db.RetrieveQueryParams,
	prepare func(squirrel.SelectBuilder) squirrel.SelectBuilder,
	objects interface{},
) (err error) {
	q, err := d.makeObjectsQuery(projectID, props, params)

	if err != nil {
		return
	}

	if prepare != nil {
		q = prepare(q)
	}

	query, args, err := q.ToSql()

	if err != nil {
		return
	}

	_, err = d.selectAll(objects, query, args...)

	return
}

func (d *SqlDb) deleteObject(projectID int, props db.ObjectProps, objectID any) error {
	if props.IsGlobal {
		return validateMutationResult(
			d.exec(
				"delete from "+props.TableName+" where id=?",
				objectID))
	} else {
		return validateMutationResult(
			d.exec(
				"delete from "+props.TableName+" where project_id=? and id=?",
				projectID,
				objectID))
	}
}

func (d *SqlDb) Close(token string) {
	err := d.sql.Db.Close()
	if err != nil {
		panic(err)
	}
}

func (d *SqlDb) PermanentConnection() bool {
	return true
}

func (d *SqlDb) Connect(_ string) {
	sqlDb, err := connect()
	if err != nil {
		panic(err)
	}

	err = sqlDb.Ping()

	if err != nil {
		if err = sqlDb.Close(); err != nil {
			log.Warn("Cannot close database connection: " + err.Error())
		}

		if err = createDb(); err != nil {
			panic(err)
		}

		sqlDb, err = connect()
		if err != nil {
			panic(err)
		}

		if err = sqlDb.Ping(); err != nil {
			panic(err)
		}
	}

	cfg, err := util.Config.GetDBConfig()
	if err != nil {
		panic(err)
	}

	var dialect gorp.Dialect

	switch cfg.Dialect {
	case util.DbDriverMySQL:
		dialect = gorp.MySQLDialect{Engine: "InnoDB", Encoding: "UTF8"}
	case util.DbDriverPostgres:
		dialect = gorp.PostgresDialect{}
	}

	d.sql = &gorp.DbMap{Db: sqlDb, Dialect: dialect}

	d.sql.AddTableWithName(db.APIToken{}, "user__token").SetKeys(false, "id")
	d.sql.AddTableWithName(db.AccessKey{}, "access_key").SetKeys(true, "id")
	d.sql.AddTableWithName(db.Environment{}, "project__environment").SetKeys(true, "id")
	d.sql.AddTableWithName(db.Inventory{}, "project__inventory").SetKeys(true, "id")
	d.sql.AddTableWithName(db.Project{}, "project").SetKeys(true, "id")
	d.sql.AddTableWithName(db.Repository{}, "project__repository").SetKeys(true, "id")
	d.sql.AddTableWithName(db.Task{}, "task").SetKeys(true, "id")
	d.sql.AddTableWithName(db.TaskOutput{}, "task__output").SetUniqueTogether("task_id", "time")
	d.sql.AddTableWithName(db.Template{}, "project__template").SetKeys(true, "id")
	d.sql.AddTableWithName(db.User{}, "user").SetKeys(true, "id")
	d.sql.AddTableWithName(db.Session{}, "session").SetKeys(true, "id")
}

func (d *SqlDb) getObjectRefs(projectID int, objectProps db.ObjectProps, objectID int) (refs db.ObjectReferrers, err error) {
	refs.Templates, err = d.getObjectRefsFrom(projectID, objectProps, objectID, db.TemplateProps)
	if err != nil {
		return
	}

	refs.Repositories, err = d.getObjectRefsFrom(projectID, objectProps, objectID, db.RepositoryProps)
	if err != nil {
		return
	}

	refs.Inventories, err = d.getObjectRefsFrom(projectID, objectProps, objectID, db.InventoryProps)
	if err != nil {
		return
	}

	refs.Schedules, err = d.getObjectRefsFrom(projectID, objectProps, objectID, db.ScheduleProps)
	if err != nil {
		return
	}

	refs.Integrations, err = d.getObjectRefsFrom(projectID, objectProps, objectID, db.IntegrationAliasProps)
	if err != nil {
		return
	}

	return
}

func (d *SqlDb) getObjectRefsFrom(
	projectID int,
	objectProps db.ObjectProps,
	objectID int,
	referringObjectProps db.ObjectProps,
) (referringObjs []db.ObjectReferrer, err error) {
	referringObjs = make([]db.ObjectReferrer, 0)

	fields, err := objectProps.GetReferringFieldsFrom(referringObjectProps.Type)

	cond := ""
	vals := []interface{}{projectID}

	for _, f := range fields {
		if cond != "" {
			cond += " or "
		}

		cond += f + " = ?"

		vals = append(vals, objectID)
	}

	if cond == "" {
		return
	}

	var referringObjects reflect.Value

	if referringObjectProps.Type == db.ScheduleProps.Type {
		var referringSchedules []db.Schedule
		_, err = d.selectAll(&referringSchedules, "select template_id id from project__schedule where project_id = ? and ("+cond+")", vals...)

		if err != nil {
			return
		}

		if len(referringSchedules) == 0 {
			return
		}

		var ids []string
		for _, schedule := range referringSchedules {
			ids = append(ids, strconv.Itoa(schedule.ID))
		}

		referringObjects = reflect.New(reflect.SliceOf(db.TemplateProps.Type))
		_, err = d.selectAll(referringObjects.Interface(),
			"select id, name from project__template where id in ("+strings.Join(ids, ",")+")")
	} else {
		referringObjects = reflect.New(reflect.SliceOf(referringObjectProps.Type))
		_, err = d.selectAll(
			referringObjects.Interface(),
			"select id, name from "+referringObjectProps.TableName+" where project_id = ? and "+cond,
			vals...)
	}

	if err != nil {
		return
	}

	for i := 0; i < referringObjects.Elem().Len(); i++ {
		id := int(referringObjects.Elem().Index(i).FieldByName("ID").Int())
		name := referringObjects.Elem().Index(i).FieldByName("Name").String()
		referringObjs = append(referringObjs, db.ObjectReferrer{ID: id, Name: name})
	}

	return
}

func (d *SqlDb) Sql() *gorp.DbMap {
	return d.sql
}

func (d *SqlDb) IsInitialized() (bool, error) {
	_, err := d.sql.SelectInt(d.PrepareQuery("select count(1) from migrations"))
	return err == nil, nil
}

func (d *SqlDb) getObjectByReferrer(referrerID int, referringObjectProps db.ObjectProps, props db.ObjectProps, objectID int, object interface{}) (err error) {
	query, args, err := squirrel.Select("*").
		From(props.TableName).
		Where("id=?", objectID).
		Where(referringObjectProps.ReferringColumnSuffix+"=?", referrerID).
		ToSql()

	if err != nil {
		return
	}

	err = d.selectOne(object, query, args...)

	if err == sql.ErrNoRows {
		err = db.ErrNotFound
	}

	return
}

func (d *SqlDb) getObjectsByReferrer(
	referrerID int,
	referringObjectProps db.ObjectProps,
	props db.ObjectProps,
	params db.RetrieveQueryParams,
	objects interface{},
) (err error) {
	var referringColumn = referringObjectProps.ReferringColumnSuffix

	columns := []string{"*"}
	if len(props.SelectColumns) > 0 {
		columns = props.SelectColumns
	}

	q := squirrel.Select(columns...).From(props.TableName + " pe")

	if props.IsGlobal {
		q = q.Where("pe." + referringColumn + " is null")
	} else {
		q = q.Where("pe."+referringColumn+"=?", referrerID)
	}

	q, err = getQueryForParams(q, "pe.", props, params)

	if err != nil {
		return
	}

	query, args, err := q.ToSql()

	if err != nil {
		return
	}

	_, err = d.selectAll(objects, query, args...)

	return
}

func (d *SqlDb) deleteByReferrer(referrerID int, referringObjectProps db.ObjectProps, props db.ObjectProps, objectID int) error {
	var referringColumn = referringObjectProps.ReferringColumnSuffix

	return validateMutationResult(
		d.exec(
			"delete from "+props.TableName+" where "+referringColumn+"=? and id=?",
			referrerID,
			objectID))
}

func (d *SqlDb) deleteObjectByReferencedID(referencedID int, referencedProps db.ObjectProps, props db.ObjectProps, objectID int) error {
	field := referencedProps.ReferringColumnSuffix

	return validateMutationResult(
		d.exec("delete from "+props.TableName+" t where t."+field+"=? and t.id=?", referencedID, objectID))
}

/**
  GENERIC IMPLEMENTATION
  **/

func InsertTemplateFromType(typeInstance interface{}) (string, []interface{}) {
	val := reflect.Indirect(reflect.ValueOf(typeInstance))
	typeFieldSize := val.Type().NumField()

	fields := ""
	values := ""
	args := make([]interface{}, 0)

	if typeFieldSize > 1 {
		fields += "("
		values += "("
	}

	for i := 0; i < typeFieldSize; i++ {
		if val.Type().Field(i).Name == "ID" {
			continue
		}
		fields += val.Type().Field(i).Tag.Get("db")
		values += "?"
		args = append(args, val.Field(i))
		if i != (typeFieldSize - 1) {
			fields += ", "
			values += ", "
		}
	}

	if typeFieldSize > 1 {
		fields += ")"
		values += ")"
	}

	return fields + " values " + values, args
}

func (d *SqlDb) GetObject(props db.ObjectProps, ID int) (object interface{}, err error) {
	query, args, err := squirrel.Select("t.*").
		From(props.TableName + " as t").
		Where(squirrel.Eq{"t.id": ID}).
		OrderBy("t.id").
		ToSql()

	if err != nil {
		return
	}
	err = d.selectOne(&object, query, args...)

	return
}

func (d *SqlDb) CreateObject(props db.ObjectProps, object interface{}) (newObject interface{}, err error) {
	//err = newObject.Validate()

	if err != nil {
		return
	}

	template, args := InsertTemplateFromType(newObject)
	insertID, err := d.insert(
		"id",
		"insert into "+props.TableName+" "+template, args...)

	if err != nil {
		return
	}

	newObject = object

	v := reflect.ValueOf(newObject)
	field := v.FieldByName("ID")
	field.SetInt(int64(insertID))

	return
}

func (d *SqlDb) GetObjectsByForeignKeyQuery(props db.ObjectProps, foreignID int, foreignProps db.ObjectProps, params db.RetrieveQueryParams, objects interface{}) (err error) {
	q := squirrel.Select("*").
		From(props.TableName+" as t").
		Where(foreignProps.ReferringColumnSuffix+"=?", foreignID)

	q, err = getQueryForParams(q, "t.", props, params)

	if err != nil {
		return
	}

	query, args, err := q.
		OrderBy("t.id").
		ToSql()

	if err != nil {
		return
	}
	err = d.selectOne(&objects, query, args...)

	return
}

func (d *SqlDb) GetAllObjectsByForeignKey(props db.ObjectProps, foreignID int, foreignProps db.ObjectProps) (objects interface{}, err error) {
	query, args, err := squirrel.Select("*").
		From(props.TableName+" as t").
		Where(foreignProps.ReferringColumnSuffix+"=?", foreignID).
		OrderBy("t.id").
		ToSql()

	if err != nil {
		return
	}

	results, errQuery := d.selectAll(&objects, query, args...)

	return results, errQuery
}

func (d *SqlDb) GetAllObjects(props db.ObjectProps) (objects interface{}, err error) {
	query, args, err := squirrel.Select("*").
		From(props.TableName + " as t").
		OrderBy("t.id").
		ToSql()

	if err != nil {
		return
	}
	var results []interface{}
	results, err = d.selectAll(&objects, query, args...)

	return results, err

}

// Retrieve the Matchers & Values referencing `id' from WebhookExtractor
// --
// Examples:
// referrerCollection := db.ObjectReferrers{}
//
//	d.GetReferencesForForeignKey(db.ProjectProps, id, map[string]db.ObjectProps{
//	  'Templates': db.TemplateProps,
//	  'Inventories': db.InventoryProps,
//	  'Repositories': db.RepositoryProps
//	}, &referrerCollection)
//
// //
//
// referrerCollection := db.WebhookExtractorReferrers{}
//
//	d.GetReferencesForForeignKey(db.WebhookProps, id, map[string]db.ObjectProps{
//	  "Matchers": db.WebhookMatcherProps,
//	  "Values": db.WebhookExtractValueProps
//	}, &referrerCollection)
func (d *SqlDb) GetReferencesForForeignKey(objectProps db.ObjectProps, objectID int, referrerMapping map[string]db.ObjectProps, referrerCollection *interface{}) (err error) {

	for key, value := range referrerMapping {
		//v := reflect.ValueOf(referrerCollection)
		referrers, errRef := d.GetObjectReferences(objectProps, value, objectID)

		if errRef != nil {
			return errRef
		}
		reflect.ValueOf(referrerCollection).FieldByName(key).Set(reflect.ValueOf(referrers))
	}

	return
}

// Find Object Referrers for objectID based on referring column taken from referringObjectProps
// Example:
// GetObjectReferences(db.WebhookMatchers, db.WebhookExtractorProps, integrationID)
func (d *SqlDb) GetObjectReferences(objectProps db.ObjectProps, referringObjectProps db.ObjectProps, objectID int) (referringObjs []db.ObjectReferrer, err error) {
	referringObjs = make([]db.ObjectReferrer, 0)

	fields, err := objectProps.GetReferringFieldsFrom(objectProps.Type)

	cond := ""
	vals := []interface{}{}

	for _, f := range fields {
		if cond != "" {
			cond += " or "
		}

		cond += f + " = ?"

		vals = append(vals, objectID)
	}

	if cond == "" {
		return
	}

	referringObjects := reflect.New(reflect.SliceOf(referringObjectProps.Type))
	_, err = d.selectAll(
		referringObjects.Interface(),
		"select id, name from "+referringObjectProps.TableName+" where "+objectProps.ReferringColumnSuffix+" = ? and "+cond,
		vals...)

	if err != nil {
		return
	}

	for i := 0; i < referringObjects.Elem().Len(); i++ {
		id := int(referringObjects.Elem().Index(i).FieldByName("ID").Int())
		name := referringObjects.Elem().Index(i).FieldByName("Name").String()
		referringObjs = append(referringObjs, db.ObjectReferrer{ID: id, Name: name})
	}

	return
}

func (d *SqlDb) GetTaskStats(projectID int, templateID *int, unit db.TaskStatUnit, filter db.TaskFilter) (stats []db.TaskStat, err error) {

	stats = make([]db.TaskStat, 0)

	if unit != db.TaskStatUnitDay {
		err = fmt.Errorf("only day unit is supported")
		return
	}

	var res []struct {
		Date   string                 `db:"date"`
		Status task_logger.TaskStatus `db:"status"`
		Count  int                    `db:"count"`
	}

	q := squirrel.Select("DATE(created) AS date, status, COUNT(*) AS count").
		From("task").
		Where("project_id=?", projectID).
		GroupBy("DATE(created), status").
		OrderBy("DATE(created) DESC")

	if templateID != nil {
		q = q.Where("template_id=?", *templateID)
	}

	if filter.Start != nil {
		q = q.Where("start>=?", *filter.Start)
	}

	if filter.End != nil {
		q = q.Where("end<?", *filter.End)
	}

	query, args, err := q.ToSql()

	if err != nil {
		return
	}

	_, err = d.selectAll(&res, query, args...)

	var date string
	var stat *db.TaskStat

	for _, r := range res {

		if date != r.Date {
			date = r.Date
			stat = &db.TaskStat{
				Date:          date,
				CountByStatus: make(map[task_logger.TaskStatus]int),
			}
			stats = append(stats, *stat)
		}

		stat.CountByStatus[r.Status] = r.Count
	}

	return
}
