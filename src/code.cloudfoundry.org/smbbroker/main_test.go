package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	fuzz "github.com/google/gofuzz"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
	"github.com/pivotal-cf/brokerapi/v11/domain"
	"github.com/pivotal-cf/brokerapi/v11/domain/apiresponses"
	"github.com/tedsuo/ifrit"
	ginkgomon "github.com/tedsuo/ifrit/ginkgomon_v2"
)

var _ = Describe("smbbroker Main", func() {
	Context("Missing required args", func() {
		var process ifrit.Process

		It("shows usage when credhubURL is not provided", func() {
			var args []string

			volmanRunner := failRunner{
				Name:       "smbbroker",
				Command:    exec.Command(binaryPath, args...),
				StartCheck: "CredhubURL parameter must be provided.",
			}

			process = ifrit.Invoke(volmanRunner)
		})

		It("shows usage when servicesConfig is not provided", func() {
			args := []string{"-credhubURL", "credhub-url"}

			volmanRunner := failRunner{
				Name:       "smbbroker",
				Command:    exec.Command(binaryPath, args...),
				StartCheck: "servicesConfig parameter must be provided.",
			}

			process = ifrit.Invoke(volmanRunner)
		})

		AfterEach(func() {
			ginkgomon.Kill(process) // this is only if incorrect implementation leaves process running
		})
	})

	Context("credhub /info returns error", func() {
		var volmanRunner *ginkgomon.Runner
		var credhubServer *ghttp.Server

		DescribeTable("should log a helpful diagnostic error message ", func(statusCode int) {
			listenAddr := "0.0.0.0:" + strconv.Itoa(8999+GinkgoParallelProcess())

			credhubServer = ghttp.NewServer()
			credhubServer.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/info"),
				ghttp.RespondWith(statusCode, "", http.Header{"X-Squid-Err": []string{"some-error"}}),
			))

			var args []string
			args = append(args, "-listenAddr", listenAddr)
			args = append(args, "-credhubURL", credhubServer.URL())
			args = append(args, "-servicesConfig", "./default_services.json")

			volmanRunner = ginkgomon.New(ginkgomon.Config{
				Name:       "smbbroker",
				Command:    exec.Command(binaryPath, args...),
				StartCheck: "smbbroker.starting",
			})

			invoke := ifrit.Invoke(volmanRunner)
			defer ginkgomon.Kill(invoke)

			time.Sleep(2 * time.Second)
			Eventually(volmanRunner.ExitCode).Should(Equal(2))
			Eventually(volmanRunner.Buffer()).Should(gbytes.Say(fmt.Sprintf(".*Attempted to connect to credhub. Expected 200. Got %d.*X-Squid-Err:\\[some-error\\].*", statusCode)))

		},
			Entry("300", http.StatusMultipleChoices),
			Entry("400", http.StatusBadRequest),
			Entry("403", http.StatusForbidden),
			Entry("500", http.StatusInternalServerError))

		It("should timeout after 30 seconds", func() {
			listenAddr := "0.0.0.0:" + strconv.Itoa(8999+GinkgoParallelProcess())

			hangForMoreThan30SecondsHandler := func(resp http.ResponseWriter, req *http.Request) {
				time.Sleep(32 * time.Second)
			}

			credhubServer = ghttp.NewServer()
			credhubServer.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/info"),
				hangForMoreThan30SecondsHandler,
			))

			var args []string
			args = append(args, "-listenAddr", listenAddr)
			args = append(args, "-credhubURL", credhubServer.URL())
			args = append(args, "-servicesConfig", "./default_services.json")

			volmanRunner = ginkgomon.New(ginkgomon.Config{
				Name:       "smbbroker",
				Command:    exec.Command(binaryPath, args...),
				StartCheck: "smbbroker.starting",
			})

			invoke := ifrit.Invoke(volmanRunner)
			defer ginkgomon.Kill(invoke)

			Eventually(volmanRunner.ExitCode, "31s").Should(Equal(2))
			Eventually(volmanRunner.Buffer()).Should(gbytes.Say(".*Unable to connect to credhub."))
		})
	})

	Context("Has required args", func() {
		var (
			args               []string
			listenAddr         string
			username, password string
			planID             = "0da18102-48dc-46d0-98b3-7a4ff6dc9c54"
			serviceOfferingID  = "9db9cca4-8fd5-4b96-a4c7-0a48f47c3bad"
			serviceInstanceID  = "service-instance-id"
			volmanRunner       *ginkgomon.Runner
			process            ifrit.Process

			credhubServer *ghttp.Server
			uaaServer     *ghttp.Server
		)

		BeforeEach(func() {
			listenAddr = "0.0.0.0:" + strconv.Itoa(8999+GinkgoParallelProcess())
			username = "admin"
			password = "password"

			os.Setenv("USERNAME", username)
			os.Setenv("PASSWORD", password)

			credhubServer = ghttp.NewServer()
			uaaServer = ghttp.NewServer()

			infoResponse := credhubInfoResponse{
				AuthServer: credhubInfoResponseAuthServer{
					URL: "some-auth-server-url",
				},
			}

			credhubServer.AppendHandlers(ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/info"),
				ghttp.RespondWithJSONEncoded(http.StatusOK, infoResponse),
			), ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/info"),
				ghttp.RespondWithJSONEncoded(http.StatusOK, infoResponse),
			))

			args = append(args, "-listenAddr", listenAddr)
			args = append(args, "-credhubURL", credhubServer.URL())
			args = append(args, "-servicesConfig", "./default_services.json")
		})

		JustBeforeEach(func() {
			volmanRunner = ginkgomon.New(ginkgomon.Config{
				Name:       "smbbroker",
				Command:    exec.Command(binaryPath, args...),
				StartCheck: "started",
			})

			process = ginkgomon.Invoke(volmanRunner)
		})

		AfterEach(func() {
			ginkgomon.Kill(process)

			credhubServer.Close()
			uaaServer.Close()
		})

		httpDoWithAuth := func(method, endpoint string, body io.Reader) (*http.Response, error) {
			req, err := http.NewRequest(method, "http://"+listenAddr+endpoint, body)
			req.Header.Add("X-Broker-Api-Version", "2.14")
			Expect(err).NotTo(HaveOccurred())

			req.SetBasicAuth(username, password)
			return http.DefaultClient.Do(req)
		}

		It("should check for a proxy", func() {
			Eventually(volmanRunner.Buffer()).Should(gbytes.Say("no-proxy-found"))
		})

		It("should listen on the given address", func() {
			resp, err := httpDoWithAuth("GET", "/v2/catalog", nil)
			Expect(err).NotTo(HaveOccurred())

			Expect(resp.StatusCode).To(Equal(200))
		})

		It("should pass services config through to catalog", func() {
			resp, err := httpDoWithAuth("GET", "/v2/catalog", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))

			bytes, err := io.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())

			var catalog apiresponses.CatalogResponse
			err = json.Unmarshal(bytes, &catalog)
			Expect(err).NotTo(HaveOccurred())

			Expect(catalog.Services[0].ID).To(Equal(serviceOfferingID))
			Expect(catalog.Services[0].Name).To(Equal("smb"))
			Expect(catalog.Services[0].Plans[0].ID).To(Equal("0da18102-48dc-46d0-98b3-7a4ff6dc9c54"))
			Expect(catalog.Services[0].Plans[0].Name).To(Equal("Existing"))
			Expect(catalog.Services[0].Plans[0].Description).To(Equal("A preexisting share"))
		})

		Context("#provision", func() {

			BeforeEach(func() {
				infoResponse := credhubInfoResponse{
					AuthServer: credhubInfoResponseAuthServer{
						URL: uaaServer.URL(),
					},
				}

				uaaServer.AppendHandlers(ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/oauth/token"),
					ghttp.RespondWith(http.StatusOK, `{ "access_token" : "111", "refresh_token" : "", "token_type" : "" }`),
				))

				credhubServer.RouteToHandler("GET", "/info", ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/info"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, infoResponse),
				))

				credhubServer.RouteToHandler("GET", "/api/v1/data", ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/api/v1/data", fmt.Sprintf("current=true&name=%%2Fsmbbroker%%2F%s", serviceInstanceID)),
					ghttp.RespondWithJSONEncoded(http.StatusOK, "{}"),
				))

				credhubServer.RouteToHandler("GET", "/version", ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/version"),
					ghttp.RespondWith(http.StatusOK, `{ "version" : "0.0.0" }`),
				))

				credhubServer.RouteToHandler("PUT", "/api/v1/data", ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/api/v1/data"),
					ghttp.RespondWith(http.StatusCreated, `{ "type" : "json", "version_created_at" : "", "id" : "", "name" : "", "value" : { } }`),
				))
			})

			It("should respond with 200", func() {
				provisionDetailsJsons, err := json.Marshal(domain.ProvisionDetails{
					ServiceID:     serviceOfferingID,
					PlanID:        planID,
					RawParameters: json.RawMessage(`{"share": "sharevalue", "version": "1.0"}`),
				})

				Expect(err).NotTo(HaveOccurred())
				reader := strings.NewReader(string(provisionDetailsJsons))
				resp, err := httpDoWithAuth("PUT", fmt.Sprintf("/v2/service_instances/%s", serviceInstanceID), reader)

				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(201))
			})
		})

		Context("#bind", func() {
			var (
				bindingID = "456"
			)
			BeforeEach(func() {
				infoResponse := credhubInfoResponse{
					AuthServer: credhubInfoResponseAuthServer{
						URL: uaaServer.URL(),
					},
				}

				uaaServer.AppendHandlers(ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/oauth/token"),
					ghttp.RespondWith(http.StatusOK, `{ "access_token" : "111", "refresh_token" : "", "token_type" : "" }`),
				))

				credhubServer.RouteToHandler("GET", "/info", ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/info"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, infoResponse),
				))

				credhubServer.RouteToHandler("GET", "/api/v1/data", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if strings.Contains(r.URL.RawQuery, bindingID) {
						w.WriteHeader(404)
					} else if strings.Contains(r.URL.RawQuery, fmt.Sprintf("current=true&name=%%2Fsmbbroker%%2F%s", serviceInstanceID)) {
						_, err := w.Write([]byte(`{ "data" : [ { "type": "value", "version_created_at": "2019", "id": "1", "name": "/some-name", "value": { "ServiceFingerPrint": "foobar" } } ] }`))
						if err != nil {
							w.WriteHeader(500)
						}
					}
				}))

				credhubServer.RouteToHandler("GET", "/version", ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/version"),
					ghttp.RespondWith(http.StatusOK, `{ "version" : "0.0.0" }`),
				))

				credhubServer.RouteToHandler("PUT", "/api/v1/data", ghttp.CombineHandlers(
					ghttp.VerifyRequest("PUT", "/api/v1/data"),
					ghttp.RespondWith(http.StatusCreated, `{ "type" : "json", "version_created_at" : "", "id" : "", "name" : "", "value" : { } }`),
				))
			})

			Context("allowed parameters", func() {
				It("should accept the parameter", func() {
					rawParametersMap := map[string]string{
						"username":   "user",
						"password":   "foo",
						"mount":      "somemount",
						"readonly":   "true",
						"domain":     "foo",
						"mfsymlinks": "true",
					}

					rawParameters, err := json.Marshal(rawParametersMap)
					Expect(err).NotTo(HaveOccurred())
					provisionDetailsJsons, err := json.Marshal(domain.BindDetails{
						ServiceID:     serviceOfferingID,
						PlanID:        planID,
						AppGUID:       "222",
						RawParameters: rawParameters,
					})
					Expect(err).NotTo(HaveOccurred())
					reader := strings.NewReader(string(provisionDetailsJsons))
					endpoint := fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s", serviceInstanceID, bindingID)
					resp, err := httpDoWithAuth("PUT", endpoint, reader)

					Expect(err).NotTo(HaveOccurred())
					Expect(resp.StatusCode).To(Equal(201))
				})
			})

			Context("invalid mfsymlinks", func() {
				var (
					bindDetailJson []byte
					mfsymlinks     = ""
				)

				BeforeEach(func() {
					fuzz.New().Fuzz(&mfsymlinks)
					mfsymlinks = strings.ReplaceAll(mfsymlinks, "%", "")

					rawParametersMap := map[string]string{
						"mfsymlinks": mfsymlinks,
					}

					rawParameters, err := json.Marshal(rawParametersMap)
					Expect(err).NotTo(HaveOccurred())

					bindDetailJson, err = json.Marshal(domain.BindDetails{
						ServiceID:     serviceOfferingID,
						PlanID:        planID,
						AppGUID:       "222",
						RawParameters: rawParameters,
					})

					Expect(err).NotTo(HaveOccurred())
				})

				It("should respond with 400", func() {
					reader := strings.NewReader(string(bindDetailJson))
					endpoint := fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s", serviceInstanceID, bindingID)
					resp, err := httpDoWithAuth("PUT", endpoint, reader)

					Expect(err).NotTo(HaveOccurred())
					Expect(resp.StatusCode).To(Equal(400))

					expectedResponse := map[string]string{
						"description": fmt.Sprintf("- validation mount options failed: %s is not a valid value for mfsymlinks\n", mfsymlinks),
					}
					expectedJsonResponse, err := json.Marshal(expectedResponse)
					Expect(err).NotTo(HaveOccurred())

					responseBody, err := io.ReadAll(resp.Body)
					Expect(err).NotTo(HaveOccurred())
					Expect(string(responseBody)).To(MatchJSON(expectedJsonResponse))
				})
			})

			Context("versions", func() {
				DescribeTable("valid versions", func(version string) {
					rawParametersMap := map[string]string{
						"username": "user",
						"version":  version,
					}

					rawParameters, err := json.Marshal(rawParametersMap)
					Expect(err).NotTo(HaveOccurred())
					provisionDetailsJsons, err := json.Marshal(domain.BindDetails{
						ServiceID:     serviceOfferingID,
						PlanID:        planID,
						AppGUID:       "222",
						RawParameters: rawParameters,
					})
					Expect(err).NotTo(HaveOccurred())
					reader := strings.NewReader(string(provisionDetailsJsons))
					endpoint := fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s", serviceInstanceID, bindingID)
					resp, err := httpDoWithAuth("PUT", endpoint, reader)

					Expect(err).NotTo(HaveOccurred())
					Expect(resp.StatusCode).To(Equal(201))

				},
					Entry("version 1", "1.0"),
					Entry("version 2", "2.0"),
					Entry("version 2.1", "2.1"),
					Entry("version 3", "3.0"),
				)

				Context("invalid version", func() {
					var (
						bindDetailJson []byte
						version        = ""
					)

					BeforeEach(func() {
						fuzz.New().Fuzz(&version)
						version = strings.ReplaceAll(version, "%", "")

						rawParametersMap := map[string]string{
							"version": version,
						}

						rawParameters, err := json.Marshal(rawParametersMap)
						Expect(err).NotTo(HaveOccurred())

						bindDetailJson, err = json.Marshal(domain.BindDetails{
							ServiceID:     serviceOfferingID,
							PlanID:        planID,
							AppGUID:       "222",
							RawParameters: rawParameters,
						})

						Expect(err).NotTo(HaveOccurred())
					})

					It("should respond with 400", func() {
						reader := strings.NewReader(string(bindDetailJson))
						endpoint := fmt.Sprintf("/v2/service_instances/%s/service_bindings/%s", serviceInstanceID, bindingID)
						resp, err := httpDoWithAuth("PUT", endpoint, reader)

						Expect(err).NotTo(HaveOccurred())
						Expect(resp.StatusCode).To(Equal(400))

						expectedResponse := map[string]string{
							"description": fmt.Sprintf("- validation mount options failed: %s is not a valid version\n", version),
						}
						expectedJsonResponse, err := json.Marshal(expectedResponse)
						Expect(err).NotTo(HaveOccurred())

						responseBody, err := io.ReadAll(resp.Body)
						Expect(err).NotTo(HaveOccurred())
						Expect(string(responseBody)).To(MatchJSON(expectedJsonResponse))
					})
				})
			})
		})

		Context("#update", func() {
			It("should respond with a 422", func() {
				updateDetailsJson, err := json.Marshal(domain.UpdateDetails{
					ServiceID: serviceOfferingID,
				})
				Expect(err).NotTo(HaveOccurred())
				reader := strings.NewReader(string(updateDetailsJson))
				resp, err := httpDoWithAuth("PATCH", fmt.Sprintf("/v2/service_instances/%s", serviceInstanceID), reader)
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(422))

				responseBody, err := io.ReadAll(resp.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(responseBody)).To(ContainSubstring("this service does not support instance updates. Please delete your service instance and create a new one with updated configuration"))
			})
		})
	})
})

