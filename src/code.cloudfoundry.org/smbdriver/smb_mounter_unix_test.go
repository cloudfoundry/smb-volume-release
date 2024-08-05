//go:build linux || darwin
// +build linux darwin

package smbdriver_test

import (
	"context"
	"fmt"
	"os"
	"strings"

	"code.cloudfoundry.org/dockerdriver"
	"code.cloudfoundry.org/dockerdriver/driverhttp"
	"code.cloudfoundry.org/goshims/ioutilshim/ioutil_fake"
	"code.cloudfoundry.org/goshims/osshim/os_fake"
	"code.cloudfoundry.org/lager/v3/lagertest"
	"code.cloudfoundry.org/smbdriver"
	vmo "code.cloudfoundry.org/volume-mount-options"
	"code.cloudfoundry.org/volumedriver"
	"code.cloudfoundry.org/volumedriver/invokerfakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("SmbMounter", func() {
	var (
		logger      *lagertest.TestLogger
		testContext context.Context
		env         dockerdriver.Env
		err         error

		fakeInvoker      *invokerfakes.FakeInvoker
		fakeInvokeResult *invokerfakes.FakeInvokeResult
		fakeIoutil       *ioutil_fake.FakeIoutil
		fakeOs           *os_fake.FakeOs

		subject volumedriver.Mounter

		opts map[string]interface{}
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("smb-mounter")
		testContext = context.TODO()
		env = driverhttp.NewHttpDriverEnv(logger, testContext)
		opts = map[string]interface{}{}
		opts["mount"] = "/data"
		opts["source"] = "source-from-opts"
		opts["username"] = "foo"
		opts["password"] = "bar"
		opts["version"] = "2.0"
		opts["mfsymlinks"] = true

		fakeInvoker = &invokerfakes.FakeInvoker{}
		fakeInvokeResult = &invokerfakes.FakeInvokeResult{}
		fakeInvoker.InvokeReturns(fakeInvokeResult)

		fakeInvokeResult.WaitReturns(nil)
		fakeInvokeResult.WaitForReturns(nil)
		fakeIoutil = &ioutil_fake.FakeIoutil{}
		fakeOs = &os_fake.FakeOs{}

		configMask, err := smbdriver.NewSmbVolumeMountMask()
		Expect(err).NotTo(HaveOccurred())

		subject = smbdriver.NewSmbMounter(fakeInvoker, fakeOs, fakeIoutil, configMask, false, false)
	})

	Context("#Mount", func() {
		JustBeforeEach(func() {
			err = subject.Mount(env, "source", "target", opts)
		})

		Context("when mount succeeds", func() {

			It("should return without error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should use the passed in variables", func() {
				Expect(err).NotTo(HaveOccurred())
				_, cmd, args, _ := fakeInvoker.InvokeArgsForCall(0)
				Expect(cmd).To(Equal("mount"))
				Expect(strings.Join(args, " ")).To(ContainSubstring("source"))
				Expect(strings.Join(args, " ")).To(ContainSubstring("target"))
				Expect(strings.Join(args, " ")).To(ContainSubstring("uid=2000"))
				Expect(strings.Join(args, " ")).To(ContainSubstring("gid=2000"))
				Expect(strings.Join(args, " ")).To(ContainSubstring("vers=2.0"))
				Expect(strings.Join(args, " ")).To(ContainSubstring("mfsymlinks"))
			})

			Context("smb versions", func() {
				JustBeforeEach(func() {
					fakeInvoker = &invokerfakes.FakeInvoker{}
					fakeInvokeResult = &invokerfakes.FakeInvokeResult{}
					fakeInvoker.InvokeReturns(fakeInvokeResult)

					configMask, err := smbdriver.NewSmbVolumeMountMask()
					Expect(err).NotTo(HaveOccurred())

					subject = smbdriver.NewSmbMounter(fakeInvoker, fakeOs, fakeIoutil, configMask, false, false)
				})

				DescribeTable("when passed smb versions", func(version string, containsVers bool) {
					opts["version"] = version
					err = subject.Mount(env, "source", "target", opts)
					Expect(err).NotTo(HaveOccurred())
					_, cmd, args, _ := fakeInvoker.InvokeArgsForCall(0)
					Expect(cmd).To(Equal("mount"))

					if containsVers {
						Expect(strings.Join(args, " ")).To(ContainSubstring(fmt.Sprintf("vers=%s", version)))
					} else {
						Expect(strings.Join(args, " ")).NotTo(ContainSubstring("vers"))
					}

				},
					Entry("1.0", "1.0", true),
					Entry("2.0", "2.0", true),
					Entry("2.1", "2.1", true),
					Entry("3.0", "3.0", true),
				)
			})

			It("should not pass username or password", func() {
				Expect(err).NotTo(HaveOccurred())
				_, cmd, args, envVars := fakeInvoker.InvokeArgsForCall(0)
				Expect(cmd).To(Equal("mount"))
				Expect(strings.Join(args, " ")).NotTo(ContainSubstring("username"))
				Expect(strings.Join(args, " ")).NotTo(ContainSubstring("password"))
				Expect(strings.Join(envVars, " ")).To(ContainSubstring("USER=foo"))
				Expect(strings.Join(envVars, " ")).To(ContainSubstring("PASSWD=bar"))
			})

			Context("when mounting read only with readonly", func() {
				Context("and readonly is passed", func() {
					BeforeEach(func() {
						opts["readonly"] = true
					})

					It("should include the ro flag", func() {
						Expect(err).NotTo(HaveOccurred())
						_, _, args, _ := fakeInvoker.InvokeArgsForCall(0)
						Expect(strings.Join(args, " ")).To(ContainSubstring("ro"))
					})
				})

				Context("and ro is passed", func() {
					BeforeEach(func() {
						opts["ro"] = true
					})

					It("should include the ro flag", func() {
						Expect(err).NotTo(HaveOccurred())
						_, _, args, _ := fakeInvoker.InvokeArgsForCall(0)
						Expect(strings.Join(args, " ")).To(ContainSubstring("ro"))
					})
				})
			})

			Context("when mounting with mfsymlinks option", func() {
				Context("and mfsymlinks is true", func() {
					BeforeEach(func() {
						opts["mfsymlinks"] = true
					})

					It("should include the mfsymlinks flag", func() {
						Expect(err).NotTo(HaveOccurred())
						_, _, args, _ := fakeInvoker.InvokeArgsForCall(0)
						Expect(strings.Join(args, " ")).To(ContainSubstring("mfsymlinks"))
						Expect(strings.Join(args, " ")).NotTo(ContainSubstring("mfsymlinks=true"))
					})
				})

				Context("and mfsymlinks is false", func() {
					BeforeEach(func() {
						opts["mfsymlinks"] = false
					})

					It("should not include the mfsymlinks flag", func() {
						Expect(err).NotTo(HaveOccurred())
						_, _, args, _ := fakeInvoker.InvokeArgsForCall(0)
						Expect(strings.Join(args, " ")).NotTo(ContainSubstring("mfsymlinks"))
					})
				})

			})

			Context("when configured without forceNoDfs", func() {
				It("should not pass the nodfs mount flag", func() {
					Expect(err).NotTo(HaveOccurred())
					_, _, args, _ := fakeInvoker.InvokeArgsForCall(0)
					Expect(strings.Join(args, " ")).NotTo(ContainSubstring("nodfs"))
				})
			})

			Context("when configured with forceNoDfs", func() {
				// The forceNoDfs option was added on 2024-01-09.
				//
				// We had seen a large deployment in which upgrading beyond jammy v1.199
				// stemcells caused all apps using SMB mounts to fail with:
				// "CIFS: VFS: cifs_mount failed w/return code = -19"
				// errors. This turned out to be because the kernel had a regression around
				// CIFS DFS handling.
				//
				// The fix was to re-bind the SMB service with the mount parameter
				// "nodfs". This option was intended to allow the platform operator to
				// apply that fix across the whole foundation, rather than relying on
				// application authors to re-bind their SMB services.
				It("should pass the nodfs mount flag", func() {
					fakeInvoker = &invokerfakes.FakeInvoker{}
					fakeInvoker.InvokeReturns(fakeInvokeResult)
					configMask, err := smbdriver.NewSmbVolumeMountMask()
					Expect(err).NotTo(HaveOccurred())

					subject = smbdriver.NewSmbMounter(fakeInvoker, fakeOs, fakeIoutil, configMask, false, true)
					Expect(subject.Mount(env, "source", "target", opts)).To(Succeed())

					_, _, args, _ := fakeInvoker.InvokeArgsForCall(0)
					Expect(strings.Join(args, " ")).To(ContainSubstring("nodfs"))
				})
			})

			Context("when configured without forceNoserverino", func() {
				It("should not pass the noserverino mount flag", func() {
					Expect(err).NotTo(HaveOccurred())
					_, _, args, _ := fakeInvoker.InvokeArgsForCall(0)
					Expect(strings.Join(args, " ")).NotTo(ContainSubstring("noserverino"))
				})
			})

			Context("when configured with forceNoserverino", func() {
				// The forceNoserverino option was added on 2023-04-15.
				//
				// We had seen a large deployment in which upgrading from xenial to jammy
				// stemcells caused all apps using SMB mounts to fail with "Stale file handle"
				// errors. This turned out to be because the SMB server was suggesting inode
				// numbers, instead of allowing the client to generate temporary inode numbers.
				//
				// The fix was to re-bind the SMB service with the mount parameter
				// "noserverino". This option was intended to allow the platform operator to
				// apply that fix across the whole foundation, rather than relying on
				// application authors to re-bind their SMB services.
				It("should pass the noserverino mount flag", func() {
					fakeInvoker = &invokerfakes.FakeInvoker{}
					fakeInvoker.InvokeReturns(fakeInvokeResult)
					configMask, err := smbdriver.NewSmbVolumeMountMask()
					Expect(err).NotTo(HaveOccurred())

					subject = smbdriver.NewSmbMounter(fakeInvoker, fakeOs, fakeIoutil, configMask, true, false)
					Expect(subject.Mount(env, "source", "target", opts)).To(Succeed())

					_, _, args, _ := fakeInvoker.InvokeArgsForCall(0)
					Expect(strings.Join(args, " ")).To(ContainSubstring("noserverino"))
				})
			})
		})

		Context("when mount cmd errors", func() {
			BeforeEach(func() {
				fakeInvokeResult.WaitReturns(fmt.Errorf("mount error"))
			})

			It("should return with error", func() {
				Expect(err).To(HaveOccurred())
				_, ok := err.(dockerdriver.SafeError)
				Expect(ok).To(BeTrue())
				Expect(err).To(MatchError("mount error"))
			})
		})

		Context("when error occurs", func() {
			BeforeEach(func() {
				opts = map[string]interface{}{}

				configMask, err2 := vmo.NewMountOptsMask(
					[]string{
						"password",
						"vers",
						"file_mode",
						"dir_mode",
						"readonly",
					},
					map[string]interface{}{},
					map[string]string{},
					[]string{},
					[]string{"username"},
				)
				Expect(err2).NotTo(HaveOccurred())

				subject = smbdriver.NewSmbMounter(fakeInvoker, fakeOs, fakeIoutil, configMask, false, false)
			})

			Context("when a required option is missing", func() {
				It("should error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Missing mandatory options"))
					_, ok := err.(dockerdriver.SafeError)
					Expect(ok).To(BeTrue())
				})
			})

			Context("when a disallowed option is passed", func() {
				BeforeEach(func() {
					opts["username"] = "fake-username"
					opts["uid"] = "uid"
				})

				It("should error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Not allowed options"))
					_, ok := err.(dockerdriver.SafeError)
					Expect(ok).To(BeTrue())
				})
			})
		})

		Context("when mandatory username argument is not provided", func() {
			BeforeEach(func() {
				opts["password"] = ""
				delete(opts, "username")
			})

			It("should return with error", func() {
				Expect(err).To(HaveOccurred())
				_, ok := err.(dockerdriver.SafeError)
				Expect(ok).To(BeTrue())
				Expect(err.Error()).To(ContainSubstring("Missing mandatory options: username"))
			})
		})

		Context("when mandatory password argument is not provided", func() {
			BeforeEach(func() {
				opts["username"] = ""
				delete(opts, "password")
			})

			It("should return with error", func() {
				Expect(err).To(HaveOccurred())
				_, ok := err.(dockerdriver.SafeError)
				Expect(ok).To(BeTrue())
				Expect(err.Error()).To(ContainSubstring("Missing mandatory options: password"))
			})
		})
	})

	Context("#Unmount", func() {
		Context("when mount succeeds", func() {
			BeforeEach(func() {
				err = subject.Unmount(env, "target")
			})

			It("should return without error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should use the passed in variables", func() {
				Expect(fakeInvoker.InvokeCallCount()).To(Equal(1))
				Expect(fakeInvokeResult.WaitCallCount()).To(Equal(1))
				_, cmd, args, _ := fakeInvoker.InvokeArgsForCall(0)
				Expect(cmd).To(Equal("umount"))
				Expect(len(args)).To(Equal(2))
				Expect(args[0]).To(Equal("-l"))
				Expect(args[1]).To(Equal("target"))
			})
		})

		Context("when unmount cmd fails", func() {
			BeforeEach(func() {
				fakeInvokeResult.WaitReturns(fmt.Errorf("umount cmd"))
				err = subject.Unmount(env, "target")
			})

			It("should return an error", func() {
				Expect(err).To(HaveOccurred())

				_, ok := err.(dockerdriver.SafeError)
				Expect(ok).To(BeTrue())
				Expect(err).To(MatchError("umount cmd"))
			})
		})

	})

	Context("#Check", func() {
		var (
			success bool
		)

		Context("when check succeeds", func() {
			BeforeEach(func() {
				success = subject.Check(env, "target", "source")
			})
			It("uses correct context", func() {
				invokeEnv, _, _, _ := fakeInvoker.InvokeArgsForCall(0)
				Expect(fmt.Sprintf("%#v", invokeEnv.Context())).To(ContainSubstring("timerCtx"))
			})
			It("reports valid mountpoint", func() {
				Expect(success).To(BeTrue())
			})
		})

		Context("when check cmd fails", func() {
			BeforeEach(func() {
				fakeInvokeResult.WaitReturns(fmt.Errorf("mountpoint cmd error"))
				success = subject.Check(env, "target", "source")
			})
			It("reports invalid mountpoint", func() {
				Expect(logger.Buffer()).To(gbytes.Say("unable to verify volume target.*mountpoint cmd error"))
				Expect(success).To(BeFalse())
			})
		})

	})

	Context("#Purge", func() {
		JustBeforeEach(func() {
			subject.Purge(env, "/var/vcap/data/some/path")
		})

		Context("when stuff is in the directory", func() {
			var fakeStuff *ioutil_fake.FakeFileInfo

			BeforeEach(func() {
				fakeStuff = &ioutil_fake.FakeFileInfo{}
				fakeStuff.NameReturns("guidy-guid-guid")
				fakeStuff.IsDirReturns(true)

				fakeIoutil.ReadDirReturns([]os.FileInfo{fakeStuff}, nil)
			})

			It("should attempt to unmount the directory", func() {
				Expect(fakeInvoker.InvokeCallCount()).To(Equal(1))
				Expect(fakeInvokeResult.WaitCallCount()).To(Equal(1))

				_, proc, args, _ := fakeInvoker.InvokeArgsForCall(0)
				Expect(proc).To(Equal("umount"))
				Expect(len(args)).To(Equal(3))
				Expect(args[0]).To(Equal("-l"))
				Expect(args[1]).To(Equal("-f"))
				Expect(args[2]).To(Equal("/var/vcap/data/some/path/guidy-guid-guid"))
				Eventually(logger.Buffer()).Should(gbytes.Say("unmount-successful"))
			})

			Context("with multiple directories", func() {
				var fakeStuff2 *ioutil_fake.FakeFileInfo

				BeforeEach(func() {
					fakeStuff2 = &ioutil_fake.FakeFileInfo{}
					fakeStuff2.NameReturns("guidy-guid-guid2")
					fakeStuff2.IsDirReturns(true)

					fakeIoutil.ReadDirReturns([]os.FileInfo{fakeStuff, fakeStuff2}, nil)
				})
				It("should attempt to unmount each directory", func() {
					Expect(fakeInvoker.InvokeCallCount()).To(Equal(2))
					Expect(fakeInvokeResult.WaitCallCount()).To(Equal(2))

					_, proc, args, _ := fakeInvoker.InvokeArgsForCall(0)
					Expect(proc).To(Equal("umount"))
					Expect(len(args)).To(Equal(3))
					Expect(args[0]).To(Equal("-l"))
					Expect(args[1]).To(Equal("-f"))
					Expect(args[2]).To(Equal("/var/vcap/data/some/path/guidy-guid-guid"))
					Eventually(logger.Buffer()).Should(gbytes.Say("unmount-successful"))

					_, proc, args, _ = fakeInvoker.InvokeArgsForCall(1)
					Expect(proc).To(Equal("umount"))
					Expect(len(args)).To(Equal(3))
					Expect(args[0]).To(Equal("-l"))
					Expect(args[1]).To(Equal("-f"))
					Expect(args[2]).To(Equal("/var/vcap/data/some/path/guidy-guid-guid2"))
					Eventually(logger.Buffer()).Should(gbytes.Say("unmount-successful"))
				})

			})

			Context("umount cmd fails", func() {
				BeforeEach(func() {
					fakeInvokeResult.WaitReturns(fmt.Errorf("umount cmd error"))
				})

				It("returns", func() {
					Expect(fakeInvokeResult.WaitCallCount()).To(Equal(1))
					Expect(logger.Buffer()).To(gbytes.Say("warning-umount-failed.*umount cmd error"))
					Consistently(logger.Buffer()).ShouldNot(gbytes.Say("unmount-successful"))
				})
			})

			It("should remove the mount directory", func() {
				Expect(fakeOs.RemoveCallCount()).To(Equal(1))

				path := fakeOs.RemoveArgsForCall(0)
				Expect(path).To(Equal("/var/vcap/data/some/path/guidy-guid-guid"))
			})

			Context("when the stuff is not a directory", func() {
				BeforeEach(func() {
					fakeStuff.IsDirReturns(false)
				})

				It("should not remove the stuff", func() {
					Expect(fakeInvoker.InvokeCallCount()).To(Equal(0))
					Expect(fakeInvokeResult.WaitCallCount()).To(Equal(0))
					Expect(fakeOs.RemoveCallCount()).To(BeZero())
				})
			})
		})
	})
})
