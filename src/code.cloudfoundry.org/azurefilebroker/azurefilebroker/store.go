package azurefilebroker

import (
	"bytes"
	"crypto/md5"
	"fmt"

	"database/sql"

	"encoding/json"

	"code.cloudfoundry.org/lager"
	"github.com/pivotal-cf/brokerapi"
)

//go:generate counterfeiter -o ../azurefilebrokerfakes/fake_store.go . Store
type Store interface {
	RetrieveServiceInstance(id string) (ServiceInstance, error)
	RetrieveBindingDetails(id string) (brokerapi.BindDetails, error)
	RetrieveFileShare(id string) (FileShare, error)

	CreateServiceInstance(id string, instance ServiceInstance) error
	CreateBindingDetails(id string, details brokerapi.BindDetails, redactRawParameter bool) error
	CreateFileShare(id string, share FileShare) error

	UpdateFileShare(id string, share FileShare) error

	DeleteServiceInstance(id string) error
	DeleteBindingDetails(id string) error
	DeleteFileShare(id string) error

	GetLockForUpdate(lockName string, timeoutInSeconds int) error
	ReleaseLockForUpdate(lockName string) error
}

type SqlStore struct {
	StoreType string
	Database  SqlConnection
}

func NewStore(logger lager.Logger, dbDriver, dbUsername, dbPassword, dbHostname, dbPort, dbName, dbCACert, hostNameInCertificate string) Store {
	var toDatabase SqlVariant
	var storeType string
	logger = logger.Session("sql-store")

	switch dbDriver {
	case "mssql":
		storeType = "mssql"
		toDatabase = NewMSSqlVariant(logger, dbUsername, dbPassword, dbHostname, dbPort, dbName, dbCACert, hostNameInCertificate)
	case "mysql":
		storeType = "mysql"
		toDatabase = NewMySqlVariant(logger, dbUsername, dbPassword, dbHostname, dbPort, dbName, dbCACert, hostNameInCertificate)
	default:
		logger.Fatal("db-driver-unrecognized", fmt.Errorf("Unrecognized Driver: %s", dbDriver))
	}
	store, err := NewStoreWithVariant(logger, storeType, toDatabase)
	if err != nil {
		logger.Fatal("new-store-with-variant", err)
	}
	return store
}

func NewStoreWithVariant(logger lager.Logger, storeType string, toDatabase SqlVariant) (Store, error) {
	database := NewSqlConnection(toDatabase)
	err := initialize(logger, database)
	if err != nil {
		logger.Error("sql-failed-to-initialize-database", err)
		return nil, err
	}

	return &SqlStore{
		StoreType: storeType,
		Database:  database,
	}, nil
}

