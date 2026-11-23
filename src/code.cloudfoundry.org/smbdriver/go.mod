module code.cloudfoundry.org/smbdriver

go 1.27.3

require (
	code.cloudfoundry.org/debugserver v0.124.0
	code.cloudfoundry.org/dockerdriver v0.116.0
	code.cloudfoundry.org/goshims v0.122.0
	code.cloudfoundry.org/lager/v3 v3.96.0
	code.cloudfoundry.org/tlsconfig v0.73.0
	code.cloudfoundry.org/volume-mount-options v0.176.0
	code.cloudfoundry.org/volumedriver v0.201.0
	github.com/maxbrunsfeld/counterfeiter/v6 v6.9.0
	github.com/onsi/ginkgo/v2 v2.34.0
	github.com/onsi/gomega v1.44.0
	github.com/tedsuo/ifrit v0.0.0-20260908181113-dd353a7daa27
	github.com/tedsuo/rata v1.0.0
)

require (
	code.cloudfoundry.org/cfhttp/v2 v2.104.0 // indirect
	code.cloudfoundry.org/clock v1.97.0 // indirect
	github.com/Masterminds/semver/v3 v3.5.0 // indirect
	github.com/bmizerany/pat v0.0.0-20210406213842-e4b6760bdd6f // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20261118010117-084b33a85f63 // indirect
	github.com/openzipkin/zipkin-go v0.4.3 // indirect
	go.yaml.in/yaml/v3 v3.0.6 // indirect
	golang.org/x/mod v0.43.0 // indirect
	golang.org/x/net v0.63.0 // indirect
	golang.org/x/sync v0.25.0 // indirect
	golang.org/x/sys v0.50.0 // indirect
	golang.org/x/text v0.44.0 // indirect
	golang.org/x/tools v0.53.0 // indirect
)

// pin ifrit until https://github.com/tedsuo/ifrit/pull/48 is merged
replace github.com/tedsuo/ifrit => github.com/tedsuo/ifrit v0.0.0-20260418191334-846868129986
