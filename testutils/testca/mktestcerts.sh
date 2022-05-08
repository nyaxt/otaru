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

kmgm --basedir ${BASEDIR} --profile clientauth --config clientauth_ca.yml setup
kmgm --basedir ${BASEDIR} --profile clientauth show --output pem ca > \
  ${OUTDIR}/clientauth_cacert.pem

for role in admin readonly invalid; do
  set -x
  kmgm --basedir ${BASEDIR} --profile clientauth --config clientauth_issue_${role}.yml issue \
    --cert ${OUTDIR}/clientauth_${role}.pem \
    --priv ${OUTDIR}/clientauth_${role}-key.pem
  set +x
done

kmgm --basedir ${BASEDIR} --profile wrong --config wrong_ca.yml setup
kmgm --basedir ${BASEDIR} --profile wrong show --output pem ca > \
  ${OUTDIR}/wrong_cacert.pem

kmgm --basedir ${BASEDIR} --profile wrong --config issue_wrong.yml issue \
  --cert ${OUTDIR}/wrong_cert.pem \
  --priv ${OUTDIR}/wrong_cert-key.pem
