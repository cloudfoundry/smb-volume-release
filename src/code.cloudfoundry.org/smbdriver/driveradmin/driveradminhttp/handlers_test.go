package driveradminhttp_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"code.cloudfoundry.org/lager/v3/lagertest"
	"code.cloudfoundry.org/smbdriver/driveradmin"
	"code.cloudfoundry.org/smbdriver/driveradmin/driveradminhttp"
	"code.cloudfoundry.org/smbdriver/smbdriverfakes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Volman Driver Handlers", func() {
	Context("when generating http handlers", func() {
		var testLogger = lagertest.NewTestLogger("HandlersTest")

		It("should produce a handler with an evacuate route", func() {
			By("faking out the driver")
			driverAdmin := &smbdriverfakes.FakeDriverAdmin{}
			driverAdmin.EvacuateReturns(driveradmin.ErrorResponse{})
			handler, err := driveradminhttp.NewHandler(testLogger, driverAdmin)
			Expect(err).NotTo(HaveOccurred())

			By("then fake serving the response using the handler")
			route, found := driveradmin.Routes.FindRouteByName(driveradmin.EvacuateRoute)
			Expect(found).To(BeTrue())

			path := fmt.Sprintf("http://0.0.0.0%s", route.Path)
			httpRequest, err := http.NewRequest("GET", path, nil)
			Expect(err).NotTo(HaveOccurred())

			httpResponseRecorder := httptest.NewRecorder()
			handler.ServeHTTP(httpResponseRecorder, httpRequest)

			By("then deserialing the HTTP response")
			response := driveradmin.ErrorResponse{}
			body, err := io.ReadAll(httpResponseRecorder.Body)
			Expect(err).NotTo(HaveOccurred())
			err = json.Unmarshal(body, &response)

			By("then expecting correct JSON conversion")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Err).Should(BeEmpty())
		})

		It("should produce a handler with an ping route", func() {
			By("faking out the driver")
			driverAdmin := &smbdriverfakes.FakeDriverAdmin{}
			driverAdmin.PingReturns(driveradmin.ErrorResponse{})
			handler, err := driveradminhttp.NewHandler(testLogger, driverAdmin)
			Expect(err).NotTo(HaveOccurred())

			By("then fake serving the response using the handler")
			route, found := driveradmin.Routes.FindRouteByName(driveradmin.PingRoute)
			Expect(found).To(BeTrue())

			path := fmt.Sprintf("http://0.0.0.0%s", route.Path)
			httpRequest, err := http.NewRequest("GET", path, nil)
			Expect(err).NotTo(HaveOccurred())

			httpResponseRecorder := httptest.NewRecorder()
			handler.ServeHTTP(httpResponseRecorder, httpRequest)

			By("then deserialing the HTTP response")
			response := driveradmin.ErrorResponse{}
			body, err := io.ReadAll(httpResponseRecorder.Body)
			Expect(err).NotTo(HaveOccurred())
			err = json.Unmarshal(body, &response)

			By("then expecting correct JSON conversion")
			Expect(err).ToNot(HaveOccurred())
			Expect(response.Err).Should(BeEmpty())
		})
	})
})
