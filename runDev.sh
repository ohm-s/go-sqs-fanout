#!/bin/bash
set -x 
set -e 
go build -o bin/fanout cmd/fanout/main.go 
chmod +x bin/fanout && \
./bin/fanout
