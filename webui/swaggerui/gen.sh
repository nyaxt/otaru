#!/bin/bash
curl -o /tmp/swagger.tgz https://registry.npmjs.org/swagger-ui-dist/-/swagger-ui-dist-3.5.0.tgz 
mkdir /tmp/swagger
tar zxvf /tmp/swagger.tgz -C /tmp/swagger
rm dist/*
mv /tmp/swagger/package/* dist/
sed -i'' -e "s,http://petstore.swagger.io/v2/swagger.json,/otaru.swagger.json," dist/index.html
go-bindata -pkg swaggerui -prefix dist/ dist
