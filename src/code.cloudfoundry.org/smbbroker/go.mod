module code.cloudfoundry.org/smbbroker

go 1.26.0

// pin ifrit until https://github.com/tedsuo/ifrit/pull/48 is merged
replace github.com/tedsuo/ifrit => github.com/tedsuo/ifrit v0.0.0-20260418191334-846868129986

replace code.cloudfoundry.org/smbdriver => ../smbdriver

require (
	code.cloudfoundry.org/brokerapi/v13 v13.0.25
	code.cloudfoundry.org/clock v1.89.0
	code.cloudfoundry.org/debugserver v0.116.0
	code.cloudfoundry.org/existingvolumebroker v0.236.0
	code.cloudfoundry.org/goshims v0.115.0
	code.cloudfoundry.org/lager/v3 v3.88.0
	code.cloudfoundry.org/service-broker-store v0.175.0
	code.cloudfoundry.org/smbdriver v0.0.0-20240819143446-ac4a9e63e92c
	code.cloudfoundry.org/volume-mount-options v0.169.0
	github.com/google/gofuzz v1.2.0
	github.com/onsi/ginkgo/v2 v2.33.0
	github.com/onsi/gomega v1.44.0
	github.com/tedsuo/ifrit v0.0.0-20260908181113-dd353a7daa27
)

require (
	code.cloudfoundry.org/cfhttp/v2 v2.96.0 // indirect
	code.cloudfoundry.org/credhub-cli v0.0.0-20260921130234-f80c1c8a1b4a // indirect
	code.cloudfoundry.org/dockerdriver v0.108.0 // indirect
	code.cloudfoundry.org/tlsconfig v0.68.0 // indirect
	code.cloudfoundry.org/volumedriver v0.192.0 // indirect
	github.com/Masterminds/semver/v3 v3.5.0 // indirect
	github.com/bmizerany/pat v0.0.0-20210406213842-e4b6760bdd6f // indirect
	github.com/cloudfoundry/go-socks5 v0.0.0-20250423223041-4ad5fea42851 // indirect
	github.com/cloudfoundry/socks5-proxy v0.2.189 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20260926063103-aaccee046517 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/go-version v1.9.0 // indirect
	github.com/openzipkin/zipkin-go v0.4.3 // indirect
	github.com/tedsuo/rata v1.0.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/crypto v0.57.0 // indirect
	golang.org/x/mod v0.41.0 // indirect
	golang.org/x/net v0.59.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	golang.org/x/tools v0.50.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)
