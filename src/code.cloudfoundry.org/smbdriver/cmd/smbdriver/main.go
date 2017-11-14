package main

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strconv"

	cf_http "code.cloudfoundry.org/cfhttp"
	cf_debug_server "code.cloudfoundry.org/debugserver"

	"code.cloudfoundry.org/goshims/filepathshim"
	"code.cloudfoundry.org/goshims/ioutilshim"
	"code.cloudfoundry.org/goshims/osshim"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/nfsdriver"

	"code.cloudfoundry.org/smbdriver"

	"code.cloudfoundry.org/lager/lagerflags"
	"code.cloudfoundry.org/smbdriver/driveradmin/driveradminhttp"
	"code.cloudfoundry.org/smbdriver/driveradmin/driveradminlocal"
	"code.cloudfoundry.org/voldriver"
	"code.cloudfoundry.org/voldriver/driverhttp"
	"code.cloudfoundry.org/voldriver/invoker"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/http_server"
	"github.com/tedsuo/ifrit/sigmon"
)

var atPort = flag.Int(
	"listenPort",
	8589,
	"Port to serve volume management functions. Listen address is always 127.0.0.1",
)

var adminPort = flag.Int(
	"adminPort",
	8590,
	"Port to serve process admin functions",
)

var driversPath = flag.String(
	"driversPath",
	"",
	"[REQUIRED] - Path to directory where drivers are installed",
)

var transport = flag.String(
	"transport",
	"tcp",
	"Transport protocol to transmit HTTP over",
)

var mountDir = flag.String(
	"mountDir",
	"/tmp/volumes",
	"Path to directory where fake volumes are created",
)

var requireSSL = flag.Bool(
	"requireSSL",
	false,
	"Whether the fake driver should require ssl-secured communication",
)

var caFile = flag.String(
	"caFile",
	"",
	"(optional) - The certificate authority public key file to use with ssl authentication",
)

var certFile = flag.String(
	"certFile",
	"",
	"(optional) - The public key file to use with ssl authentication",
)

var keyFile = flag.String(
	"keyFile",
	"",
	"(optional) - The private key file to use with ssl authentication",
)
var clientCertFile = flag.String(
	"clientCertFile",
	"",
	"(optional) - The public key file to use with client ssl authentication",
)

var clientKeyFile = flag.String(
	"clientKeyFile",
	"",
	"(optional) - The private key file to use with client ssl authentication",
)

var insecureSkipVerify = flag.Bool(
	"insecureSkipVerify",
	false,
	"Whether SSL communication should skip verification of server IP addresses in the certificate",
)

var mountFlagAllowed = flag.String(
	"mountFlagAllowed",
	"",
	"[REQUIRED] - This is a comma separted list of parameters allowed to be send in extra config. Each of this parameters can be specify by brokers",
)

var mountFlagDefault = flag.String(
	"mountFlagDefault",
	"",
	"(optional) - This is a comma separted list of like params:value. This list specify default value of parameters. If parameters has default value and is not in allowed list, this default value become a forced value who's cannot be override",
)

const fsType = "cifs"
const listenAddress = "127.0.0.1"

func main() {
	parseCommandLine()

	var localDriverServer ifrit.Runner

	logger, logTap := newLogger()
	logger.Info("start")
	defer logger.Info("end")

	config := smbdriver.NewSmbConfig()
	config.ReadConf(*mountFlagAllowed, *mountFlagDefault, []string{"username", "password"})

	mounter := smbdriver.NewSmbMounter(
		invoker.NewRealInvoker(),
		&osshim.OsShim{},
		&ioutilshim.IoutilShim{},
		config,
	)

	client := nfsdriver.NewNfsDriver(
		logger,
		&osshim.OsShim{},
		&filepathshim.FilepathShim{},
		&ioutilshim.IoutilShim{},
		*mountDir,
		mounter,
	)

	if *transport == "tcp" {
		localDriverServer = createSmbDriverServer(logger, client, *atPort, *driversPath, false)
	} else if *transport == "tcp-json" {
		localDriverServer = createSmbDriverServer(logger, client, *atPort, *driversPath, true)
	} else {
		localDriverServer = createSmbDriverUnixServer(logger, client, *atPort)
	}

	servers := grouper.Members{
		{"localdriver-server", localDriverServer},
	}

	if dbgAddr := cf_debug_server.DebugAddress(flag.CommandLine); dbgAddr != "" {
		servers = append(grouper.Members{
			{"debug-server", cf_debug_server.Runner(dbgAddr, logTap)},
		}, servers...)
	}

	adminClient := driveradminlocal.NewDriverAdminLocal()
	adminHandler, _ := driveradminhttp.NewHandler(logger, adminClient)
	// TODO handle error
	adminAddress := listenAddress + ":" + strconv.Itoa(*adminPort)
	adminServer := http_server.New(adminAddress, adminHandler)

	servers = append(grouper.Members{
		{"driveradmin", adminServer},
	}, servers...)

	process := ifrit.Invoke(processRunnerFor(servers))
	logger.Info("started")

	adminClient.SetServerProc(process)
	adminClient.RegisterDrainable(client)

	untilTerminated(logger, process)
}

