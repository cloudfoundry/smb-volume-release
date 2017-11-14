package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"code.cloudfoundry.org/azurefilebroker/azurefilebroker"
	"code.cloudfoundry.org/azurefilebroker/utils"
	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/debugserver"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagerflags"

	"github.com/pivotal-cf/brokerapi"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/http_server"
)

var atAddress = flag.String(
	"listenAddr",
	"0.0.0.0:9000",
	"host:port to serve service broker API",
)

var serviceName = flag.String(
	"serviceName",
	"smbvolume",
	"Name of the service to register with cloud controller",
)

var serviceID = flag.String(
	"serviceID",
	"06948cb0-cad7-4buh-leba-9ed8b5c345a0",
	"ID of the service to register with cloud controller",
)

var environment = flag.String(
	"environment",
	"Preexisting",
	"The environment. `Preexisting` or the environment for Azure Management Service: `AzureCloud`, `AzureChinaCloud`, `AzureUSGovernment` or `AzureGermanCloud`",
)

// DB
var dbDriver = flag.String(
	"dbDriver",
	"",
	"[REQUIRED] - Database driver name when using SQL to store broker state. `mssql` or `mysql`",
)

var cfServiceName = flag.String(
	"cfServiceName",
	"",
	"(optional) - For CF pushed apps, the service name in VCAP_SERVICES where we should find database credentials. If this option is set, all db parameters will be extracted from the service binding except dbCACert and hostNameInCertificate",
)

var dbHostname = flag.String(
	"dbHostname",
	"",
	"(optional) - Database hostname when using SQL to store broker state",
)

var dbPort = flag.String(
	"dbPort",
	"",
	"(optional) - Database port when using SQL to store broker state",
)

var dbName = flag.String(
	"dbName",
	"",
	"(optional) - Database name when using SQL to store broker state",
)

var hostNameInCertificate = flag.String(
	"hostNameInCertificate",
	"",
	"(optional) - The Common Name (CN) in the server certificate. For Azure SQL service or Azure MySQL service, please see more details in README.md",
)

var dbCACert = flag.String(
	"dbCACert",
	"",
	"(optional) - CA Cert to verify SSL connection.",
)

// Bind
var allowedOptions = flag.String(
	"allowedOptions",
	"share,uid,gid,file_mode,dir_mode,readonly,vers,mount,domain,username,password,sec",
	"A comma separated list of parameters allowed to be set in during bind operations",
)

var defaultOptions = flag.String(
	"defaultOptions",
	"",
	"A comma separated list of defaults specified as param:value. If a parameter has a default value and is not in the allowed list, this default value becomes a fixed value that cannot be overridden",
)

// Azure
var tenantID = flag.String(
	"tenantID",
	"",
	"(optional) - Required for Azure Management Service. The tenant id for your service principal",
)

var clientID = flag.String(
	"clientID",
	"",
	"(optional) - Required for Azure Management Service. The client id for your service principal",
)

var clientSecret = flag.String(
	"clientSecret",
	"",
	"(optional) - Required for Azure Management Service. The client secret for your service principal",
)

var defaultSubscriptionID = flag.String(
	"defaultSubscriptionID",
	"",
	"(optional) - The default Azure Subscription id to use for storage accounts",
)

var defaultResourceGroupName = flag.String(
	"defaultResourceGroupName",
	"",
	"(optional) - The default resource group name to use for storage accounts",
)

var defaultLocation = flag.String(
	"defaultLocation",
	"",
	"(optional) - The default location to use for creating storage accounts",
)

var allowCreateStorageAccount = flag.Bool(
	"allowCreateStorageAccount",
	true,
	"Allow Broker to create storage accounts",
)

var allowCreateFileShare = flag.Bool(
	"allowCreateFileShare",
	true,
	"Allow Broker to create file shares",
)

var allowDeleteStorageAccount = flag.Bool(
	"allowDeleteStorageAccount",
	false,
	"Allow Broker to delete storage accounts which are created by Broker",
)

var allowDeleteFileShare = flag.Bool(
	"allowDeleteFileShare",
	false,
	"Allow Broker to delete file shares which are created by Broker",
)

// AzureStack
// TBD: AzureStack DOES NOT support file service now. Keep these for future.
var azureStackDomain = flag.String(
	"azureStackDomain",
	"",
	"Required when environment is AzureStack. The domain for your AzureStack deployment",
)

var azureStackAuthentication = flag.String(
	"azureStackAuthentication",
	"",
	"Required when environment is AzureStack. The authentication type for your AzureStack deployment. AzureAD, AzureStackAD or AzureStack",
)

var azureStackResource = flag.String(
	"azureStackResource",
	"",
	"Required when environment is AzureStack. The token resource for your AzureStack deployment",
)

var azureStackEndpointPrefix = flag.String(
	"azureStackEndpointPrefix",
	"",
	"Required when environment is AzureStack. The endpoint prefix for your AzureStack deployment",
)

var (
	username   string
	password   string
	dbUsername string
	dbPassword string
)

func main() {
	parseCommandLine()
	parseEnvironment()

	checkParams()

	sink, err := lager.NewRedactingWriterSink(os.Stdout, lager.INFO, nil, nil)
	if err != nil {
		panic(err)
	}
	logger, logSink := lagerflags.NewFromSink("azurefilebroker", sink)
	logger.Info("starting")
	defer logger.Info("end")

	server := createServer(logger)

	if dbgAddr := debugserver.DebugAddress(flag.CommandLine); dbgAddr != "" {
		server = utils.ProcessRunnerFor(grouper.Members{
			{"debug-server", debugserver.Runner(dbgAddr, logSink)},
			{"broker-api", server},
		})
	}

	process := ifrit.Invoke(server)
	logger.Info("started")
	utils.UntilTerminated(logger, process)
}

