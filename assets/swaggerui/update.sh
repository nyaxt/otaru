#!/bin/bash
VERSION=3.17.1
curl -o /tmp/swagger.tgz https://registry.npmjs.org/swagger-ui-dist/-/swagger-ui-dist-${VERSION}.tgz 
mkdir /tmp/swagger
tar zxvf /tmp/swagger.tgz -C /tmp/swagger
rm dist/*
mv /tmp/swagger/package/* dist/
sed -i'' -e "s,https://petstore.swagger.io/v2/swagger.json,/otaru.swagger.json," dist/index.html
