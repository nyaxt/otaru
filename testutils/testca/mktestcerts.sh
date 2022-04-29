#!/bin/bash

cd $(dirname $0)
OUTDIR=$(pwd)
BASEDIR=$(pwd)/kmgmbasedir
mkdir -p ${BASEDIR}

kmgm --basedir ${BASEDIR} --config ca.yml setup
kmgm --basedir ${BASEDIR} show --output pem ca > \
  ${OUTDIR}/cacert.pem

kmgm --basedir ${BASEDIR} --config issue_server.yml issue \
  --cert ${OUTDIR}/cert.pem \
  --priv ${OUTDIR}/cert-key.pem
