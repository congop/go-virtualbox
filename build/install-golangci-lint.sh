#!/bin/bash

###############################################################################
# Does install golangci-lint
# Usage: 
#    install-golangci-lint.sh <version>
#  fb74c2e
###############################################################################  

version=${1}
#remove starting v
#Removing white, even those in beetween, version is anyway not suppose to have one
version_without_v_prefix="${version// /}"
version_without_v_prefix=${version#v}

GOLANGCI_LINT_PATH=$(go env GOPATH)/bin/golangci-lint

 echo "Golangci-lint install wanted: version=$version  version_without_v_prefix=${version_without_v_prefix}"

if [ -z "${version_without_v_prefix}" ]; then
    echo "Usage: $0 <version e.g. 1.27.0 or v1.27.0>"
    exit 1
fi

if [ -x "${GOLANGCI_LINT_PATH}" ]; then
    current_version_str=$(${GOLANGCI_LINT_PATH} --version 2>&1 || true)
    echo "Golangci-lint installed checking version: required version:[${version}] trimmed-v-prefix: [${version_without_v_prefix}] current_version_str=[${current_version_str}]"
    # mind the pattern matching dddXYZqqq==*XYZ*
    if [[ ${current_version_str} == *"${version_without_v_prefix}"* ]]; then
        echo "Golangci-lint  required version ( ${version} ) already installed, kipping installation: full version text: ${current_version_str}"
        exit 0
    fi
fi

echo "Installing golangci-lint ${version}"

curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin ${version}

# This is for logging. We do not want it to to fail this script.
# E.g. v1.44.2 would fail if built with go-version 1.18
${GOLANGCI_LINT_PATH} --version || true

