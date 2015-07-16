FROM golang:1.4
ENV GOOS linux
ENV GOARCH amd64
RUN go get github.com/tools/godep
VOLUME /out

COPY . /go/src/github.com/nyaxt/otaru
WORKDIR /go/src/github.com/nyaxt/otaru
RUN godep go install github.com/nyaxt/otaru/fuse/cli

CMD cp /go/bin/cli /out/cli
