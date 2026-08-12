# NovaGrid Node

NovaGrid 节点客户端公开仓库，计划包含共享 Agent 核心、Windows 原生桌面客户端、Ubuntu Agent、llama.cpp Runtime 管理、模型下载、遥测和安全更新。

当前状态：仅完成仓库基线初始化，尚未开始产品代码开发。

第一版边界：Windows 原生 CUDA，不使用 WSL；Ubuntu 独立支持；单设备单 GPU 单请求；节点只建立出站 TLS 连接；不提供任意远程 Shell，不读取个人目录或采集无关活动。

公开源代码不代表模型权重或第三方组件自动获得相同许可。正式开源许可证将在依赖与商业策略确认后单独加入；在此之前保留全部权利。
