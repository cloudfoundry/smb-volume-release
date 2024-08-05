package main

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strconv"

	cf_debug_server "code.cloudfoundry.org/debugserver"
	"code.cloudfoundry.org/dockerdriver"
	"code.cloudfoundry.org/dockerdriver/driverhttp"
	"code.cloudfoundry.org/goshims/bufioshim"
	"code.cloudfoundry.org/goshims/filepathshim"
	"code.cloudfoundry.org/goshims/ioutilshim"
	"code.cloudfoundry.org/goshims/osshim"
	"code.cloudfoundry.org/goshims/timeshim"
	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/lager/v3/lagerflags"
	"code.cloudfoundry.org/smbdriver"
	"code.cloudfoundry.org/smbdriver/driveradmin/driveradminhttp"
	"code.cloudfoundry.org/smbdriver/driveradmin/driveradminlocal"
	"code.cloudfoundry.org/tlsconfig"
	"code.cloudfoundry.org/volumedriver"
	"code.cloudfoundry.org/volumedriver/invoker"
	"code.cloudfoundry.org/volumedriver/mountchecker"
	"code.cloudfoundry.org/volumedriver/oshelper"
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

// The forceNoserverino flag was added on 2023-04-15.
//
// We had seen a large deployment in which upgrading from xenial to jammy
// stemcells caused all apps using SMB mounts to fail with "Stale file handle"
// errors. This turned out to be because the SMB server was suggesting inode
// numbers, instead of allowing the client to generate temporary inode numbers.
//
// The fix was to re-bind the SMB service with the mount parameter
// "noserverino". This flag was intended to allow the platform operator to
// apply that fix across the whole foundation, rather than relying on
// application authors to re-bind their SMB services.
var forceNoserverino = flag.Bool(
	"forceNoserverino",
	false,
	"Force all SMB mounts to use the 'noserverino' mount flag, regardless of what the service binding asks for",
)

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
var forceNoDfs = flag.Bool(
	"forceNoDfs",
	false,
	"Force all smb mounts to use the 'nodfs' mount flag, regardless of what the service binding asks for",
)

const listenAddress = "127.0.0.1"

