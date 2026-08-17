# NovaGrid Node

NovaGrid 节点客户端公开仓库，计划包含共享 Agent 核心、Windows 原生桌面客户端、Ubuntu Agent、llama.cpp Runtime 管理、模型下载、遥测和安全更新。

当前状态：正在建立遵循 NodeChannel v1 的确定性模拟 Node 和模拟文本 Runtime。模拟实现不访问 GPU、网络、用户目录或真实设备身份，不得用于声明真实平台兼容。

第一版边界：Windows 原生 CUDA，不使用 WSL；Ubuntu 独立支持；单设备单 GPU 单请求；节点只建立出站 TLS 连接；不提供任意远程 Shell，不读取个人目录或采集无关活动。

公开源代码不代表模型权重或第三方组件自动获得相同许可。正式开源许可证将在依赖与商业策略确认后单独加入；在此之前保留全部权利。

## 本地运行

需要 Go 1.23。运行一次确定性成功场景：

```bash
go run ./cmd/novagrid-mock-node --scenario success
```

支持的场景为 `success`、`reject`、`timeout`、`disconnect` 和 `late`。命令只输出场景、决定、结果和 Token 数，不输出请求或回答正文。

## 验证

```bash
make lint
make test
make build
go test -race ./...
make test-scenarios
```

NodeChannel Go 类型从已发布的 Protocol `v0.1.0` 生成，来源、版本和再生成命令见 `protocol/node/v1/README.md`。

## 依赖

- `google.golang.org/protobuf v1.34.2`（BSD-3-Clause）：解析和保留 Protobuf 未知字段；直接手写协议结构会失去机器兼容保证。
- `google.golang.org/grpc v1.65.0`（Apache-2.0）：编译 NodeChannel 生成的双向流接口；本任务不建立真实网络连接。
- `github.com/gogo/protobuf v1.3.2`（BSD-3-Clause，仅生成工具）：提供本地 `protoc` 发行包缺少的标准 `timestamp.proto`；不进入 Node 运行时。

前两个依赖是 Protocol Go 消费者的最小官方运行时，第三个只用于再生成协议代码。替代方案是自行维护编解码器或只使用 JSON 模拟，但会丢失 Protobuf 未知字段兼容性和 gRPC 服务签名检查，因此不采用。
