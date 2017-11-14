package smbdriver_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestSMBDriver(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SMBDriver Suite")
}
