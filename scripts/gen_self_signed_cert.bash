#!/bin/bash

source $(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.bash
otaru::gen_self_signed_cert
