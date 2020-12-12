#!/bin/bash -ex

COMMAND_TO_RUN='ginkgo -nodes 1 -v .'
if [[ -n "$DEV" ]]; then
    COMMAND_TO_RUN='bash'
fi

export DOCKER_STORAGE_OPTIONS='--storage-opt dm.basesize=100G'
. start-bosh

source /tmp/local-bosh/director/env
export DOCKER_TMP_DIR=$(find /tmp/ -name "tmp.*")

docker \
--tls \
--tlscacert=${DOCKER_TMP_DIR}/ca.pem \
--tlscert=${DOCKER_TMP_DIR}/cert.pem \
--tlskey=${DOCKER_TMP_DIR}/key.pem \
run \
--network=director_network \
-v $PWD/smb-volume-release/:/smb-volume-release \
-v /tmp:/tmp \
-w /smb-volume-release/src/bosh_release \
-t \
-i \
--env BOSH_ENVIRONMENT=10.245.0.3 \
--env BOSH_CLIENT=${BOSH_CLIENT} \
--env BOSH_CLIENT_SECRET=${BOSH_CLIENT_SECRET} \
--env BOSH_CA_CERT=${BOSH_CA_CERT} \
--env SMB_VOLUME_RELEASE_PATH=/smb-volume-release \
harbor-repo.vmware.com/dockerhub-proxy-cache/cfpersi/bosh-release-tests \
    bash -c "echo '**** from the bash shell, run ginkgo -nodes 1 -r -v .' && $COMMAND_TO_RUN"