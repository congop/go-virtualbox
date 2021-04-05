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

if [[ (-n "${version_without_v_prefix}") ]]; then
    echo "Usage: $0 <version e.g. 1.27.0 or v1.27.0>"
    exit 1
fi

if command ${GOLANGCI_LINT_PATH} > /dev/null; then
    echo "Golangci-lint installed checking verion: required version:[${version}] trimmed-v-prefix: [${version_without_v_prefix}]"
    current_version_str=$(${GOLANGCI_LINT_PATH} --version)
    if [[ ${current_version_str} == *"${version_without_v_prefix}"* ]]; then
        echo "required version ( ${version} ) already installed, kipping installation: full version text: ${current_version_str}"
        exit 0
    fi
fi

echo "Installing golangci-lint ${version}"

curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin ${version}

${GOLANGCI_LINT_PATH} --version

