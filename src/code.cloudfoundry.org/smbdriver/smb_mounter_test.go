package smbdriver_test

import (
	"context"
	"fmt"

	"os"
	"strings"

	"code.cloudfoundry.org/goshims/ioutilshim/ioutil_fake"
	"code.cloudfoundry.org/goshims/osshim/os_fake"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/nfsdriver"
	"code.cloudfoundry.org/smbdriver"
	"code.cloudfoundry.org/voldriver"
	"code.cloudfoundry.org/voldriver/driverhttp"
	"code.cloudfoundry.org/voldriver/voldriverfakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SmbMounter", func() {
	var (
		logger      lager.Logger
		testContext context.Context
		env         voldriver.Env
		err         error

		fakeInvoker *voldriverfakes.FakeInvoker
		fakeIoutil  *ioutil_fake.FakeIoutil
		fakeOs      *os_fake.FakeOs

		subject nfsdriver.Mounter

		opts map[string]interface{}
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("smb-mounter")
		testContext = context.TODO()
		env = driverhttp.NewHttpDriverEnv(logger, testContext)
		opts = map[string]interface{}{}

		fakeInvoker = &voldriverfakes.FakeInvoker{}
		fakeIoutil = &ioutil_fake.FakeIoutil{}
		fakeOs = &os_fake.FakeOs{}

		config := smbdriver.NewSmbConfig()
		config.ReadConf("username,password,vers,uid,gid,file_mode,dir_mode,readonly,ro", "", []string{})

		subject = smbdriver.NewSmbMounter(fakeInvoker, fakeOs, fakeIoutil, config)
	})

	Context("#Mount", func() {
		Context("when mount succeeds", func() {
			JustBeforeEach(func() {
				fakeInvoker.InvokeReturns(nil, nil)
				err = subject.Mount(env, "source", "target", opts)
			})

			It("should return without error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should use the passed in variables", func() {
				_, cmd, args := fakeInvoker.InvokeArgsForCall(0)
				Expect(cmd).To(Equal("mount"))
				Expect(strings.Join(args, " ")).To(ContainSubstring("source"))
				Expect(strings.Join(args, " ")).To(ContainSubstring("target"))
			})

			Context("when mounting read only with readonly", func() {
				Context("and readonly is passed", func() {
					BeforeEach(func() {
						opts["readonly"] = true
					})

					It("should include the ro flag", func() {
						_, _, args := fakeInvoker.InvokeArgsForCall(0)
						Expect(strings.Join(args, " ")).To(ContainSubstring("ro"))
					})
				})

				Context("and ro is passed", func() {
					BeforeEach(func() {
						opts["ro"] = true
					})

					It("should include the ro flag", func() {
						_, _, args := fakeInvoker.InvokeArgsForCall(0)
						Expect(strings.Join(args, " ")).To(ContainSubstring("ro"))
					})
				})
			})
		})

		Context("when mount errors", func() {
			BeforeEach(func() {
				fakeInvoker.InvokeReturns([]byte("error"), fmt.Errorf("error"))

				err = subject.Mount(env, "source", "target", opts)
			})

			It("should return without error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when mount is cancelled", func() {
			// TODO: when we pick up the lager.Context
		})

		Context("when error occurs", func() {
			BeforeEach(func() {
				opts = map[string]interface{}{}

				config := smbdriver.NewSmbConfig()
				config.ReadConf("password,vers,file_mode,dir_mode,readonly", "", []string{"username"})

				subject = smbdriver.NewSmbMounter(fakeInvoker, fakeOs, fakeIoutil, config)

				fakeInvoker.InvokeReturns(nil, nil)
			})

			JustBeforeEach(func() {
				err = subject.Mount(env, "source", "target", opts)
			})

			Context("when a required option is missing", func() {
				It("should error", func() {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("Missing mandatory options"))
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
				})
			})
		})
	})

	Context("#Unmount", func() {
		Context("when mount succeeds", func() {
			BeforeEach(func() {
				fakeInvoker.InvokeReturns(nil, nil)

				err = subject.Unmount(env, "target")
			})

			It("should return without error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should use the passed in variables", func() {
				_, cmd, args := fakeInvoker.InvokeArgsForCall(0)
				Expect(cmd).To(Equal("umount"))
				Expect(strings.Join(args, " ")).To(ContainSubstring("target"))
			})
		})

		Context("when unmount fails", func() {
			BeforeEach(func() {
				fakeInvoker.InvokeReturns([]byte("error"), fmt.Errorf("error"))
				err = subject.Unmount(env, "target")
			})

			It("should return an error", func() {
				Expect(err).To(HaveOccurred())
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
				env, _, _ := fakeInvoker.InvokeArgsForCall(0)
				Expect(fmt.Sprintf("%#v", env.Context())).To(ContainSubstring("timerCtx"))
			})
			It("reports valid mountpoint", func() {
				Expect(success).To(BeTrue())
			})
		})

		Context("when check fails", func() {
			BeforeEach(func() {
				fakeInvoker.InvokeReturns([]byte("error"), fmt.Errorf("error"))
				success = subject.Check(env, "target", "source")
			})
			It("reports invalid mountpoint", func() {
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

			It("should remove stuff", func() {
				Expect(fakeOs.RemoveAllCallCount()).NotTo(BeZero())
				path := fakeOs.RemoveAllArgsForCall(0)
				Expect(path).To(Equal("/var/vcap/data/some/path/guidy-guid-guid"))
			})

			Context("when the stuff is not a directory", func() {
				BeforeEach(func() {
					fakeStuff.IsDirReturns(false)
				})
				It("should not remove the stuff", func() {
					Expect(fakeOs.RemoveAllCallCount()).To(BeZero())
				})
			})
		})
	})
})
