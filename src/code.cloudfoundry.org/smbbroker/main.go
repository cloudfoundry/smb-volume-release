package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/debugserver"
	"code.cloudfoundry.org/existingvolumebroker"
	"code.cloudfoundry.org/existingvolumebroker/utils"
	"code.cloudfoundry.org/goshims/osshim"
	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/lager/v3/lagerflags"
	"code.cloudfoundry.org/service-broker-store/brokerstore"
	vmo "code.cloudfoundry.org/volume-mount-options"
	vmou "code.cloudfoundry.org/volume-mount-options/utils"
	"github.com/pivotal-cf/brokerapi/v11"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/http_server"
)

var atAddress = flag.String(
	"listenAddr",
	"0.0.0.0:8999",
	"host:port to serve service broker API",
)

var servicesConfig = flag.String(
	"servicesConfig",
	"",
	"[REQUIRED] - Path to services config to register with cloud controller",
)

var credhubURL = flag.String(
	"credhubURL",
	"",
	"(optional) CredHub server URL when using CredHub to store broker state",
)

var credhubCACertPath = flag.String(
	"credhubCACertPath",
	"",
	"(optional) Path to CA Cert for CredHub",
)

var uaaClientID = flag.String(
	"uaaClientID",
	"",
	"(optional) UAA client ID when using CredHub to store broker state",
)

var uaaClientSecret = flag.String(
	"uaaClientSecret",
	"",
	"(optional) UAA client secret when using CredHub to store broker state",
)

var uaaCACertPath = flag.String(
	"uaaCACertPath",
	"",
	"(optional) Path to CA Cert for UAA used for CredHub authorization",
)

var storeID = flag.String(
	"storeID",
	"smbbroker",
	"(optional) Store ID used to namespace instance details and bindings (credhub only)",
)

var (
	username string
	password string
)

func main() {
	parseCommandLine()
	parseEnvironment()

	checkParams()

	logger, logSink := newLogger()
	logger.Info("starting")
	defer logger.Info("ends")

	verifyCredhubIsReachable(logger)

	server := createServer(logger)

	if dbgAddr := debugserver.DebugAddress(flag.CommandLine); dbgAddr != "" {
		server = utils.ProcessRunnerFor(grouper.Members{
			{Name: "debug-server", Runner: debugserver.Runner(dbgAddr, logSink)},
			{Name: "broker-api", Runner: server},
		})
	}

	process := ifrit.Invoke(server)
	logger.Info("started")
	utils.UntilTerminated(logger, process)
}

func verifyCredhubIsReachable(logger lager.Logger) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	configureCACert(logger, client)
	utils.IsThereAProxy(&osshim.OsShim{}, logger)

	resp, err := client.Get(*credhubURL + "/info")
	if err != nil {
		logger.Fatal("Unable to connect to credhub", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.Fatal(fmt.Sprintf("Attempted to connect to credhub. Expected 200. Got %d", resp.StatusCode), nil, lager.Data{"response_headers": fmt.Sprintf("%v", resp.Header)})
	}
}

func configureCACert(logger lager.Logger, client *http.Client) {
	if *credhubCACertPath != "" {
		certpool := x509.NewCertPool()

		certPEM, err := os.ReadFile(*credhubCACertPath)
		if err != nil {
			logger.Fatal("reading credhub ca cert path", err)
		}

		ok := certpool.AppendCertsFromPEM(certPEM)
		if !ok {
			logger.Fatal("appending certs from PEM", err)
		}
		// disable "G402 (CWE-295): TLS MinVersion too low. (Confidence: HIGH, Severity: HIGH)"
		// #nosec G402 - Enforcing a MinVersion for TLS could break numerous existing systems
		clientTLSConf := &tls.Config{
			RootCAs: certpool,
		}

		transport := &http.Transport{
			TLSClientConfig: clientTLSConf,
		}

		client.Transport = transport
	}
}

func parseCommandLine() {
	lagerflags.AddFlags(flag.CommandLine)
	debugserver.AddFlags(flag.CommandLine)
	flag.Parse()
}

