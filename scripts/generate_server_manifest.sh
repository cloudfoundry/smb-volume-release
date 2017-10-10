#!/bin/bash
#generate_manifest.sh

set -e -x

home="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && cd .. q&& pwd )"
templates=${home}/templates

MANIFEST_NAME=smb-test-server-azure-manifest

spiff merge ${templates}/smb-test-server-manifest-azure.yml $1 $2 > $PWD/$MANIFEST_NAME.yml

echo manifest written to $PWD/$MANIFEST_NAME.yml
