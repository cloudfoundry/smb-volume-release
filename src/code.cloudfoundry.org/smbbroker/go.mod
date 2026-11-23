module code.cloudfoundry.org/smbbroker

go 1.27.3

// pin ifrit until https://github.com/tedsuo/ifrit/pull/48 is merged
replace github.com/tedsuo/ifrit => github.com/tedsuo/ifrit v0.0.0-20260418191334-846868129986

replace code.cloudfoundry.org/smbdriver => ../smbdriver

require (
	code.cloudfoundry.org/brokerapi/v13 v13.0.25
	code.cloudfoundry.org/clock v1.97.0
	code.cloudfoundry.org/debugserver v0.124.0
	code.cloudfoundry.org/existingvolumebroker v0.244.0
	code.cloudfoundry.org/goshims v0.122.0
	code.cloudfoundry.org/lager/v3 v3.96.0
	code.cloudfoundry.org/service-broker-store v0.183.0
	code.cloudfoundry.org/smbdriver v0.0.0-20240819143446-ac4a9e63e92c
	code.cloudfoundry.org/volume-mount-options v0.176.0
	github.com/google/gofuzz v1.2.0
	github.com/onsi/ginkgo/v2 v2.34.0
	github.com/onsi/gomega v1.44.0
	github.com/tedsuo/ifrit v0.0.0-20260908181113-dd353a7daa27
)

require (
	code.cloudfoundry.org/cfhttp/v2 v2.104.0 // indirect
	code.cloudfoundry.org/credhub-cli v0.0.0-20261120165619-4af7cfa7be21 // indirect
	code.cloudfoundry.org/dockerdriver v0.116.0 // indirect
	code.cloudfoundry.org/tlsconfig v0.73.0 // indirect
	code.cloudfoundry.org/volumedriver v0.201.0 // indirect
	github.com/Masterminds/semver/v3 v3.5.0 // indirect
	github.com/bmizerany/pat v0.0.0-20210406213842-e4b6760bdd6f // indirect
	github.com/cloudfoundry/go-socks5 v0.0.0-20250423223041-4ad5fea42851 // indirect
	github.com/cloudfoundry/socks5-proxy v0.2.198 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20261118010117-084b33a85f63 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/go-version v1.9.0 // indirect
	github.com/openzipkin/zipkin-go v0.4.3 // indirect
	github.com/tedsuo/rata v1.0.0 // indirect
	go.yaml.in/yaml/v3 v3.0.6 // indirect
	golang.org/x/crypto v0.59.0 // indirect
	golang.org/x/mod v0.43.0 // indirect
	golang.org/x/net v0.63.0 // indirect
	golang.org/x/sync v0.25.0 // indirect
	golang.org/x/sys v0.50.0 // indirect
	golang.org/x/text v0.44.0 // indirect
	golang.org/x/tools v0.53.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)