func parseEnvironment() {
	username, _ = os.LookupEnv("USERNAME")
	password, _ = os.LookupEnv("PASSWORD")
	uaaClientSecretString, _ := os.LookupEnv("UAA_CLIENT_SECRET")
	if uaaClientSecretString != "" {
		uaaClientSecret = &uaaClientSecretString
	}
	uaaClientIDString, _ := os.LookupEnv("UAA_CLIENT_ID")
	if uaaClientIDString != "" {
		uaaClientID = &uaaClientIDString
	}
}

func checkParams() {
	if *credhubURL == "" {
		fmt.Fprint(os.Stderr, "\nERROR: CredhubURL parameter must be provided.\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if *servicesConfig == "" {
		fmt.Fprint(os.Stderr, "\nERROR: servicesConfig parameter must be provided.\n\n")
		flag.Usage()
		os.Exit(1)
	}
}

func newLogger() (lager.Logger, *lager.ReconfigurableSink) {
	lagerConfig := lagerflags.ConfigFromFlags()
	lagerConfig.RedactSecrets = true

	return lagerflags.NewFromConfig("smbbroker", lagerConfig)
}

func createServer(logger lager.Logger) ifrit.Runner {
	var credhubCACert string
	if *credhubCACertPath != "" {
		b, err := os.ReadFile(*credhubCACertPath)
		if err != nil {
			logger.Fatal("cannot-read-credhub-ca-cert", err, lager.Data{"path": *credhubCACertPath})
		}
		credhubCACert = string(b)
	}

	var uaaCACert string
	if *uaaCACertPath != "" {
		b, err := os.ReadFile(*uaaCACertPath)
		if err != nil {
			logger.Fatal("cannot-read-credhub-ca-cert", err, lager.Data{"path": *uaaCACertPath})
		}
		uaaCACert = string(b)
	}

	store := brokerstore.NewStore(
		logger,
		*credhubURL,
		credhubCACert,
		*uaaClientID,
		*uaaClientSecret,
		uaaCACert,
		*storeID,
	)

	versionValidator := vmo.UserOptsValidationFunc(validateVersion)
	symlinksValidator := vmo.UserOptsValidationFunc(validateMfsymlinks)

	configMask, err := vmo.NewMountOptsMask(
		strings.Split(AllowedOptions(), ","),
		vmou.ParseOptionStringToMap("", ":"),
		map[string]string{
			"readonly": "ro",
			"share":    "source",
		},
		[]string{},
		[]string{"source"},
		versionValidator, symlinksValidator,
	)
	if err != nil {
		logger.Fatal("creating-config-mask-error", err)
	}

	logger.Debug("smbbroker-startup-config", lager.Data{"config-mask": configMask})

	services, err := NewServicesFromConfig(*servicesConfig)
	if err != nil {
		logger.Fatal("loading-services-config-error", err)
	}

	serviceBroker := existingvolumebroker.New(
		existingvolumebroker.BrokerTypeSMB,
		logger,
		services,
		&osshim.OsShim{},
		clock.NewClock(),
		store,
		configMask,
	)

	credentials := brokerapi.BrokerCredentials{Username: username, Password: password}
	handler := brokerapi.New(serviceBroker, slog.New(lager.NewHandler(lager.NewLogger("broker-api"))), credentials)

	return http_server.New(*atAddress, handler)
}

func validateMfsymlinks(key string, val string) error {

	if key != "mfsymlinks" {
		return nil
	}

	if val == "true" {
		return nil
	}

	return fmt.Errorf("%s is not a valid value for mfsymlinks", val)
}

func validateVersion(key string, val string) error {
	validVersions := []string{"1.0", "2.0", "2.1", "3.0", "3.1.1"}

	if key != "version" {
		return nil
	}

	for _, validVersion := range validVersions {
		if val == validVersion {
			return nil
		}
	}

	return fmt.Errorf("%s is not a valid version", val)
}
