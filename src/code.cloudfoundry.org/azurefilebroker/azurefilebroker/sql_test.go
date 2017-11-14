package azurefilebroker_test

import (
	"code.cloudfoundry.org/azurefilebroker/azurefilebroker"
	"code.cloudfoundry.org/azurefilebroker/azurefilebrokerfakes"

	"errors"
	"time"

	"code.cloudfoundry.org/goshims/sqlshim/sql_fake"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const exampleCaCert = `
-----BEGIN CERTIFICATE-----
MIIGOTCCBCGgAwIBAgIJAOE/vJd8EB24MA0GCSqGSIb3DQEBBQUAMIGyMQswCQYD
VQQGEwJGUjEPMA0GA1UECAwGQWxzYWNlMRMwEQYDVQQHDApTdHJhc2JvdXJnMRgw
FgYDVQQKDA93d3cuZnJlZWxhbi5vcmcxEDAOBgNVBAsMB2ZyZWVsYW4xLTArBgNV
BAMMJEZyZWVsYW4gU2FtcGxlIENlcnRpZmljYXRlIEF1dGhvcml0eTEiMCAGCSqG
SIb3DQEJARYTY29udGFjdEBmcmVlbGFuLm9yZzAeFw0xMjA0MjcxMDE3NDRaFw0x
MjA1MjcxMDE3NDRaMIGyMQswCQYDVQQGEwJGUjEPMA0GA1UECAwGQWxzYWNlMRMw
EQYDVQQHDApTdHJhc2JvdXJnMRgwFgYDVQQKDA93d3cuZnJlZWxhbi5vcmcxEDAO
BgNVBAsMB2ZyZWVsYW4xLTArBgNVBAMMJEZyZWVsYW4gU2FtcGxlIENlcnRpZmlj
YXRlIEF1dGhvcml0eTEiMCAGCSqGSIb3DQEJARYTY29udGFjdEBmcmVlbGFuLm9y
ZzCCAiIwDQYJKoZIhvcNAQEBBQADggIPADCCAgoCggIBAODp+8oQcK+MTuWPZVxJ
ZR75paK4zcUngupYXWSGWFXPTV7vssFk6vInePArTL+T9KwHfiZ29Pp3UbzDlysY
Kz9f9Ae50jGD6xVPwXgQ/VI979GyFXzhiEMtSYykF04tBJiDl2/FZxbHPpNxC39t
14kwuDqBin9N/ZbT5+45tbbS8ziXS+QgL5hD2q2eYCWayrGEt1Y+jDAdHDHmGnZ8
d4hbgILJAs3IInOCDjC4c1gwHFb8G4QHHTwVhjhqpkq2hQHgzWBC1l2Dku/oDYev
Zu/pfpTo3z6+NOYBrUWseQmIuG+DGMQA9KOuSQveyTywBm4G4vZKn0sCu1/v2+9T
BGv41tgS/Yf6oeeQVrbS4RFY1r9qTK6DW9wkTTesa4xoDKQrWjSJ7+aa8tvBXLGX
x2xdRNWLeRMuGBSOihwXmDr+rCJRauT7pItN5X+uWNTX1ofNksQSUMaFJ5K7L0LU
iQqU2Yyt/8UphdVZL4EFkGSA13UDWtb9mM1hY0h65LlSYwCchEphrtI9cuV+ITrS
NcN6cP/dqDx1/jWd6dqjNu7+dugwX5elQS9uUYCFmugR5s1m2eeBg3QuC7gZLE0N
NbgS7oSxKJe9KeOcw68jHWfBKsCfBfQ4fU2t/ntMybT3hCdEMQu4dgM5Tyw/UeFq
0SaJyTl+G1bTzS0FW6uUp6NLAgMBAAGjUDBOMB0GA1UdDgQWBBQjbC09PildeLhs
Pqriuy4ebIfyUzAfBgNVHSMEGDAWgBQjbC09PildeLhsPqriuy4ebIfyUzAMBgNV
HRMEBTADAQH/MA0GCSqGSIb3DQEBBQUAA4ICAQCwRJpJCgp7S+k9BT6X3kBefonE
EOYtyWXBPpuyG3Qlm1rdhc66DCGForDmTxjMmHYtNmAVnM37ILW7MoflWrAkaY19
gv88Fzwa5e6rWK4fTSpiEOc5WB2A3HPN9wJnhQXt1WWMDD7jJSLxLIwFqkzpDbDE
9122TtnIbmKNv0UQpzPV3Ygbqojy6eZHUOT05NaOT7vviv5QwMAH5WeRfiCys8CG
Sno/o830OniEHvePTYswLlX22LyfSHeoTQCCI8pocytl7IwARKCvBgeFqvPrMiqP
ch16FiU9II8KaMgpebrUSz3J1BApOOd1LBd42BeTAkNSxjRvbh8/lDWfnE7ODbKc
b6Ad3V9flFb5OBZH4aTi6QfrDnBmbLgLL8o/MLM+d3Kg94XRU9LjC2rjivQ6MC53
EnWNobcJFY+soXsJokGtFxKgIx8XrhF5GOsT2f1pmMlYL4cjlU0uWkPOOkhq8tIp
R8cBYphzXu1v6h2AaZLRq184e30ZO98omKyQoQ2KAm5AZayRrZZtjvEZPNamSuVQ
iPe3o/4tyQGq+jEMAEjLlDECu0dEa6RFntcbBPMBP3wZwE2bI9GYgvyaZd63DNdm
Xd65m0mmfOWYttfrDT3Q95YP54nHpIxKBw1eFOzrnXOqbKVmJ/1FDP2yWeooKVLf
KvbxUcDaVvXB0EU0bg==
-----END CERTIFICATE-----
`

var _ = Describe("SqlConnection", func() {
	var (
		database   azurefilebroker.SqlConnection
		toDatabase = &azurefilebrokerfakes.FakeSqlVariant{}
		fakeSqlDb  = &sql_fake.FakeSqlDB{}
	)

	BeforeEach(func() {
		database = azurefilebroker.NewSqlConnection(toDatabase)
	})

	Describe(".Connect", func() {
		var (
			err error
		)

		Context("when it can connect to a valid database", func() {
			BeforeEach(func() {
				toDatabase.ConnectReturns(fakeSqlDb, nil)
				err = database.Connect()
			})

			It("reports no error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should ping the connection to make sure it works", func() {
				Expect(fakeSqlDb.PingCallCount()).To(BeNumerically(">=", 1))
			})
		})

		Context("when it cannot connect to a valid database", func() {
			BeforeEach(func() {
				toDatabase.ConnectReturns(nil, errors.New("something wrong"))
			})

			It("reports error", func() {
				err = database.Connect()
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when it is give invalid database", func() {
			It("reports error", func() {
				defer func() {
					r := recover()
					Expect(r).To(Equal("variant cannot be nil"))
				}()
				azurefilebroker.NewSqlConnection(nil)
			})
		})
	})

	Context("when connected", func() {
		var query = `something`

		BeforeEach(func() {
			toDatabase.ConnectReturns(fakeSqlDb, nil)
			database.Connect()
		})

		Describe(".GetInitializeDatabaseSQL", func() {
			var createTablesSQL = []string{"fake-sql"}
			JustBeforeEach(func() {
				toDatabase.GetInitializeDatabaseSQLReturns(createTablesSQL)
			})

			It("should call through", func() {
				result := database.GetInitializeDatabaseSQL()
				Expect(result).To(Equal(createTablesSQL))
				Expect(toDatabase.GetInitializeDatabaseSQLCallCount()).To(BeNumerically(">=", 1))
			})
		})

		Describe(".GetAppLockSQL", func() {
			JustBeforeEach(func() {
				toDatabase.GetAppLockSQLReturns(query)
			})

			It("should call through", func() {
				result := database.GetAppLockSQL()
				Expect(result).To(Equal(query))
				Expect(toDatabase.GetAppLockSQLCallCount()).To(BeNumerically(">=", 1))
			})
		})

		Describe(".GetReleaseAppLockSQL", func() {
			JustBeforeEach(func() {
				toDatabase.GetReleaseAppLockSQLReturns(query)
			})

			It("should call through", func() {
				result := database.GetReleaseAppLockSQL()
				Expect(result).To(Equal(query))
				Expect(toDatabase.GetReleaseAppLockSQLCallCount()).To(BeNumerically(">=", 1))
			})
		})

		Describe(".Ping", func() {
			It("should call through", func() {
				database.Ping()
				Expect(fakeSqlDb.PingCallCount()).To(BeNumerically(">=", 1))
			})
		})

		Describe(".Close", func() {
			It("should call through and close variant", func() {
				database.Close()
				Expect(fakeSqlDb.CloseCallCount()).To(Equal(1))
			})
		})

		Describe(".SetMaxIdleConns", func() {
			It("should call through", func() {
				database.SetMaxIdleConns(1)
				Expect(fakeSqlDb.SetMaxIdleConnsCallCount()).To(Equal(1))
			})
		})

		Describe(".SetMaxOpenConns", func() {
			It("should call through", func() {
				database.SetMaxOpenConns(1)
				Expect(fakeSqlDb.SetMaxOpenConnsCallCount()).To(Equal(1))
			})
		})

		Describe(".SetConnMaxLifetime", func() {
			It("should call through", func() {
				database.SetConnMaxLifetime(time.Duration(1))
				Expect(fakeSqlDb.SetConnMaxLifetimeCallCount()).To(Equal(1))
			})
		})

		Describe(".Stats", func() {
			It("should call through", func() {
				database.Stats()
				Expect(fakeSqlDb.StatsCallCount()).To(Equal(1))
			})
		})

		Describe(".Prepare", func() {
			It("should call through", func() {
				database.Prepare(`something`)
				Expect(fakeSqlDb.PrepareArgsForCall(fakeSqlDb.PrepareCallCount() - 1)).To(Equal(query))
				Expect(fakeSqlDb.PrepareCallCount()).To(Equal(1))
			})
		})

		Describe(".Exec", func() {
			It("should call through", func() {
				database.Exec(`something`)
				Expect(fakeSqlDb.ExecArgsForCall(fakeSqlDb.ExecCallCount() - 1)).To(Equal(query))
				Expect(fakeSqlDb.ExecCallCount()).To(Equal(1))
			})
		})

		Describe(".Query", func() {
			It("should call through", func() {
				database.Query(`something`)
				Expect(fakeSqlDb.QueryArgsForCall(fakeSqlDb.QueryCallCount() - 1)).To(Equal(query))
				Expect(fakeSqlDb.QueryCallCount()).To(Equal(1))
			})
		})

		Describe(".QueryRow", func() {
			It("should call through", func() {
				database.QueryRow(`something`)
				Expect(fakeSqlDb.QueryRowArgsForCall(fakeSqlDb.QueryRowCallCount() - 1)).To(Equal(query))
				Expect(fakeSqlDb.QueryRowCallCount()).To(Equal(1))
			})
		})

		Describe(".Begin", func() {
			It("should call through", func() {
				database.Begin()
				Expect(fakeSqlDb.BeginCallCount()).To(Equal(1))
			})
		})

		Describe(".Driver", func() {
			It("should call through", func() {
				database.Driver()
				Expect(fakeSqlDb.DriverCallCount()).To(Equal(1))
			})
		})

	})
})