func main() {
	parseCommandLine()

	var smbDriverServer ifrit.Runner

	logger, logSink := newLogger()
	logger.Info("start")
	defer logger.Info("end")

	configMask, err := smbdriver.NewSmbVolumeMountMask()
	exitOnFailure(logger, err)

	mounter := smbdriver.NewSmbMounter(
		invoker.NewProcessGroupInvoker(),
		&osshim.OsShim{},
		&ioutilshim.IoutilShim{},
		configMask,
		*forceNoserverino,
		*forceNoDfs,
	)

	client := volumedriver.NewVolumeDriver(
		logger,
		&osshim.OsShim{},
		&filepathshim.FilepathShim{},
		&ioutilshim.IoutilShim{},
		&timeshim.TimeShim{},
		mountchecker.NewChecker(&bufioshim.BufioShim{}, &osshim.OsShim{}),
		*mountDir,
		mounter,
		oshelper.NewOsHelper(),
	)

	if *transport == "tcp" {
		smbDriverServer = createSmbDriverServer(logger, client, *atPort, *driversPath, false)
	} else if *transport == "tcp-json" {
		smbDriverServer = createSmbDriverServer(logger, client, *atPort, *driversPath, true)
	} else {
		smbDriverServer = createSmbDriverUnixServer(logger, client, *atPort)
	}

	servers := grouper.Members{
		{Name: "smbdriver-server", Runner: smbDriverServer},
	}

	if dbgAddr := cf_debug_server.DebugAddress(flag.CommandLine); dbgAddr != "" {
		servers = append(grouper.Members{
			{Name: "debug-server", Runner: cf_debug_server.Runner(dbgAddr, logSink)},
		}, servers...)
	}

	adminClient := driveradminlocal.NewDriverAdminLocal()
	adminHandler, _ := driveradminhttp.NewHandler(logger, adminClient)
	adminAddress := listenAddress + ":" + strconv.Itoa(*adminPort)
	adminServer := http_server.New(adminAddress, adminHandler)

	servers = append(grouper.Members{
		{Name: "driveradmin", Runner: adminServer},
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

func createSmbDriverServer(logger lager.Logger, client dockerdriver.Driver, atPort int, driversPath string, jsonSpec bool) ifrit.Runner {
	atAddress := listenAddress + ":" + strconv.Itoa(atPort)
	advertisedUrl := "http://" + atAddress
	logger.Info("writing-spec-file", lager.Data{"location": driversPath, "name": "smbdriver", "address": advertisedUrl, "unique-volume-ids": true})
	if jsonSpec {
		driverJsonSpec := dockerdriver.DriverSpec{Name: "smbdriver", Address: advertisedUrl, UniqueVolumeIds: true}

		if *requireSSL {
			absCaFile, err := filepath.Abs(*caFile)
			exitOnFailure(logger, err)
			absClientCertFile, err := filepath.Abs(*clientCertFile)
			exitOnFailure(logger, err)
			absClientKeyFile, err := filepath.Abs(*clientKeyFile)
			exitOnFailure(logger, err)
			driverJsonSpec.TLSConfig = &dockerdriver.TLSConfig{InsecureSkipVerify: *insecureSkipVerify, CAFile: absCaFile, CertFile: absClientCertFile, KeyFile: absClientKeyFile}
			driverJsonSpec.Address = "https://" + atAddress
		}

		jsonBytes, err := json.Marshal(driverJsonSpec)

		exitOnFailure(logger, err)
		err = dockerdriver.WriteDriverSpec(logger, driversPath, "smbdriver", "json", jsonBytes)
		exitOnFailure(logger, err)
	} else {
		err := dockerdriver.WriteDriverSpec(logger, driversPath, "smbdriver", "spec", []byte(advertisedUrl))
		exitOnFailure(logger, err)
	}

	handler, err := driverhttp.NewHandler(logger, client)
	exitOnFailure(logger, err)

	var server ifrit.Runner
	if *requireSSL {
		tlsConfig, err := tlsconfig.
			Build(
				tlsconfig.WithIdentityFromFile(*certFile, *keyFile),
				tlsconfig.WithInternalServiceDefaults(),
			).
			Server(tlsconfig.WithClientAuthenticationFromFile(*caFile))
		if err != nil {
			logger.Fatal("tls-configuration-failed", err)
		}
		server = http_server.NewTLSServer(atAddress, handler, tlsConfig)
	} else {
		server = http_server.New(atAddress, handler)
	}

	return server
}

func createSmbDriverUnixServer(logger lager.Logger, client dockerdriver.Driver, atPort int) ifrit.Runner {
	atAddress := listenAddress + ":" + strconv.Itoa(atPort)
	handler, err := driverhttp.NewHandler(logger, client)
	exitOnFailure(logger, err)
	return http_server.NewUnixServer(atAddress, handler)
}

func newLogger() (lager.Logger, *lager.ReconfigurableSink) {
	lagerConfig := lagerflags.ConfigFromFlags()
	lagerConfig.RedactSecrets = true
	lagerConfig.RedactPatterns = SmbRedactValuePatterns()

	return lagerflags.NewFromConfig("smb-driver-server", lagerConfig)
}

func parseCommandLine() {
	lagerflags.AddFlags(flag.CommandLine)
	cf_debug_server.AddFlags(flag.CommandLine)
	flag.Parse()
}

func SmbRedactValuePatterns() []string {
	nfsPasswordValuePattern := `.*password=.*`
	awsAccessKeyIDPattern := `AKIA[A-Z0-9]{16}`
	/* #nosec */
	awsSecretAccessKeyPattern := `KEY["']?\s*(?::|=>|=)\s*["']?[A-Z0-9/\+=]{40}["']?`
	cryptMD5Pattern := `\$1\$[A-Z0-9./]{1,16}\$[A-Z0-9./]{22}`
	cryptSHA256Pattern := `\$5\$[A-Z0-9./]{1,16}\$[A-Z0-9./]{43}`
	cryptSHA512Pattern := `\$6\$[A-Z0-9./]{1,16}\$[A-Z0-9./]{86}`
	privateKeyHeaderPattern := `-----BEGIN(.*)PRIVATE KEY-----`

	return []string{nfsPasswordValuePattern, awsAccessKeyIDPattern, awsSecretAccessKeyPattern, cryptMD5Pattern, cryptSHA256Pattern, cryptSHA512Pattern, privateKeyHeaderPattern}
}
