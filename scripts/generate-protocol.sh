#!/bin/sh
set -eu

workspace_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
protocol_root="$workspace_root/protocol"
expected_tag="v0.1.0"
expected_commit="5455d11917ab684c84f164eaa7f12831f8fece6b"

actual_commit=$(git -C "$protocol_root" rev-parse HEAD)
if [ "$actual_commit" != "$expected_commit" ]; then
  echo "协议版本不匹配：需要 $expected_tag@$expected_commit，当前为 $actual_commit" >&2
  exit 1
fi

timestamp_include=$(go env GOPATH)/pkg/mod/github.com/gogo/protobuf@v1.3.2/protobuf
if [ ! -f "$timestamp_include/google/protobuf/timestamp.proto" ]; then
  echo "缺少 google/protobuf/timestamp.proto；请先安装锁定的生成工具依赖" >&2
  exit 1
fi

protoc \
  -I "$protocol_root/protobuf" \
  -I "$timestamp_include" \
  --go_out="$workspace_root/node" \
  --go_opt=module=github.com/yeruyi1024/novagrid-node \
  --go_opt=Mnode/v1/node.proto=github.com/yeruyi1024/novagrid-node/protocol/node/v1 \
  --go_opt=Mgoogle/protobuf/timestamp.proto=google.golang.org/protobuf/types/known/timestamppb \
  --go-grpc_out="$workspace_root/node" \
  --go-grpc_opt=module=github.com/yeruyi1024/novagrid-node \
  --go-grpc_opt=Mnode/v1/node.proto=github.com/yeruyi1024/novagrid-node/protocol/node/v1 \
  --go-grpc_opt=Mgoogle/protobuf/timestamp.proto=google.golang.org/protobuf/types/known/timestamppb \
  "$protocol_root/protobuf/node/v1/node.proto"