func (r failRunner) Run(sigChan <-chan os.Signal, ready chan<- struct{}) error {
	defer GinkgoRecover()

	allOutput := gbytes.NewBuffer()

	debugWriter := gexec.NewPrefixedWriter(
		fmt.Sprintf("\x1b[32m[d]\x1b[%s[%s]\x1b[0m ", r.AnsiColorCode, r.Name),
		GinkgoWriter,
	)

	var err error
	r.session, err = gexec.Start(
		r.Command,
		gexec.NewPrefixedWriter(
			fmt.Sprintf("\x1b[32m[o]\x1b[%s[%s]\x1b[0m ", r.AnsiColorCode, r.Name),
			io.MultiWriter(allOutput, GinkgoWriter),
		),
		gexec.NewPrefixedWriter(
			fmt.Sprintf("\x1b[91m[e]\x1b[%s[%s]\x1b[0m ", r.AnsiColorCode, r.Name),
			io.MultiWriter(allOutput, GinkgoWriter),
		),
	)

	Expect(err).ShouldNot(HaveOccurred())

	fmt.Fprintf(debugWriter, "spawned %s (pid: %d)\n", r.Command.Path, r.Command.Process.Pid)

	if r.sessionReady != nil {
		close(r.sessionReady)
	}

	startCheckDuration := r.StartCheckTimeout
	if startCheckDuration == 0 {
		startCheckDuration = 5 * time.Second
	}

	var startCheckTimeout <-chan time.Time
	if r.StartCheck != "" {
		startCheckTimeout = time.After(startCheckDuration)
	}

	detectStartCheck := allOutput.Detect(r.StartCheck)

	for {
		select {
		case <-detectStartCheck: // works even with empty string
			allOutput.CancelDetects()
			startCheckTimeout = nil
			detectStartCheck = nil
			close(ready)

		case <-startCheckTimeout:
			// clean up hanging process
			r.session.Kill().Wait()

			// fail to start
			return fmt.Errorf(
				"did not see %s in command's output within %s. full output:\n\n%s",
				r.StartCheck,
				startCheckDuration,
				string(allOutput.Contents()),
			)

		case signal := <-sigChan:
			r.session.Signal(signal)

		case <-r.session.Exited:
			if r.Cleanup != nil {
				r.Cleanup()
			}

			Expect(string(allOutput.Contents())).To(ContainSubstring(r.StartCheck))
			Expect(r.session.ExitCode()).To(Not(Equal(0)), "Expected process to exit with non-zero, got: 0")
			return nil
		}
	}
}

type credhubInfoResponse struct {
	AuthServer credhubInfoResponseAuthServer `json:"auth-server"`
}

type credhubInfoResponseAuthServer struct {
	URL string `json:"url"`
}

type failRunner struct {
	Command           *exec.Cmd
	Name              string
	AnsiColorCode     string
	StartCheck        string
	StartCheckTimeout time.Duration
	Cleanup           func()
	session           *gexec.Session
	sessionReady      chan struct{}
}
