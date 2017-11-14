package azurefilebroker

import (
	"database/sql"
	"database/sql/driver"
	"time"

	"code.cloudfoundry.org/goshims/sqlshim"
)

type AppLock interface {
	GetAppLockSQL() string
	GetReleaseAppLockSQL() string
}

type DBInitialize interface {
	GetInitializeDatabaseSQL() []string
}

//go:generate counterfeiter -o ../azurefilebrokerfakes/fake_sql_variant.go . SqlVariant
type SqlVariant interface {
	Connect() (sqlshim.SqlDB, error)

	DBInitialize
	AppLock
}

//go:generate counterfeiter -o ../azurefilebrokerfakes/fake_sql_connection.go . SqlConnection
type SqlConnection interface {
	Connect() error
	sqlshim.SqlDB

	DBInitialize
	AppLock
}

type sqlConnection struct {
	sqlDB sqlshim.SqlDB
	leaf  SqlVariant
}

func NewSqlConnection(variant SqlVariant) SqlConnection {
	if variant == nil {
		panic("variant cannot be nil")
	}
	return &sqlConnection{
		leaf: variant,
	}
}

func (c *sqlConnection) Connect() error {
	sqlDB, err := c.leaf.Connect()
	if err != nil {
		return err
	}

	c.sqlDB = sqlDB

	err = c.Ping()
	return err
}

func (c *sqlConnection) GetInitializeDatabaseSQL() []string {
	return c.leaf.GetInitializeDatabaseSQL()
}

func (c *sqlConnection) GetAppLockSQL() string {
	return c.leaf.GetAppLockSQL()
}

func (c *sqlConnection) GetReleaseAppLockSQL() string {
	return c.leaf.GetReleaseAppLockSQL()
}

func (c *sqlConnection) Ping() error {
	return c.sqlDB.Ping()
}

func (c *sqlConnection) Close() error {
	return c.sqlDB.Close()
}

func (c *sqlConnection) SetMaxIdleConns(n int) {
	c.sqlDB.SetMaxIdleConns(n)
}

func (c *sqlConnection) SetMaxOpenConns(n int) {
	c.sqlDB.SetMaxOpenConns(n)
}

func (c *sqlConnection) SetConnMaxLifetime(d time.Duration) {
	c.sqlDB.SetConnMaxLifetime(d)
}

func (c *sqlConnection) Stats() sql.DBStats {
	return c.sqlDB.Stats()
}

func (c *sqlConnection) Prepare(query string) (*sql.Stmt, error) {
	return c.sqlDB.Prepare(query)
}

func (c *sqlConnection) Exec(query string, args ...interface{}) (sql.Result, error) {
	return c.sqlDB.Exec(query, args...)
}

func (c *sqlConnection) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return c.sqlDB.Query(query, args...)
}

func (c *sqlConnection) QueryRow(query string, args ...interface{}) *sql.Row {
	return c.sqlDB.QueryRow(query, args...)
}

func (c *sqlConnection) Begin() (*sql.Tx, error) {
	return c.sqlDB.Begin()
}

func (c *sqlConnection) Driver() driver.Driver {
	return c.sqlDB.Driver()
}
