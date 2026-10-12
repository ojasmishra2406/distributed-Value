#!/bin/bash
python3 -m grpc_tools.protoc -Iapi/proto --python_out=. --grpc_python_out=. api/proto/kv.proto