func exitOnFailure(logger lager.Logger, err error) {
	if err != nil {
		logger.Fatal("fatal-err-aborting", err)
	}
}

func untilTerminated(logger lager.Logger, process ifrit.Process) {
	err := <-process.Wait()
	exitOnFailure(logger, err)
}

func processRunnerFor(servers grouper.Members) ifrit.Runner {
	return sigmon.New(grouper.NewOrdered(os.Interrupt, servers))
}

func createSmbDriverServer(logger lager.Logger, client voldriver.Driver, atPort int, driversPath string, jsonSpec bool) ifrit.Runner {
	atAddress := listenAddress + ":" + strconv.Itoa(atPort)
	advertisedUrl := "http://" + atAddress
	logger.Info("writing-spec-file", lager.Data{"location": driversPath, "name": "smbdriver", "address": advertisedUrl})
	if jsonSpec {
		driverJsonSpec := voldriver.DriverSpec{Name: "smbdriver", Address: advertisedUrl}

		if *requireSSL {
			absCaFile, err := filepath.Abs(*caFile)
			exitOnFailure(logger, err)
			absClientCertFile, err := filepath.Abs(*clientCertFile)
			exitOnFailure(logger, err)
			absClientKeyFile, err := filepath.Abs(*clientKeyFile)
			exitOnFailure(logger, err)
			driverJsonSpec.TLSConfig = &voldriver.TLSConfig{InsecureSkipVerify: *insecureSkipVerify, CAFile: absCaFile, CertFile: absClientCertFile, KeyFile: absClientKeyFile}
			driverJsonSpec.Address = "https://" + atAddress
		}

		jsonBytes, err := json.Marshal(driverJsonSpec)

		exitOnFailure(logger, err)
		err = voldriver.WriteDriverSpec(logger, driversPath, "smbdriver", "json", jsonBytes)
		exitOnFailure(logger, err)
	} else {
		err := voldriver.WriteDriverSpec(logger, driversPath, "smbdriver", "spec", []byte(advertisedUrl))
		exitOnFailure(logger, err)
	}

	handler, err := driverhttp.NewHandler(logger, client)
	exitOnFailure(logger, err)

	var server ifrit.Runner
	if *requireSSL {
		tlsConfig, err := cf_http.NewTLSConfig(*certFile, *keyFile, *caFile)
		if err != nil {
			logger.Fatal("tls-configuration-failed", err)
		}
		server = http_server.NewTLSServer(atAddress, handler, tlsConfig)
	} else {
		server = http_server.New(atAddress, handler)
	}

	return server
}

func createSmbDriverUnixServer(logger lager.Logger, client voldriver.Driver, atPort int) ifrit.Runner {
	atAddress := listenAddress + ":" + strconv.Itoa(atPort)
	handler, err := driverhttp.NewHandler(logger, client)
	exitOnFailure(logger, err)
	return http_server.NewUnixServer(atAddress, handler)
}

func newLogger() (lager.Logger, *lager.ReconfigurableSink) {
	sink, err := lager.NewRedactingWriterSink(os.Stdout, lager.DEBUG, []string{"[Pp]wd", "[Pp]ass", "args"}, nil)
	if err != nil {
		panic(err)
	}
	logger, reconfigurableSink := lagerflags.NewFromSink("smb-driver-server", sink)
	return logger, reconfigurableSink
}

func parseCommandLine() {
	lagerflags.AddFlags(flag.CommandLine)
	cf_debug_server.AddFlags(flag.CommandLine)
	flag.Parse()
}
