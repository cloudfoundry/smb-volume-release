package driveradminlocal_test

import (
	"context"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	"code.cloudfoundry.org/smbdriver/driveradmin"
	"code.cloudfoundry.org/smbdriver/driveradmin/driveradminlocal"
	"code.cloudfoundry.org/smbdriver/smbdriverfakes"
	"code.cloudfoundry.org/voldriver"
	"code.cloudfoundry.org/voldriver/driverhttp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Driver Admin Local", func() {
	var logger lager.Logger
	var ctx context.Context
	var env voldriver.Env
	var driverAdminLocal *driveradminlocal.DriverAdminLocal
	var err driveradmin.ErrorResponse

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("driveradminlocal")
		ctx = context.TODO()
		env = driverhttp.NewHttpDriverEnv(logger, ctx)
	})

	Context("created", func() {
		BeforeEach(func() {
			driverAdminLocal = driveradminlocal.NewDriverAdminLocal()
		})

		Describe("Evacuate", func() {
			JustBeforeEach(func() {
				err = driverAdminLocal.Evacuate(env)
			})
			Context("when the driver evacuates with no process set", func() {
				It("should fail", func() {
					Expect(err.Err).To(ContainSubstring("server process not found"))
				})
			})
			Context("when the driver evacuates with a process set", func() {
				var fakeProcess *smbdriverfakes.FakeProcess

				BeforeEach(func() {
					fakeProcess = &smbdriverfakes.FakeProcess{}
					driverAdminLocal.SetServerProc(fakeProcess)
				})

				It("should signal the process to terminate", func() {
					Expect(err.Err).To(BeEmpty())
					Expect(fakeProcess.SignalCallCount()).NotTo(Equal(0))
				})
				Context("when there is a drainable server registered", func() {
					var fakeDrainable *smbdriverfakes.FakeDrainable
					BeforeEach(func() {
						fakeDrainable = &smbdriverfakes.FakeDrainable{}
						driverAdminLocal.RegisterDrainable(fakeDrainable)
					})
					It("should drain", func() {
						Expect(fakeDrainable.DrainCallCount()).NotTo(Equal(0))
					})
				})

			})
		})

		Describe("Ping", func() {
			Context("when the driver pings", func() {
				BeforeEach(func() {
					err = driverAdminLocal.Ping(env)
				})

				It("should not fail", func() {
					Expect(err.Err).To(Equal(""))
				})
			})
		})
	})
})
