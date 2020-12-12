#!/bin/bash -eux

docker run \
-t \
-i \
-e DEV=true \
--privileged \
-v /Users/pivotal/workspace/smb-volume-release/:/smb-volume-release \
--workdir=/ \
harbor-repo.vmware.com/dockerhub-proxy-cache/bosh/main-bosh-docker \
/smb-volume-release/scripts/run-bosh-release-tests-in-docker-env.sh
