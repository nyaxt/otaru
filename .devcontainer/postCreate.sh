#!/bin/bash
set -euo pipefail

PREFIX="/usr/local" && \
VERSION="1.41.0" && \
curl -sSL \
"https://github.com/bufbuild/buf/releases/download/v${VERSION}/buf-$(uname -s)-$(uname -m).tar.gz" | \
sudo tar -xvzf - -C "${PREFIX}" --strip-components 1
