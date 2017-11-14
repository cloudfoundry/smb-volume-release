package azurefilebroker

import (
	"fmt"

	"crypto/tls"
	"crypto/x509"
	"time"

	"code.cloudfoundry.org/goshims/sqlshim"
	"code.cloudfoundry.org/lager"
	"github.com/go-sql-driver/mysql"
)

type mysqlVariant struct {
	sql                   sqlshim.Sql
	dbConnectionString    string
	caCert                string
	hostNameInCertificate string
	dbName                string
	logger                lager.Logger
}

func NewMySqlVariant(logger lager.Logger, username, password, host, port, dbName, caCert, hostNameInCertificate string) SqlVariant {
	return NewMySqlVariantWithSqlObject(logger, username, password, host, port, dbName, caCert, hostNameInCertificate, &sqlshim.SqlShim{})
}

func NewMySqlVariantWithSqlObject(logger lager.Logger, username, password, host, port, dbName, caCert, hostNameInCertificate string, sql sqlshim.Sql) SqlVariant {
	return &mysqlVariant{
		sql:                   sql,
		dbConnectionString:    fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", username, password, host, port, dbName),
		caCert:                caCert,
		hostNameInCertificate: hostNameInCertificate,
		dbName:                dbName,
		logger:                logger,
	}
}

func (c *mysqlVariant) Connect() (sqlshim.SqlDB, error) {
	logger := c.logger.Session("mysql-connection-connect")
	logger.Info("start")
	defer logger.Info("end")

	if c.caCert != "" {
		logger.Info("secure-mysql-with-certificate")
		cfg, err := mysql.ParseDSN(c.dbConnectionString)
		if err != nil {
			err := fmt.Errorf("Invalid connection string for %s", c.dbName)
			logger.Error("invalid-db-connection-string", err)
			return nil, err
		}

		certBytes := []byte(c.caCert)

		caCertPool := x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM(certBytes); !ok {
			err := fmt.Errorf("Invalid CA Cert for %s", c.dbName)
			logger.Error("failed-to-parse-sql-ca", err)
			return nil, err
		}

		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			RootCAs:            caCertPool,
			ServerName:         c.hostNameInCertificate,
		}
		ourKey := "azurefilebroker-tls"
		mysql.RegisterTLSConfig(ourKey, tlsConfig)
		cfg.TLSConfig = ourKey
		cfg.Timeout = 10 * time.Minute
		cfg.ReadTimeout = 10 * time.Minute
		cfg.WriteTimeout = 10 * time.Minute

		c.dbConnectionString = cfg.FormatDSN()
	} else if c.hostNameInCertificate != "" {
		logger.Info("secure-mysql-without-certificate")
		err := mysql.RegisterTLSConfig("custom", &tls.Config{
			ServerName: c.hostNameInCertificate,
		})
		if err != nil {
			err := fmt.Errorf("Invalid hostNameInCertificate for %s", c.dbName)
			logger.Error("failed-to-register-tlsconfig", err)
			return nil, err
		}

		c.dbConnectionString = fmt.Sprintf("%s?allowNativePasswords=true&tls=custom", c.dbConnectionString)
	}

	sqlDB, err := c.sql.Open("mysql", c.dbConnectionString)
	return sqlDB, err
}

func (c *mysqlVariant) GetInitializeDatabaseSQL() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS service_instances(
			id VARCHAR(255) PRIMARY KEY,
			service_id VARCHAR(255),
			plan_id VARCHAR(255),
			organization_guid VARCHAR(255),
			space_guid VARCHAR(255),
			target_name VARCHAR(4096),
			hash_key VARCHAR(255),
			value VARCHAR(4096),
			UNIQUE (hash_key)
		)`,
		`CREATE TABLE IF NOT EXISTS service_bindings(
			id VARCHAR(255) PRIMARY KEY,
			value VARCHAR(4096)
		)`,
		`CREATE TABLE IF NOT EXISTS file_shares(
			id VARCHAR(255) PRIMARY KEY,
			instance_id VARCHAR(255),
			FOREIGN KEY instance_id(instance_id) REFERENCES service_instances(id),
			file_share_name VARCHAR(255),
			value VARCHAR(4096),
			CONSTRAINT file_share UNIQUE (instance_id, file_share_name)
		)`,
	}
}

func (c *mysqlVariant) GetAppLockSQL() string {
	return "SELECT GET_LOCK(?, ?)"
}

func (c *mysqlVariant) GetReleaseAppLockSQL() string {
	return "SELECT RELEASE_LOCK(?)"
}
