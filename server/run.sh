#!/bin/bash
endpoint="3q48uo76fdqg34f8q367dghf4"
port="8443"
go run . -cert ../certs/selfsigned.crt -key ../certs/selfsigned.key  -port "${port}" --endpoint "${endpoint}"