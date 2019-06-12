package bosh_release_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"os/exec"
)

var _ = Describe("BoshReleaseTest", func() {
	BeforeEach(func() {
		deploy()
	})

	It("should have a smbdriver process running", func() {
		state := findProcessState("smbdriver")

		Expect(state).To(Equal("running"))
	})

	Context("when smbdriver is disabled", func() {

		BeforeEach(func() {
			cmd := exec.Command("bosh", "-d", "bosh_release_test", "delete-deployment", "-n")
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			deploy("./operations/disable-smbdriver.yml")
		})

		It("should not install packages or run rpcbind", func() {
			cmd := exec.Command("bosh", "-d", "bosh_release_test", "ssh", "-c", "stat /sbin/mount.cifs")
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(1), string(session.Out.Contents()))

			Expect(findProcessState("smbdriver")).To(Equal(""))
		})
	})
})

