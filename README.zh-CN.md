# NOETIVELA-open

**Model, Routing & Inference Fabric — 开放集成面**
AxisRobo Architecture & Research Program

> **English:** [README.md](README.md) · **中文文档:** [README.zh-CN.md](README.zh-CN.md)

> NOETIVELA is an enterprise inference fabric that catalogs models and endpoints,
> converts task requirements into governed inference contracts, and routes each
> request across cloud, private and edge intelligence using policy, quality,
> latency, reliability, locality and total-cost evidence.

本仓库是 NOETIVELA 的**对外开放面**，采用 **Apache-2.0** 许可，可被任意
（包括商业闭源）产品自由集成。

| 仓库 | 角色 | 许可 |
|---|---|---|
| **NOETIVELA-open（本仓库）** | SDK、examples、API 规格、CLI/binary、公开文档 | Apache-2.0 |
| [NOETIVELA](../NOETIVELA) | 核心引擎实现 | AGPL-3.0 |
| [NOETIVELA-ee](../NOETIVELA-ee) | 企业版高价值功能 | 商业许可 |

## 本仓库包含什么

| 目录 | 内容 |
|---|---|
| `contracts/` | **权威 API 规格**：OpenAPI（data/control plane）、Protobuf、InferenceContract JSON Schema、Routing Policy DSL 语法 |
| `sdk/` | 官方 SDK：Go / Python / TypeScript（后续 Java / Rust） |
| `examples/` | quickstart、合同审查 Agent、多 Agent 软件工程、边缘自治等端到端示例 |
| `cli/` | `noetivela` CLI 源码与预编译 binary 发布说明 |

> 公开用户文档（概念、快速开始、迁移指南、能力矩阵）正在编写中，后续发布在
> 官方文档站点；内部设计与开发文档见 NOETIVELA-ee。

## 本仓库不包含什么

- 核心引擎实现（在 NOETIVELA core，AGPL）。
- 企业功能（多租户、HA、TCoS/FinOps、learned routing 等，在 NOETIVELA-ee）。
- 任何 provider credential 或客户配置。

## 快速开始

```bash
# 1. 起控制面（bootstrap 示例目录）与数据面（示例 policy）
NOETIVELA_BOOTSTRAP_FILE=../NOETIVELA/deploy/bootstrap/manifest.example.json \
    go run ../NOETIVELA/backend/cmd/noetivela-controller &   # :8081
NOETIVELA_POLICY_FILE=../NOETIVELA/deploy/policies/legal.confidential.npl \
    go run ../NOETIVELA/backend/cmd/noetivela-gateway &       # :8080

# 2. 用 Go SDK 提交 Inference Contract（而非硬编码模型名）
go run ./examples/quickstart
```

```go
import "github.com/axisrobo/noetivela-open/sdk/go/client"

c := client.New("http://localhost:8080").WithContract(&client.Contract{
    Task:     "contract_clause_extraction",
    Modality: []string{"text"},
    Policy:   &client.Policy{AllowedRegions: []string{"sg", "cn-north"},
                              ProviderTraining: "prohibited"},
    Context:  &client.Context{SessionID: "case-8821"},
})

// 不消耗 token 的可满足性检查 + 排名
res, _ := c.ValidateContract(ctx, contract, "legal-confidential-v2")
ranked, _ := c.EligibleCandidates(ctx, contract, "legal-confidential-v2")

// 受治理的推理请求；返回结果 + RoutingDecision + TCoS
chat, _ := c.Chat(ctx, "auto", []client.Message{{Role: "user", Content: "..."}})
fmt.Println(chat.Noetivela.EndpointRef, chat.Noetivela.TCoS.Total)
```

## 版本对齐

三仓库共享同一 semver minor 线；本仓库的 `contracts/` 是 API 规格的权威
发布点，core / ee 引用固定版本。Breaking change 必须先在本仓库标记
deprecated。

## License

Apache License 2.0 — 见 [LICENSE](LICENSE)。
