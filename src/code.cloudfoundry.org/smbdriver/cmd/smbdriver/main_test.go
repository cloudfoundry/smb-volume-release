package main_test

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"

	. "code.cloudfoundry.org/smbdriver/cmd/smbdriver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Main", func() {
	var (
		session                *gexec.Session
		command                *exec.Cmd
		dir                    string
		expectedStartOutput    string
		expectedStartErrOutput string
	)

	Context("given a running server", func() {
		BeforeEach(func() {
			expectedStartOutput = "smb-driver-server.started"
			command = exec.Command(driverPath)
		})

		JustBeforeEach(func() {
			var err error
			session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Eventually(session.Out).Should(gbytes.Say(expectedStartOutput))
			Eventually(session.Err).Should(gbytes.Say(expectedStartErrOutput))
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			session.Kill().Wait()
		})

		Context("with a driver path", func() {
			BeforeEach(func() {
				var err error
				dir, err = os.MkdirTemp("", "driversPath")
				Expect(err).ToNot(HaveOccurred())

				command.Args = append(command.Args, "-driversPath="+dir)
				command.Args = append(command.Args, "-transport=tcp-json")
			})

			It("listens on tcp/8589 by default", func() {
				EventuallyWithOffset(1, func() error {
					_, err := net.Dial("tcp", "127.0.0.1:8589")
					return err
				}, 5).ShouldNot(HaveOccurred())

				specFile := filepath.Join(dir, "smbdriver.json")
				specFileContents, err := os.ReadFile(specFile)
				Expect(err).NotTo(HaveOccurred())

				Expect(string(specFileContents)).To(MatchJSON(`{
				"Name": "smbdriver",
				"Addr": "http://127.0.0.1:8589",
				"TLSConfig": null,
				"UniqueVolumeIds": true
			}`))
			})

			Context("when invalid args are supplied", func() {

				BeforeEach(func() {
					command.Args = []string{"invalidargs"}
					expectedStartOutput = "fatal-err-aborting"
				})

				It("should error", func() {
					EventuallyWithOffset(1, func() error {
						_, err := net.Dial("tcp", "0.0.0.0:7595")
						return err
					}, 5).Should(HaveOccurred())
				})
			})
		})
	})

	Context("smb uses the right redaction patterns", func() {
		It("should redact 'password'", func() {
			Expect(SmbRedactValuePatterns()).To(ContainElement(`.*password=.*`))
		})
	})
})