func initialize(logger lager.Logger, db SqlConnection) error {
	logger = logger.Session("initialize-database")
	logger.Info("start")
	defer logger.Info("end")

	var err error
	err = db.Connect()
	if err != nil {
		logger.Error("sql-connect-to-database", err)
		return err
	}

	for _, query := range db.GetInitializeDatabaseSQL() {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func (s *SqlStore) RetrieveServiceInstance(id string) (ServiceInstance, error) {
	var serviceID string
	var value []byte
	serviceInstance := ServiceInstance{}

	query := "SELECT id, value FROM service_instances WHERE id = ?"
	err := s.Database.QueryRow(query, id).Scan(&serviceID, &value)
	if err == nil {
		err = json.Unmarshal(value, &serviceInstance)
		if err != nil {
			return ServiceInstance{}, err
		}
		return serviceInstance, nil
	} else if err == sql.ErrNoRows {
		return serviceInstance, brokerapi.ErrInstanceDoesNotExist
	}
	return serviceInstance, err
}

func (s *SqlStore) RetrieveBindingDetails(id string) (brokerapi.BindDetails, error) {
	var bindingID string
	var value []byte
	bindDetails := brokerapi.BindDetails{}

	query := "SELECT id, value FROM service_bindings WHERE id = ?"
	err := s.Database.QueryRow(query, id).Scan(&bindingID, &value)
	if err == nil {
		err = json.Unmarshal(value, &bindDetails)
		if err != nil {
			return bindDetails, err
		}
		return bindDetails, nil
	} else if err == sql.ErrNoRows {
		return bindDetails, brokerapi.ErrInstanceDoesNotExist
	}
	return bindDetails, err
}

func (s *SqlStore) RetrieveFileShare(id string) (FileShare, error) {
	var serviceID string
	var value []byte
	share := FileShare{}

	query := "SELECT id, value FROM file_shares WHERE id = ?"
	err := s.Database.QueryRow(query, id).Scan(&serviceID, &value)
	if err == nil {
		err = json.Unmarshal(value, &share)
		if err != nil {
			return share, err
		}
		return share, nil
	} else if err == sql.ErrNoRows {
		return share, brokerapi.ErrInstanceDoesNotExist
	}
	return share, err
}

func (s *SqlStore) CreateServiceInstance(id string, instance ServiceInstance) error {
	jsonData, err := json.Marshal(instance)
	if err != nil {
		return err
	}

	// Maximum length of a unique key in mysql is 767
	// So here we calculates MD5 to generate a unique key to avoid duplicate instances
	var buffer bytes.Buffer
	buffer.WriteString(instance.ServiceID)
	buffer.WriteString(instance.PlanID)
	buffer.WriteString(instance.OrganizationGUID)
	buffer.WriteString(instance.SpaceGUID)
	buffer.WriteString(instance.TargetName)
	hashKey := fmt.Sprintf("%x", md5.Sum(buffer.Bytes()))

	query := "INSERT INTO service_instances (id, service_id, plan_id, organization_guid, space_guid, target_name, hash_key, value) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"
	_, err = s.Database.Exec(query, id, instance.ServiceID, instance.PlanID, instance.OrganizationGUID, instance.SpaceGUID, instance.TargetName, hashKey, jsonData)
	if err != nil {
		return err
	}
	return nil
}

func (s *SqlStore) CreateBindingDetails(id string, details brokerapi.BindDetails, redactRawParameter bool) error {
	// Preexisting shares do not need to use RawParameters of BindDetails in unbind operation
	// For security, do not store RawParameters in broker's database.
	if redactRawParameter {
		details.RawParameters = nil
	}

	jsonData, err := json.Marshal(details)
	if err != nil {
		return err
	}

	query := "INSERT INTO service_bindings (id, value) VALUES (?, ?)"
	_, err = s.Database.Exec(query, id, jsonData)
	if err != nil {
		return err
	}
	return nil
}

func (s *SqlStore) CreateFileShare(id string, share FileShare) error {
	jsonData, err := json.Marshal(share)
	if err != nil {
		return err
	}

	query := "INSERT INTO file_shares (id, instance_id, file_share_name, value) VALUES (?, ?, ?, ?)"
	_, err = s.Database.Exec(query, id, share.InstanceID, share.FileShareName, jsonData)
	if err != nil {
		return err
	}
	return nil
}

func (s *SqlStore) DeleteServiceInstance(id string) error {
	query := "DELETE FROM service_instances WHERE id = ?"
	_, err := s.Database.Exec(query, id)
	if err != nil {
		return err
	}
	return nil
}

func (s *SqlStore) DeleteBindingDetails(id string) error {
	query := "DELETE FROM service_bindings WHERE id = ?"
	_, err := s.Database.Exec(query, id)
	if err != nil {
		return err
	}
	return nil
}

func (s *SqlStore) DeleteFileShare(id string) error {
	query := "DELETE FROM file_shares WHERE id = ?"
	_, err := s.Database.Exec(query, id)
	if err != nil {
		return err
	}
	return nil
}

func (s *SqlStore) UpdateFileShare(id string, share FileShare) error {
	jsonData, err := json.Marshal(share)
	if err != nil {
		return err
	}
	query := "UPDATE file_shares set value = ? WHERE id = ?"
	result, err := s.Database.Exec(query, jsonData, id)
	if err != nil {
		return err
	}
	ret, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("Cannot parse RowsAffected when updating the file share: %v", err)
	}
	if ret == int64(0) {
		return fmt.Errorf("Cannot update the file share in the database")
	}
	return nil
}

func (s *SqlStore) GetLockForUpdate(lockName string, seconds int) error {
	query := s.Database.GetAppLockSQL()
	var ret int
	err := s.Database.QueryRow(query, lockName, seconds).Scan(&ret)
	if err != nil {
		return fmt.Errorf("Cannot get the lock %q for update in %d seconds. Error: %v", lockName, seconds, err)
	}
	if ret != 1 {
		return fmt.Errorf("Cannot get the lock %q for update in %d seconds because the lock has been obtained by some process", lockName, seconds)
	}
	return nil
}

func (s *SqlStore) ReleaseLockForUpdate(lockName string) error {
	query := s.Database.GetReleaseAppLockSQL()

	_, err := s.Database.Exec(query, lockName)
	if err != nil {
		return fmt.Errorf("Cannot release the lock %q for update. Error: %v", lockName, err)
	}
	return nil
}
