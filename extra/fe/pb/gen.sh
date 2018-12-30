#!/bin/bash

cd `dirname $0`
echo "* Running protocs for `pwd`"
GOPATH=${GOPATH:-$HOME/go}

protofiles=( otaru-fe.proto )
protocflags="-I${GOPATH}/src -I. -I../../../pb"
protocflags="${protocflags} -I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway"
protocflags="${protocflags} -I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis"

set -v 
protoc ${protocflags} \
  --go_out=plugins=grpc:. \
  ${protofiles[*]}
protoc ${protocflags} \
  --grpc-gateway_out=logtostderr=true:. \
  ${protofiles[*]}
protoc ${protocflags} \
  --swagger_out=logtostderr=true:./json/dist \
  ${protofiles[*]}
go generate ./json
