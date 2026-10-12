#!/bin/bash
export PATH=/usr/local/go/bin:$PATH
cd /mnt/c/Users/mishr/OneDrive/Documents/Distributed-values
rm -rf data_bench
go run cmd/node/main.go -port=50051 -dir=data_bench
