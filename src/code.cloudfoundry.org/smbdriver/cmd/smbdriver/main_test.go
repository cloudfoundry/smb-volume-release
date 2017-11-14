package main_test

import (
	"io/ioutil"
	"net"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Main", func() {
	var (
		session *gexec.Session
		command *exec.Cmd
		err     error
	)

	BeforeEach(func() {
		command = exec.Command(driverPath)
	})

	JustBeforeEach(func() {
		session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		session.Kill().Wait()
	})

	Context("with a driver path", func() {
		BeforeEach(func() {
			dir, err := ioutil.TempDir("", "driversPath")
			Expect(err).ToNot(HaveOccurred())

			command.Args = append(command.Args, "-driversPath="+dir)
		})

		It("listens on tcp/8589 by default", func() {
			EventuallyWithOffset(1, func() error {
				_, err := net.Dial("tcp", "127.0.0.1:8589")
				return err
			}, 5).ShouldNot(HaveOccurred())
		})
	})
})
