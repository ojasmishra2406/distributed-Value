#!/bin/bash
kubectl port-forward kv-node-0 50051:50051 &
PF0=$!
kubectl port-forward kv-node-1 50052:50051 &
PF1=$!
kubectl port-forward kv-node-2 50053:50051 &
PF2=$!

sleep 3
./script_linux

kill $PF0 $PF1 $PF2
