package azurefilebroker_test

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"

	"code.cloudfoundry.org/azurefilebroker/azurefilebroker"
	"code.cloudfoundry.org/azurefilebroker/azurefilebrokerfakes"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	"github.com/pivotal-cf/brokerapi"

	"database/sql"
	"encoding/json"
	"reflect"

	"code.cloudfoundry.org/goshims/sqlshim/sql_fake"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

var _ = Describe("Store", func() {
	var (
		store                                                     azurefilebroker.Store
		logger                                                    lager.Logger
		fakeSqlDb                                                 = &sql_fake.FakeSqlDB{}
		fakeVariant                                               = &azurefilebrokerfakes.FakeSqlVariant{}
		err                                                       error
		storeType                                                 string
		bindingID, serviceID, planID, orgGUID, spaceGUID, appGUID string
		instanceID, targetName, fileShareID, fileShareName        string
		serviceInstance                                           azurefilebroker.ServiceInstance
		sqlStore                                                  azurefilebroker.SqlStore
		db                                                        *sql.DB
		mock                                                      sqlmock.Sqlmock
		bindResource                                              brokerapi.BindResource
		rawParameters                                             json.RawMessage
		bindDetails                                               brokerapi.BindDetails
		fileShare                                                 azurefilebroker.FileShare
	)

	BeforeEach(func() {
		createTablesSQL := []string{
			`CREATE TABLE service_instances(...)`,
			`CREATE TABLE service_bindings(...)`,
			`CREATE TABLE file_shares(...)`,
		}
		logger = lagertest.NewTestLogger("test-broker")
		fakeVariant.ConnectReturns(fakeSqlDb, nil)
		fakeVariant.GetInitializeDatabaseSQLReturns(createTablesSQL)
		storeType = "mssql"
		store, err = azurefilebroker.NewStoreWithVariant(logger, storeType, fakeVariant)
		Expect(err).ToNot(HaveOccurred())
		db, mock, err = sqlmock.New()
		Expect(err).ToNot(HaveOccurred())
		sqlStore = azurefilebroker.SqlStore{
			StoreType: storeType,
			Database:  azurefilebrokerfakes.FakeSQLMockConnection{db},
		}
	})

	It("should open a db connection", func() {
		Expect(fakeVariant.ConnectCallCount()).To(BeNumerically(">=", 1))
	})

	It("should create tables if they don't exist", func() {
		Expect(err).To(BeNil())
		Expect(fakeSqlDb.ExecCallCount()).To(BeNumerically(">=", 3))
		Expect(fakeSqlDb.ExecArgsForCall(0)).To(ContainSubstring("CREATE TABLE service_instances"))
		Expect(fakeSqlDb.ExecArgsForCall(1)).To(ContainSubstring("CREATE TABLE service_bindings"))
		Expect(fakeSqlDb.ExecArgsForCall(2)).To(ContainSubstring("CREATE TABLE file_shares"))
	})

	Describe("RetrieveServiceInstance", func() {
		Context("When the instance exists", func() {
			BeforeEach(func() {
				Expect(err).NotTo(HaveOccurred())
				instanceID = "instance_123"
				serviceID = "service_123"
				planID = "plan_123"
				orgGUID = "org_123"
				spaceGUID = "space_123"
				targetName = "target_123"

				columns := []string{"id", "value"}

				rows := sqlmock.NewRows(columns)
				jsonvalue, err := json.Marshal(azurefilebroker.ServiceInstance{PlanID: planID, ServiceID: serviceID, OrganizationGUID: orgGUID, SpaceGUID: spaceGUID, TargetName: targetName})
				Expect(err).NotTo(HaveOccurred())
				rows.AddRow(serviceID, jsonvalue)

				mock.ExpectQuery("SELECT id, value FROM service_instances WHERE id = ?").WithArgs(instanceID).WillReturnRows(rows)
			})
			JustBeforeEach(func() {
				serviceInstance, err = sqlStore.RetrieveServiceInstance(instanceID)
			})
			It("should return the instance", func() {
				Expect(err).To(BeNil())
				Expect(mock.ExpectationsWereMet()).Should(Succeed())
				Expect(serviceInstance.ServiceID).To(Equal(serviceID))
				Expect(serviceInstance.PlanID).To(Equal(planID))
				Expect(serviceInstance.OrganizationGUID).To(Equal(orgGUID))
				Expect(serviceInstance.SpaceGUID).To(Equal(spaceGUID))
				Expect(serviceInstance.TargetName).To(Equal(targetName))
			})
		})

		Context("When the instance does not exist", func() {
			BeforeEach(func() {
				mock.ExpectQuery("SELECT id, value FROM service_instances WHERE id = ?").WithArgs(instanceID)
			})
			JustBeforeEach(func() {
				serviceInstance, err = sqlStore.RetrieveServiceInstance(instanceID)
			})
			It("should return an error", func() {
				Expect(err).To(HaveOccurred())
				Expect(reflect.DeepEqual(serviceInstance, azurefilebroker.ServiceInstance{})).To(BeTrue())
			})
		})

	})

	Describe("RetrieveBindingDetails", func() {
		Context("When the instance exists", func() {
			BeforeEach(func() {
				Expect(err).NotTo(HaveOccurred())
				appGUID = "app_123"
				planID = "plan_123"
				serviceID = "service_123"
				bindingID = "binding_123"
				bindResource = brokerapi.BindResource{AppGuid: appGUID, Route: "binding-route"}

				columns := []string{"id", "value"}
				rows := sqlmock.NewRows(columns)
				jsonvalue, err := json.Marshal(brokerapi.BindDetails{AppGUID: appGUID, PlanID: planID, ServiceID: serviceID, BindResource: &bindResource, RawParameters: rawParameters})
				Expect(err).NotTo(HaveOccurred())
				rows.AddRow(bindingID, jsonvalue)

				mock.ExpectQuery("SELECT id, value FROM service_bindings WHERE id = ?").WithArgs(bindingID).WillReturnRows(rows)
			})
			JustBeforeEach(func() {
				bindDetails, err = sqlStore.RetrieveBindingDetails(bindingID)
			})
			It("should return the binding details", func() {
				Expect(err).To(BeNil())
				Expect(mock.ExpectationsWereMet()).Should(Succeed())
				Expect(bindDetails.ServiceID).To(Equal(serviceID))
				Expect(bindDetails.PlanID).To(Equal(planID))
				Expect(bindDetails.AppGUID).To(Equal(appGUID))
				Expect(bindDetails.BindResource.AppGuid).To(Equal(appGUID))
				Expect(bindDetails.BindResource.Route).To(Equal("binding-route"))
				Expect(bindDetails.RawParameters).To(Equal(rawParameters))
			})
		})

		Context("When the binding does not exist", func() {
			BeforeEach(func() {
				mock.ExpectQuery("SELECT id, value FROM service_bindings WHERE id = ?").WithArgs(bindingID)
			})
			JustBeforeEach(func() {
				bindDetails, err = sqlStore.RetrieveBindingDetails(bindingID)
			})
			It("should return an error", func() {
				Expect(err).To(HaveOccurred())
				Expect(reflect.DeepEqual(bindDetails, brokerapi.BindDetails{})).To(BeTrue())
			})
		})
	})

	Describe("RetrieveFileShare", func() {
		Context("When the file share exists", func() {
			BeforeEach(func() {
				Expect(err).NotTo(HaveOccurred())
				fileShareID = "file_share_123"
				instanceID = "instance_123"
				fileShareName = "file_share_123"

				columns := []string{"id", "value"}

				rows := sqlmock.NewRows(columns)
				jsonvalue, err := json.Marshal(azurefilebroker.FileShare{InstanceID: instanceID, FileShareName: fileShareName})
				Expect(err).NotTo(HaveOccurred())
				rows.AddRow(fileShareID, jsonvalue)

				mock.ExpectQuery("SELECT id, value FROM file_shares WHERE id = ?").WithArgs(fileShareID).WillReturnRows(rows)
			})
			JustBeforeEach(func() {
				fileShare, err = sqlStore.RetrieveFileShare(fileShareID)
			})
			It("should return the file share", func() {
				Expect(err).To(BeNil())
				Expect(mock.ExpectationsWereMet()).Should(Succeed())
				Expect(fileShare.InstanceID).To(Equal(instanceID))
				Expect(fileShare.FileShareName).To(Equal(fileShareName))
			})
		})

		Context("When the file share does not exist", func() {
			BeforeEach(func() {
				mock.ExpectQuery("SELECT id, value FROM file_shares WHERE id = ?").WithArgs(fileShareID)
			})
			JustBeforeEach(func() {
				fileShare, err = sqlStore.RetrieveFileShare(fileShareID)
			})
			It("should return an error", func() {
				Expect(err).To(HaveOccurred())
				Expect(reflect.DeepEqual(fileShare, azurefilebroker.FileShare{})).To(BeTrue())
			})
		})

	})

	Describe("CreateServiceInstance", func() {
		BeforeEach(func() {
			Expect(err).NotTo(HaveOccurred())
			orgGUID = "org_123"
			planID = "plan_123"
			serviceID = "service_123"
			spaceGUID = "space_123"
			instanceID = "instance_123"
			targetName = "target_123"
			serviceInstance = azurefilebroker.ServiceInstance{ServiceID: serviceID, PlanID: planID, OrganizationGUID: orgGUID, SpaceGUID: spaceGUID, TargetName: targetName}
			jsonValue, err := json.Marshal(serviceInstance)
			Expect(err).NotTo(HaveOccurred())

			var buffer bytes.Buffer
			buffer.WriteString(serviceInstance.ServiceID)
			buffer.WriteString(serviceInstance.PlanID)
			buffer.WriteString(serviceInstance.OrganizationGUID)
			buffer.WriteString(serviceInstance.SpaceGUID)
			buffer.WriteString(serviceInstance.TargetName)
			hashKey := fmt.Sprintf("%x", md5.Sum(buffer.Bytes()))

			result := sqlmock.NewResult(1, 1)
			mock.ExpectExec(`INSERT INTO service_instances \(id, service_id, plan_id, organization_guid, space_guid, target_name, hash_key, value\) VALUES \([?], [?], [?], [?], [?], [?], [?], [?]\)`).WithArgs(instanceID, serviceID, planID, orgGUID, spaceGUID, targetName, hashKey, jsonValue).WillReturnResult(result)
		})
		JustBeforeEach(func() {
			err = sqlStore.CreateServiceInstance(instanceID, serviceInstance)
		})
		It("should not error and call INSERT INTO on the db", func() {
			Expect(err).To(BeNil())
			Expect(mock.ExpectationsWereMet()).Should(Succeed())
		})
	})

	Describe("CreateBindingDetails", func() {
		BeforeEach(func() {
			Expect(err).NotTo(HaveOccurred())
			appGUID = "instance_123"
			planID = "plan_123"
			serviceID = "service_123"
			bindingID = "binding_123"
			bindResource = brokerapi.BindResource{AppGuid: appGUID, Route: "binding-route"}
			bindDetails = brokerapi.BindDetails{AppGUID: appGUID, PlanID: planID, ServiceID: serviceID, BindResource: &bindResource, RawParameters: rawParameters}
			jsonValue, err := json.Marshal(bindDetails)
			Expect(err).NotTo(HaveOccurred())

			result := sqlmock.NewResult(1, 1)
			mock.ExpectExec(`INSERT INTO service_bindings \(id, value\) VALUES \([?], [?]\)`).WithArgs(bindingID, jsonValue).WillReturnResult(result)
		})
		JustBeforeEach(func() {
			err = sqlStore.CreateBindingDetails(bindingID, bindDetails, false)
		})

		It("should not error and call INSERT INTO on the db", func() {
			Expect(err).To(BeNil())
			Expect(mock.ExpectationsWereMet()).Should(Succeed())
		})
	})

	Describe("CreateFileShare", func() {
		BeforeEach(func() {
			Expect(err).NotTo(HaveOccurred())
			fileShareID = "file_share_123"
			instanceID = "instance_123"
			fileShareName = "file_share_123"
			fileShare = azurefilebroker.FileShare{InstanceID: instanceID, FileShareName: fileShareName}
			jsonValue, err := json.Marshal(fileShare)
			Expect(err).NotTo(HaveOccurred())

			result := sqlmock.NewResult(1, 1)
			mock.ExpectExec("INSERT INTO file_shares").WithArgs(fileShareID, instanceID, fileShareName, jsonValue).WillReturnResult(result)
		})
		JustBeforeEach(func() {
			err = sqlStore.CreateFileShare(fileShareID, fileShare)
		})
		It("should not error and call INSERT INTO on the db", func() {
			Expect(err).To(BeNil())
			Expect(mock.ExpectationsWereMet()).Should(Succeed())
		})
	})

	Describe("DeleteServiceInstance", func() {
		BeforeEach(func() {
			Expect(err).NotTo(HaveOccurred())
			instanceID = "instance_123"
			result := sqlmock.NewResult(1, 1)
			mock.ExpectExec("DELETE FROM service_instances WHERE id = ?").WithArgs(instanceID).WillReturnResult(result)
		})
		JustBeforeEach(func() {
			err = sqlStore.DeleteServiceInstance(instanceID)
		})
		It("should not error and call DELETE FROM on the db", func() {
			Expect(err).To(BeNil())
			Expect(mock.ExpectationsWereMet()).Should(Succeed())
		})
	})

	Describe("DeleteBindingDetails", func() {
		BeforeEach(func() {
			Expect(err).NotTo(HaveOccurred())
			bindingID = "my_binding"
			result := sqlmock.NewResult(1, 1)
			mock.ExpectExec("DELETE FROM service_bindings WHERE id = ?").WithArgs(bindingID).WillReturnResult(result)
		})
		JustBeforeEach(func() {
			err = sqlStore.DeleteBindingDetails(bindingID)
		})
		It("should not error and call DELETE FROM on the db", func() {
			Expect(err).To(BeNil())
			Expect(mock.ExpectationsWereMet()).Should(Succeed())
		})
	})

	Describe("DeleteFileShare", func() {
		BeforeEach(func() {
			Expect(err).NotTo(HaveOccurred())
			fileShareID = "file_share_123"
			result := sqlmock.NewResult(1, 1)
			mock.ExpectExec("DELETE FROM file_shares WHERE id = ?").WithArgs(fileShareID).WillReturnResult(result)
		})
		JustBeforeEach(func() {
			err = sqlStore.DeleteFileShare(fileShareID)
		})
		It("should not error and call DELETE FROM on the db", func() {
			Expect(err).To(BeNil())
			Expect(mock.ExpectationsWereMet()).Should(Succeed())
		})
	})

	Describe("UpdateFileShare", func() {
		Context("when the file share exists", func() {
			BeforeEach(func() {
				Expect(err).NotTo(HaveOccurred())
				fileShareID = "file_share_123"
				instanceID = "instance_123"
				fileShareName = "file_share_123"
				fileShare = azurefilebroker.FileShare{InstanceID: instanceID, FileShareName: fileShareName}
				jsonValue, err := json.Marshal(fileShare)
				Expect(err).NotTo(HaveOccurred())

				result := sqlmock.NewResult(0, 1)
				mock.ExpectExec("UPDATE file_shares").WithArgs(jsonValue, fileShareID).WillReturnResult(result)
			})
			JustBeforeEach(func() {
				err = sqlStore.UpdateFileShare(fileShareID, fileShare)
			})
			It("should not error and call INSERT INTO on the db", func() {
				Expect(err).To(BeNil())
				Expect(mock.ExpectationsWereMet()).Should(Succeed())
			})
		})

		Context("when the file share does not exist", func() {
			BeforeEach(func() {
				Expect(err).NotTo(HaveOccurred())
				fileShareID = "file_share_123"
				instanceID = "instance_123"
				fileShareName = "file_share_123"
				fileShare = azurefilebroker.FileShare{InstanceID: instanceID, FileShareName: fileShareName}
				jsonValue, err := json.Marshal(fileShare)
				Expect(err).NotTo(HaveOccurred())

				result := sqlmock.NewResult(0, 0)
				mock.ExpectExec("UPDATE file_shares").WithArgs(jsonValue, fileShareID).WillReturnResult(result)
			})
			JustBeforeEach(func() {
				err = sqlStore.UpdateFileShare(fileShareID, fileShare)
			})
			It("should not error and call INSERT INTO on the db", func() {
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("GetLockForUpdate", func() {
		var (
			lockName string
			seconds  int
			columns  []string
		)
		BeforeEach(func() {
			Expect(err).NotTo(HaveOccurred())
			lockName = "lock_123"
			seconds = 60

			columns = []string{"value"}
		})

		Context("When the lock is obtained", func() {
			BeforeEach(func() {
				Expect(err).NotTo(HaveOccurred())

				rows := sqlmock.NewRows(columns)
				rows.AddRow(1)

				mock.ExpectQuery(sqlStore.Database.GetAppLockSQL()).WithArgs(lockName, seconds).WillReturnRows(rows)
			})
			JustBeforeEach(func() {
				err = sqlStore.GetLockForUpdate(lockName, seconds)
			})
			It("should return the lock", func() {
				Expect(err).To(BeNil())
			})
		})

		Context("When the lock cannot be obtained", func() {
			Context(" and a error is returned", func() {
				BeforeEach(func() {
					Expect(err).NotTo(HaveOccurred())
					mock.ExpectQuery(sqlStore.Database.GetAppLockSQL()).WithArgs(lockName, seconds).WillReturnError(errors.New("error"))
				})
				JustBeforeEach(func() {
					err = sqlStore.GetLockForUpdate(lockName, seconds)
				})
				It("should return an error", func() {
					Expect(err).To(HaveOccurred())
				})
			})

			Context(" and 0 is returned", func() {
				BeforeEach(func() {
					Expect(err).NotTo(HaveOccurred())
					rows := sqlmock.NewRows(columns)
					rows.AddRow(0)

					mock.ExpectQuery(sqlStore.Database.GetAppLockSQL()).WithArgs(lockName, seconds).WillReturnRows(rows)
				})
				JustBeforeEach(func() {
					err = sqlStore.GetLockForUpdate(lockName, seconds)
				})
				It("should return an error", func() {
					Expect(err).To(HaveOccurred())
				})
			})
		})

	})

	Describe("ReleaseLockForUpdate", func() {
		var (
			lockName string
		)
		BeforeEach(func() {
			Expect(err).NotTo(HaveOccurred())
			lockName = "lock_123"
		})
		Context("When the lock is released", func() {
			BeforeEach(func() {
				Expect(err).NotTo(HaveOccurred())

				result := sqlmock.NewResult(0, 0)
				mock.ExpectExec(sqlStore.Database.GetReleaseAppLockSQL()).WithArgs(lockName).WillReturnResult(result)
			})
			JustBeforeEach(func() {
				err = sqlStore.ReleaseLockForUpdate(lockName)
			})
			It("should not return an error", func() {
				Expect(err).To(BeNil())
			})
		})

		Context("When the lock cannot be released", func() {
			BeforeEach(func() {
				Expect(err).NotTo(HaveOccurred())

				mock.ExpectExec(sqlStore.Database.GetReleaseAppLockSQL()).WithArgs(lockName).WillReturnError(errors.New("error"))
			})
			JustBeforeEach(func() {
				err = sqlStore.ReleaseLockForUpdate(lockName)
			})
			It("should return an error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

	})
})
