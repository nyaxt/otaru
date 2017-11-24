#!/bin/bash

cd `dirname $0`
GOPATH=${GOPATH:-$HOME/go}

protofiles=( otaru.proto )
protocflags="-I${GOPATH}/src -I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis -I."
protoc $protocflags \
  --go_out=plugins=grpc:. \
  ${protofiles[*]}
protoc $protocflags \
  --grpc-gateway_out=logtostderr=true:. \
  ${protofiles[*]}
protoc $protocflags \
  --swagger_out=logtostderr=true:./json/src \
  ${protofiles[*]}
(cd json && go-bindata -pkg json -prefix src/ src)
go generate .
