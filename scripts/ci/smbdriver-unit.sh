#!/bin/bash

set -e -x

cd smb-volume-release/

export GOROOT=/usr/local/go
export PATH=$GOROOT/bin:$PATH

export GOPATH=$PWD
export PATH=$PWD/bin:$PATH

go get github.com/onsi/ginkgo/ginkgo
go get github.com/onsi/gomega

pushd src/github.com/cloudfoundry/smbdriver
  ginkgo -r -keepGoing -p -trace -randomizeAllSpecs -progress --race "$@"
popd
