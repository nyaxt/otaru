#!/bin/bash
cd $(dirname $0)
go-bindata -pkg webui -prefix dist/ dist
