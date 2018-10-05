#!/bin/bash

set -e -x

cd smb-volume-release/

export GOROOT=/usr/local/go
export PATH=$GOROOT/bin:$PATH

export GOPATH=$PWD
export PATH=$PWD/bin:$PATH

go get github.com/onsi/ginkgo/ginkgo
go get github.com/onsi/gomega
go get gopkg.in/DATA-DOG/go-sqlmock.v1

pushd src/code.cloudfoundry.org/${SUBMODULE_NAME}
  ginkgo -r -keepGoing -p -trace -randomizeAllSpecs -progress --race "$@"
popd
