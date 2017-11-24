FROM golang:1.9
ENV GOOS linux
ENV GOARCH amd64

VOLUME ["/out", "/otaru-testconf"]

COPY . /go/src/github.com/nyaxt/otaru
WORKDIR /go/src/github.com/nyaxt/otaru

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go install github.com/nyaxt/otaru/cmd/... github.com/nyaxt/otaru/debugcmd/... github.com/nyaxt/otaru/gcloud/auth/otaru-gcloudauthcli

CMD cp /go/bin/otaru* /out/