func parseCommandLine() {
	lagerflags.AddFlags(flag.CommandLine)
	debugserver.AddFlags(flag.CommandLine)
	flag.Parse()
}

func parseEnvironment() {
	var ok bool
	username, ok = os.LookupEnv("USERNAME")
	if !ok {
		// To support automatic broker registration with the Cloud Controller
		username, _ = os.LookupEnv("SECURITY_USER_NAME")
	}
	password, ok = os.LookupEnv("PASSWORD")
	if !ok {
		// To support automatic broker registration with the Cloud Controller
		password, _ = os.LookupEnv("SECURITY_USER_PASSWORD")
	}
	dbUsername, _ = os.LookupEnv("DBUSERNAME")
	dbPassword, _ = os.LookupEnv("DBPASSWORD")
}

func checkParams() {
	if *dbDriver == "" {
		fmt.Fprint(os.Stderr, "\nERROR: dbDriver parameter is required.\n\n")
		flag.Usage()
		os.Exit(1)
	}
	if username == "" || password == "" {
		fmt.Fprint(os.Stderr, "\nERROR: Both USERNAME and PASSWORD environment are required.\n\n")
		os.Exit(1)
	}
}

func parseVcapServices(logger lager.Logger) {
	// populate db parameters from VCAP_SERVICES and pitch a fit if there isn't one.
	services, hasValue := os.LookupEnv("VCAP_SERVICES")
	if !hasValue {
		logger.Fatal("missing-vcap-services-environment", errors.New("missing VCAP_SERVICES environment"))
	}

	servicesValues := map[string][]interface{}{}
	err := json.Unmarshal([]byte(services), &servicesValues)
	if err != nil {
		logger.Fatal("json-unmarshal", err)
	}

	dbServiceValues, ok := servicesValues[*cfServiceName]
	if !ok {
		logger.Fatal("missing-service-binding", errors.New("VCAP_SERVICES missing specified db service"), lager.Data{"servicesValues": servicesValues})
	}

	dbService := dbServiceValues[0].(map[string]interface{})

	credentials := dbService["credentials"].(map[string]interface{})
	logger.Debug("credentials-parsed", lager.Data{"credentials": credentials})

	dbUsername = credentials["username"].(string)
	dbPassword = credentials["password"].(string)
	*dbHostname = credentials["hostname"].(string)
	*dbPort = fmt.Sprintf("%.0f", credentials["port"].(float64))
	*dbName = credentials["name"].(string)
}

func createServer(logger lager.Logger) ifrit.Runner {
	// if we are CF pushed
	if *cfServiceName != "" {
		parseVcapServices(logger)
	}

	store := azurefilebroker.NewStore(logger, *dbDriver, dbUsername, dbPassword, *dbHostname, *dbPort, *dbName, *dbCACert, *hostNameInCertificate)

	mount := azurefilebroker.NewAzurefilebrokerMountConfig()
	mount.ReadConf(*allowedOptions, *defaultOptions)
	logger.Info("createServer.mount", lager.Data{
		"Allowed": mount.Allowed,
		"Forced":  mount.Forced,
		"Options": mount.Options,
	})

	azureConfig := azurefilebroker.NewAzureConfig(*environment, *tenantID, *clientID, *clientSecret, *defaultSubscriptionID, *defaultResourceGroupName, *defaultLocation)
	logger.Info("createServer.cloud.azureConfig", lager.Data{
		"Environment":              azureConfig.Environment,
		"TenanID":                  azureConfig.TenanID,
		"ClientID":                 azureConfig.ClientID,
		"DefaultSubscriptionID":    azureConfig.DefaultSubscriptionID,
		"DefaultResourceGroupName": azureConfig.DefaultResourceGroupName,
		"DefaultLocation":          azureConfig.DefaultLocation,
	})
	controlConfig := azurefilebroker.NewControlConfig(*allowCreateStorageAccount, *allowCreateFileShare, *allowDeleteStorageAccount, *allowDeleteFileShare)
	logger.Info("createServer.cloud.controlConfig", lager.Data{
		"AllowCreateStorageAccount": controlConfig.AllowCreateStorageAccount,
		"AllowCreateFileShare":      controlConfig.AllowCreateFileShare,
		"AllowDeleteStorageAccount": controlConfig.AllowDeleteStorageAccount,
		"AllowDeleteFileShare":      controlConfig.AllowDeleteFileShare,
	})
	azureStackConfig := azurefilebroker.NewAzureStackConfig(*azureStackDomain, *azureStackAuthentication, *azureStackResource, *azureStackEndpointPrefix)
	logger.Info("createServer.cloud.azureStackConfig", lager.Data{
		"AzureStackAuthentication": azureStackConfig.AzureStackAuthentication,
		"AzureStackDomain":         azureStackConfig.AzureStackDomain,
		"AzureStackEndpointPrefix": azureStackConfig.AzureStackEndpointPrefix,
		"AzureStackResource":       azureStackConfig.AzureStackResource,
	})
	cloud := azurefilebroker.NewAzurefilebrokerCloudConfig(azureConfig, controlConfig, azureStackConfig)

	err := cloud.Validate()
	if err != nil {
		logger.Fatal("createServer.validate-cloud-config", err)
	}

	config := azurefilebroker.NewAzurefilebrokerConfig(mount, cloud)

	serviceBroker := azurefilebroker.New(logger, *serviceName, *serviceID, clock.NewClock(), store, config)

	credentials := brokerapi.BrokerCredentials{Username: username, Password: password}
	handler := brokerapi.New(serviceBroker, logger.Session("broker-api"), credentials)

	return http_server.New(*atAddress, handler)
}
