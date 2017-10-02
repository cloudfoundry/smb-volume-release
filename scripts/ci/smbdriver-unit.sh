#!/bin/bash

set -e -x

cd smb-volume-release/

export GOROOT=/usr/local/go
export PATH=$GOROOT/bin:$PATH

export GOPATH=$PWD
export PATH=$PWD/bin:$PATH

go install github.com/onsi/ginkgo/ginkgo
go install github.com/onsi/gomega

pushd src/github.com/cloudfoundry/smbdriver
  ginkgo -r -keepGoing -p -trace -randomizeAllSpecs -progress --race "$@"
popd
