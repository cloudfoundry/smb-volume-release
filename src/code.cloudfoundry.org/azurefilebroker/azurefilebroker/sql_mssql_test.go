package azurefilebroker_test

import (
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"

	"errors"

	"code.cloudfoundry.org/azurefilebroker/azurefilebroker"
	"code.cloudfoundry.org/goshims/sqlshim/sql_fake"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MssqlVariant", func() {
	var (
		logger   lager.Logger
		fakeSql  *sql_fake.FakeSql
		err      error
		database azurefilebroker.SqlVariant

		cert                  string
		hostNameInCertificate string
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("mssql-variant-test")

		fakeSql = &sql_fake.FakeSql{}
	})

	JustBeforeEach(func() {
		database = azurefilebroker.NewMSSqlVariantWithShims(logger, "username", "password", "host", "port", "dbName", cert, hostNameInCertificate, fakeSql)
	})

	Describe(".Connect", func() {
		JustBeforeEach(func() {
			_, err = database.Connect()
		})

		Context("when hostNameInCertificate is specified", func() {
			BeforeEach(func() {
				cert = ""
				hostNameInCertificate = "domainname"
			})

			It("open call has correctly formed connection string", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeSql.OpenCallCount()).To(Equal(1))
				dbType, connectionString := fakeSql.OpenArgsForCall(0)
				Expect(dbType).To(Equal("mssql"))
				Expect(connectionString).To(Equal("sqlserver://username:password@host:port?database=dbName\u0026TrustServerCertificate=false\u0026encrypt=true\u0026hostNameInCertificate=domainname"))
			})
		})

		Context("when ca cert specified", func() {
			BeforeEach(func() {
				cert = exampleCaCert
				hostNameInCertificate = "domainname"
			})

			It("open call has correctly formed connection string", func() {
				Expect(err).NotTo(HaveOccurred())

				Expect(fakeSql.OpenCallCount()).To(Equal(1))
				dbType, connectionString := fakeSql.OpenArgsForCall(0)
				Expect(dbType).To(Equal("mssql"))
				Expect(connectionString).To(Equal("sqlserver://username:password@host:port?database=dbName\u0026TrustServerCertificate=false\u0026certificate=%2Ftmp%2FdbCert\u0026encrypt=true\u0026hostNameInCertificate=domainname"))
			})
		})

		Context("when neither ca cert nor hostNameInCertificate specified", func() {
			BeforeEach(func() {
				cert = ""
				hostNameInCertificate = ""
			})

			Context("when it can connect to a valid database", func() {
				BeforeEach(func() {
					fakeSql.OpenReturns(nil, nil)
				})

				It("reports no error", func() {
					Expect(err).NotTo(HaveOccurred())

					Expect(fakeSql.OpenCallCount()).To(Equal(1))
					dbType, connectionString := fakeSql.OpenArgsForCall(fakeSql.OpenCallCount() - 1)
					Expect(dbType).To(Equal("mssql"))
					Expect(connectionString).To(Equal("sqlserver://username:password@host:port?database=dbName"))
				})
			})

			Context("when it cannot connect to a valid database", func() {
				BeforeEach(func() {
					fakeSql = &sql_fake.FakeSql{}
					fakeSql.OpenReturns(nil, errors.New("something wrong"))
				})

				It("reports error", func() {
					Expect(err).To(HaveOccurred())
				})
			})
		})
	})
})
