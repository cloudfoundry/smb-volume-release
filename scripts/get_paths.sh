#!/bin/bash

absolute_path() {
  (cd $1 && pwd)
}

scripts_path=$(absolute_path `dirname $0`)

SMB_RELEASE_DIR=${SMB_RELEASE_DIR:-$(absolute_path $scripts_path/..)}

echo SMB_RELEASE_DIR=$SMB_RELEASE_DIR
