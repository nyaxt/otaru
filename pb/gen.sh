#!/bin/bash

cd `dirname $0`
GOPATH=${GOPATH:-$HOME/go}

protofiles=(
  system.proto
)
protocflags="-I${GOPATH}/src -I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis -I."
for protofile in $protofiles; do
  protoc $protocflags \
    --go_out=plugins=grpc:. \
    $protofile
  protoc $protocflags \
    --grpc-gateway_out=logtostderr=true:. \
    $protofile
  protoc $protocflags \
    --swagger_out=logtostderr=true:. \
    $protofile
done
go generate .
