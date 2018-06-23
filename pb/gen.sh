#!/bin/bash

cd `dirname $0`
GOPATH=${GOPATH:-$HOME/go}

protofiles=( otaru.proto )
protocflags="-I${GOPATH}/src -I."
protocflags="${protocflags} -I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway"
protocflags="${protocflags} -I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis"
protoc ${protocflags} \
  --go_out=plugins=grpc:. \
  ${protofiles[*]}
protoc ${protocflags} \
  --grpc-gateway_out=logtostderr=true:. \
  ${protofiles[*]}
protoc ${protocflags} \
  --swagger_out=logtostderr=true:./json/dist \
  ${protofiles[*]}
go generate .
