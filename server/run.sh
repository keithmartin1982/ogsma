#!/bin/bash
endpoint="ws"
port="8443"
go run . -cert ../certs/selfsigned.crt -key ../certs/selfsigned.key  -port "${port}" --endpoint "${endpoint}"