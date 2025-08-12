#!/bin/bash

set -eu
set -o pipefail

THIS_FILE_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
CI="${THIS_FILE_DIR}/../../wg-app-platform-runtime-ci"
. "$CI/shared/helpers/release-note-helpers.bash"
. "$CI/shared/helpers/git-helpers.bash"
REPO_NAME=$(git_get_remote_name)
REPO_PATH="${THIS_FILE_DIR}/../"
unset THIS_FILE_DIR

SMB_START_REF="${1}" # ex: "v0.0.7"
SMB_END_REF="${2}" # ex: "v0.0.8"

get_non_bot_commits "${SMB_START_REF}" "${SMB_END_REF}"

SMB_START_REF_DOCKER_DRIVER=$(git rev-parse "${SMB_START_REF}:src/code.cloudfoundry.org/dockerdriver")
SMB_END_REF_DOCKER_DRIVER=$(git rev-parse "${SMB_END_REF}:src/code.cloudfoundry.org/dockerdriver")
pushd src/code.cloudfoundry.org/dockerdriver > /dev/null
  display_go_mod_diff "${SMB_START_REF_DOCKER_DRIVER}" "${SMB_END_REF_DOCKER_DRIVER}" go.mod "dockerdriver"
  echo ""
popd > /dev/null

display_go_mod_diff "${SMB_START_REF}" "${SMB_END_REF}" src/code.cloudfoundry.org/smbbroker/go.mod "smbbroker"
echo ""

display_go_mod_diff "${SMB_START_REF}" "${SMB_END_REF}" src/code.cloudfoundry.org/smbdriver/go.mod "smbdriver"
echo ""

display_blob_change_info "${SMB_START_REF}" "${SMB_END_REF}" config/blobs.yml
