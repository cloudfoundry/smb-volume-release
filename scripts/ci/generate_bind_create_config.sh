#!/bin/bash

set -e -x

source "$(dirname "$0")/utils.sh"

check_param SMB_REMOTE_PATH
check_param SMB_USERNAME
check_param SMB_PASSWORD

CREATE_CONFIG=${PWD}/bind-create-config/create-config.json
BIND_CONFIG=${PWD}/bind-create-config/bind-config.json

echo "{\"share\":\"$SMB_REMOTE_PATH\"}" > "${CREATE_CONFIG}"

json_payload=$(echo "{\"username\":\"$SMB_USERNAME\",\"password\":\"$SMB_PASSWORD\"}" | sed 's/"/\\"/g')
echo "[\"$json_payload\"]" > "${BIND_CONFIG}"
