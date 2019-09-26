package bosh_release_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"os/exec"
	"time"
)

var _ = Describe("BoshReleaseTest", func() {
	BeforeEach(func() {
		deploy()

		cmd := exec.Command("bosh", "-d", "bosh_release_test", "start", "-n", "smbdriver")
		session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, 16*time.Minute).Should(gexec.Exit(0), string(session.Out.Contents()))
	})

	It("should have a smbdriver process running", func() {
		state := findProcessState("smbdriver")

		Expect(state).To(Equal("running"))
	})

	Context("Monit restart", func() {
		JustBeforeEach(func() {
			cmd := exec.Command("bosh", "-d", "bosh_release_test", "restart", "smbdriver", "-n")
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
		})

		Context("smb mount path has a child directory", func() {
			BeforeEach(func() {
				By("sudo touch /var/vcap/data/volumes/smb/child_directory")
				cmd := exec.Command("bosh", "-d", "bosh_release_test", "ssh", "-c", "sudo touch /var/vcap/data/volumes/smb/child_directory")
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0), string(session.Out.Contents()))

				By("sudo chown root:root /var/vcap/data/volumes/smb/child_directory")
				cmd = exec.Command("bosh", "-d", "bosh_release_test", "ssh", "-c", "sudo chown root:root /var/vcap/data/volumes/smb/child_directory")
				session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0), string(session.Out.Contents()))
			})

			It("Only cell_mount_path should chown and not any child directories", func() {
				cmd := exec.Command("bosh", "-d", "bosh_release_test", "ssh", "-c", "sudo stat --format='%U:%G' /var/vcap/data/volumes/smb")
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0), string(session.Out.Contents()))
				Expect(session).Should(gbytes.Say("vcap:vcap"))


				cmd = exec.Command("bosh", "-d", "bosh_release_test", "ssh", "-c", "sudo stat --format='%U:%G' /var/vcap/data/volumes/smb/child_directory")
				session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0), string(session.Out.Contents()))
				Expect(session).Should(gbytes.Say("root:root"))
			})
		})
	})

	Context("smbdriver drain", func() {
		It("should successfully drain", func() {
			By("bosh stopping the smbdriver")
			cmd := exec.Command("bosh", "-d", "bosh_release_test", "stop", "-n", "smbdriver")
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0), string(session.Out.Contents()))
		})

		Context("when smbdriver is not reachable", func() {
			BeforeEach(func() {
				By("drain cannot reach the smbdriver")
				cmd := exec.Command("bosh", "-d", "bosh_release_test", "ssh", "smbdriver", "-c", "sudo iptables -t filter -A OUTPUT -p tcp --dport 8590  -j DROP")
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0), string(session.Out.Contents()))
			})

			AfterEach(func() {
				cmd := exec.Command("bosh", "-d", "bosh_release_test", "ssh", "smbdriver", "-c", "sudo iptables -t filter -D OUTPUT -p tcp --dport 8590  -j DROP")
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0), string(session.Out.Contents()))

				cmd = exec.Command("bosh", "-d", "bosh_release_test", "start", "-n", "smbdriver")
				session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0), string(session.Out.Contents()))
			})

			It("should successfully drain", func() {
				cmd := exec.Command("bosh", "-d", "bosh_release_test", "stop", "-n", "smbdriver")
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0), string(session.Out.Contents()))
			})
		})

		Context("when the rep process takes longer than 15 minutes to exit", func() {
			BeforeEach(func() {

				By("bosh -d bosh_release_test scp "+repBuildPackagePath+" smbdriver:/tmp/rep", func() {
					cmd := exec.Command("bosh", "-d", "bosh_release_test", "scp", repBuildPackagePath, "smbdriver:/tmp/rep")
					session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					Eventually(session).Should(gexec.Exit(0), string(session.Out.Contents()))
				})

				By("bosh -d bosh_release_test ssh -c sudo chmod +x /tmp/rep && sudo mv /tmp/rep /bin/rep", func() {
					cmd := exec.Command("bosh", "-d", "bosh_release_test", "ssh", "-c", "sudo chmod +x /tmp/rep && sudo mv /tmp/rep /bin/rep")
					_, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
				})

				By("bosh -d bosh_release_test ssh -c rep", func() {
					cmd := exec.Command("bosh", "-d", "bosh_release_test", "ssh", "-c", "rep")
					_, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
				})
			})

			AfterEach(func() {
				By("bosh -d bosh_release_test ssh smbdriver -c sudo pkill -f 'rep'", func() {
					cmd := exec.Command("bosh", "-d", "bosh_release_test", "ssh", "smbdriver", "-c", "sudo pkill -f 'rep'")
					session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					Eventually(session).Should(gexec.Exit(0), string(session.Out.Contents()))
				})
			})

			It("should timeout and successfully drain", func() {
				By("stopping smbdriver")
				cmd := exec.Command("bosh", "-d", "bosh_release_test", "stop", "-n", "smbdriver")
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session, 16*time.Minute).Should(gexec.Exit(0), string(session.Out.Contents()))
			})
		})
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
			exitCodeIndicatingThatFileDoesNotExist := 1

			cmd := exec.Command("bosh", "-d", "bosh_release_test", "ssh", "-c", "stat /sbin/mount.cifs")
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(exitCodeIndicatingThatFileDoesNotExist), string(session.Out.Contents()))

			Expect(findProcessState("smbdriver")).To(Equal(""))
		})
	})
})
