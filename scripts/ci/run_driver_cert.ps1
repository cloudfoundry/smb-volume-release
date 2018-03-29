$ErrorActionPreference = “Stop”;
trap { $host.SetShouldExit(1) }

function CheckLastExitCode {
    param ([int[]]$SuccessCodes = @(0), [scriptblock]$CleanupScript=$null)

    if ($SuccessCodes -notcontains $LastExitCode) {
        if ($CleanupScript) {
            "Executing cleanup script: $CleanupScript"
            &$CleanupScript
        }
        $msg = @"
EXE RETURNED EXIT CODE $LastExitCode
CALLSTACK:$(Get-PSCallStack | Out-String)
"@
        Stop-Process -Name "smbdriver"
        throw $msg
    }
}

cd smb-volume-release

$env:GOPATH=$PWD
$env:PATH="$PWD/bin;$env:PATH"

go install github.com/onsi/ginkgo/ginkgo

$driver_address="http://0.0.0.0:8589"

mkdir voldriver_plugins
$drivers_path="$PWD/voldriver_plugins"

mkdir "$PWD/tmp"
$SOURCE="$env:smbremotepath"

"{ `"volman_driver_path`": `"./voldriver_plugins`", `"driver_address`": `"$driver_address`", `"driver_name`": `"smbdriver`", `"create_config`": { `"Name`": `"smb-volume-name`", `"Opts`": {`"source`":`"$SOURCE`",`"uid`":`"2000`",`"gid`":`"2000`",`"username`":`"$env:smbusername`",`"password`":`"$env:smbpassword`"} } } " | Set-Content $PWD/tmp/fixture.json -Force 

$env:FIXTURE_FILENAME="$PWD/tmp/fixture.json"

go build -o "./tmp/smbdriver" "src/code.cloudfoundry.org/smbdriver/cmd/smbdriver/main.go"

go get -t code.cloudfoundry.org/volume_driver_cert

$mountDir="$PWD/tmp/mountdir"
mkdir $mountDir


Start-Process -NoNewWindow ./tmp/smbdriver "-listenPort=8589 -transport=tcp -driversPath=$drivers_path -mountDir=$mountDir --mountFlagAllowed=`"username,password,uid,gid,file_mode,dir_mode,readonly,domain,vers,sec`" --mountFlagDefault=`"uid:2000,gid:2000`""

ginkgo -v -keepGoing src/code.cloudfoundry.org/volume_driver_cert
CheckLastExitCode

Stop-Process -Name "smbdriver"
