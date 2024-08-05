package main_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

func TestSmbDriver(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SMB Main Suite")
}

var driverPath string

var _ = BeforeSuite(func() {
	SetDefaultEventuallyTimeout(1 * time.Minute)
	var err error
	driverPath, err = Build("code.cloudfoundry.org/smbdriver/cmd/smbdriver", "-mod=vendor")
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	CleanupBuildArtifacts()
})
