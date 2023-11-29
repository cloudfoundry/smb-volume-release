package bosh_release_test

import (
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("BoshReleaseTest", func() {
	BeforeEach(func() {
		deploy()

		cmd := exec.Command("bosh", "-d", "bosh_release_test", "start", "-n", "smbdriver")
		session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session, 16*time.Minute).Should(gexec.Exit(0), string(session.Out.Contents()))

		stubSleep()
	})

	AfterEach(func() {
		unstubSleep()
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
					session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					Eventually(session).Should(gexec.Exit(0), string(session.Out.Contents()))
				})

				By("bosh -d bosh_release_test ssh -c rep", func() {
					cmd := exec.Command("bosh", "-d", "bosh_release_test", "ssh", "-c", "rep")
					_, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					// there's a race condition in CI where the stop is executed before the above ssh was able to be setup and start the process. This should fix it.
					cmd = exec.Command("bosh", "-d", "bosh_release_test", "ssh", "-c", "until pgrep -x rep; do sleep 1; done")
					_, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
					Expect(err).NotTo(HaveOccurred())
					time.Sleep(10 * time.Second)
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

			It("should timeout and fail drain", func() {
				By("stopping smbdriver")
				cmd := exec.Command("bosh", "-d", "bosh_release_test", "stop", "-n", "smbdriver")
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session.Out, 16*time.Minute).Should(gbytes.Say("drain scripts failed. Failed Jobs: smbdriver"))
				Eventually(session, 16*time.Minute).Should(gexec.Exit(1), string(session.Out.Contents()))
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

	When("deploying on any stemcell", func() {
		When("LD_LIBRARY_PATH is set", func() {
			It("will successfully execute keyctl", func() {
				cmd := exec.Command("bosh", "-d", "bosh_release_test", "ssh", "-c", "LD_LIBRARY_PATH=/var/vcap/packages/keyutils/keyutils/ /var/vcap/packages/keyutils/keyutils/keyctl --version")
				session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0), string(session.Out.Contents()))
			})
		})
	})
	When("LD_LIBRARY_PATH is NOT set", func() {
		It("will fail with an error about a missing shared library", func() {
			cmd := exec.Command("bosh", "-d", "bosh_release_test", "ssh", "-c", "/var/vcap/packages/keyutils/keyutils/keyctl --version")
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Eventually(session, time.Minute).Should(gbytes.Say("/var/vcap/packages/keyutils/keyutils/keyctl: /lib/x86_64-linux-gnu/libkeyutils.so.1: version `KEYUTILS_1.10' not found"))
			Eventually(session).Should(gexec.Exit(1), string(session.Out.Contents()))
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

func unstubSleep() {
	cmd := exec.Command("bosh", "-d", "bosh_release_test", "ssh", "-c", "sudo rm -f /usr/bin/sleep")
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session).Should(gexec.Exit(0), string(session.Out.Contents()))
}

func stubSleep() {
	cmd := exec.Command("bosh", "-d", "bosh_release_test", "ssh", "-c", "sudo touch /usr/bin/sleep && sudo chmod +x /usr/bin/sleep")
	session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session).Should(gexec.Exit(0), string(session.Out.Contents()))
}
