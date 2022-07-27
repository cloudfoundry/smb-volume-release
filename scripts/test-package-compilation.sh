#!/usr/bin/env bash

set -x

# BASE_DIR="$(cd "$(dirname "$0")/.." && pwd)"

set -eo pipefail

docker run  \
  -v "${PWD}:/release" \
  -w /release/blobs \
  -it \
  cryogenics/bosh-docker-boshlite-ubuntu-jammy \
  /bin/bash -c "$(cat << 'EOF'
    set -x
    set -e

    TMPDIR=/tmp

    install_vendored_packages() {
      bucket_name="$(sed -En 's/^.*bucket_name: (.*)$/\1/p' ../config/final.yml)"

      for package_dir in ../packages/*
      do
        if [ -f "${package_dir}/spec.lock" ]
        then
          package_name="$(basename "$package_dir")"
          package_fingerprint="$(sed -En 's/^fingerprint: (.*)$/\1/p' "${package_dir}/spec.lock")"
          blobstore_id="$(grep -A1 "version: ${package_fingerprint}" "../.final_builds/packages/${package_name}/index.yml" | sed -En 's/.*blobstore_id: (.*)$/\1/p')"

          export BOSH_INSTALL_TARGET="/var/vcap/packages/${package_name}"
          mkdir -p "$BOSH_INSTALL_TARGET"

          mkdir -p "${TMPDIR}/${package_name}.tgz"
          ( cd "${TMPDIR}/${package_name}.tgz"
            wget --no-check-certificate --no-proxy "https://${bucket_name}.s3.amazonaws.com/${blobstore_id}" -O- | tar -zxv

            ls -la

            /bin/bash packaging
          )
        fi
      done
    }
    install_vendored_packages


    PACKAGE=cifs-utils

    export BOSH_INSTALL_TARGET="/var/vcap/packages/${PACKAGE}"

    export BOSH_COMPILE_TARGET="$(mktemp -d)"
    cp -R "../packages/${PACKAGE}"/* "$BOSH_COMPILE_TARGET"
    cp -R * "$BOSH_COMPILE_TARGET"

    ( cd "$BOSH_COMPILE_TARGET"
      /bin/bash packaging
    )
EOF
  )"
