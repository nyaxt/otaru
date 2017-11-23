FROM golang:1.9
ENV GOOS linux
ENV GOARCH amd64

RUN apt-get update \
    && apt-get install -y git xz-utils \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

RUN go get github.com/jteeuwen/go-bindata/...

VOLUME ["/out", "/otaru-testconf"]

COPY . /go/src/github.com/nyaxt/otaru
WORKDIR /go/src/github.com/nyaxt/otaru

RUN go-bindata -pkg webui dist/...
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go install github.com/nyaxt/otaru/cmd/... github.com/nyaxt/otaru/debugcmd/... github.com/nyaxt/otaru/gcloud/auth/otaru-gcloudauthcli

CMD cp /go/bin/otaru* /out/
