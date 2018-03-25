#!/bin/bash
set +e +x

TARGET_BROKER=$PWD/pcf-tile/resources/azurefilebroker.zip
TARGET_SMB=$PWD/pcf-tile/resources/smb-volume-0.1.4+dev.1.tgz

rm -f $TARGET_BROKER
rm -f $TARGET_SMB

rm -rf .dev_builds
rm -rf dev_releases

rm -f pcf-tile/tile-history.yml
rm -rf src/code.cloudfoundry.org/azurefilebroker/bin

echo "Building SMB volume release"
bosh create-release --tarball=$TARGET_SMB

direnv allow .
echo "Update to the latest"
git checkout master
./scripts/update

echo "Building the broker"
./scripts/build-broker

echo "Packaging the broker"
pushd src/code.cloudfoundry.org/azurefilebroker
  zip -r $TARGET_BROKER bin/azurefilebroker Procfile
popd

echo "Building the tile"
pushd pcf-tile
  if [ "$1" = "-major" ]; then
    tile build major
  elif [ "$1" = "-minor" ]; then
    tile build minor
  else
    tile build
  fi
popd