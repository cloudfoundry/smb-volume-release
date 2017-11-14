package azurefilebrokerfakes

import (
	"code.cloudfoundry.org/goshims/sqlshim"
)

type FakeSQLMockConnection struct {
	sqlshim.SqlDB
}

func (fake FakeSQLMockConnection) Connect() error {
	return nil
}

func (fake FakeSQLMockConnection) GetInitializeDatabaseSQL() []string {
	return nil
}

func (fake FakeSQLMockConnection) GetAppLockSQL() string {
	return "fakegetlock ? ?"
}

func (fake FakeSQLMockConnection) GetReleaseAppLockSQL() string {
	return "fakereleaselock ?"
}
