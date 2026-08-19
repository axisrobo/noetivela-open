# NOETIVELA

**模型、路由与推理织造（Model, Routing & Inference Fabric）**
AxisRobo Architecture & Research Program

> **English:** [README.md](README.md) · **中文文档:** [README.zh-CN.md](README.zh-CN.md)

## 什么是 NOETIVELA？

NOETIVELA 是**企业级推理织造（enterprise inference fabric）**，它把模型选择当作
**受治理的决策**，而不是猜测。它统一目录化你已经在运行的模型与端点（云 API、
私有 NIM/vLLM、边缘），把每个任务需求转化为**推理契约（Inference Contract）**，
再基于策略、质量、时延、可靠性、地域与总成本证据，把每次请求路由到目录中的
合适候选。

它**不是** OpenAI 兼容代理，也**不是**"自动切换便宜模型"的开关。那些工具只是
加了一层转发；NOETIVELA 加的是一个控制面。

### 解决的问题

- **供应商碎片化。** 团队在云、私有与边缘端点之间追逐最好的模型——结果是在应用
  里硬编码模型名，被单一供应商锁定。
- **路由路径缺乏治理。** 质量/成本启发式算法会把请求路由到策略禁止的区域、供应商
  或数据级别——因为打分完全不知道哪些是**硬门禁（hard gate）**。
- **决策不透明。** 模型答错时，没人能解释为什么选它——没有审计轨迹、没有回放、
  无法修复。
- **成本失控。** 没有按任务计费，前沿模型的支出要到账单寄来才被发现。
- **凭据散落。** 应用直接持有各 provider 的 key；租户隔离、轮换与爆炸半径都是
  事后才考虑的事。
- **模型更替阵痛。** 金丝雀、影子、弃用、退役全靠手工、有风险——于是陈旧模型
  一直留在生产环境。

### 核心能力

| 能力 | 作用 |
|---|---|
| **模型与端点生命周期** | 模型、版本、部署与端点是头等、受治理的对象——像代码一样金丝雀、影子、弃用与退役。 |
| **推理契约** | 请求声明的是*需求*（`task`、`modality`、地域、数据级别、会话）——而不是模型名。应用从不硬编码模型。 |
| **硬门禁候选集** | 安全、地域、新鲜度与兼容性门禁先评估。违规**绝不被加权打分补偿**；不合格候选直接剔除。 |
| **多目标路由** | 在合格候选集中，基于质量、时延、可靠性、地域与总成本做优化——通过可读的路由策略 DSL 或学习模型。 |
| **TCoS 证据** | 每次决策都携带**服务总成本（TCoS）**分解与质量调整后的经济账，路由被证明比固定前沿模型基线更省。 |
| **可解释的路由决策** | 每个请求都产生可审计的决策轨迹：为什么选它、备选是什么、花了多少。可用账本反事实回放任何策略。 |
| **治理与安全** | 租户级凭据隔离（应用不持 key）、绝不绕过的预算/配额上限、默认拒绝的授权。 |

> 关于路由流水线、对象模型与经济的深入说明，见
> [NOETIVELA core README](../NOETIVELA)。

## NOETIVELA-open —— 本仓库

NOETIVELA 由三个仓库组成。本仓库是其**对外开放面**，采用 **Apache-2.0** 许可，
可被任意（包括商业闭源）产品自由集成。

| 仓库 | 角色 | 许可 |
|---|---|---|
| **NOETIVELA-open（本仓库）** | SDK、examples、API 规格、CLI/binary、公开文档 | Apache-2.0 |
| [NOETIVELA](../NOETIVELA) | 核心引擎实现 | AGPL-3.0 |
| [NOETIVELA-ee](../NOETIVELA-ee) | 企业版高价值功能 | 商业许可 |

### 本仓库包含什么

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

## CLI 与二进制

`noetivela` CLI 对接运行中的 gateway/controller，驱动完整治理链路：
contract → validate → eligible → chat → decision trace。

**安装** — 从 [GitHub Releases](../../releases) 下载对应平台预编译二进制
（linux / darwin / windows × amd64 / arm64，含 `checksums.txt`），或源码构建：

```bash
git clone https://github.com/axisrobo/noetivela-open.git
cd noetivela-open/cli
go build -o noetivela .      # 然后放入 PATH
```

**使用**（需要先启动 gateway，见"快速开始"）：

```bash
export NOETIVELA_URL=http://localhost:8080   # 默认 http://localhost:8080

noetivela version
noetivela contract validate -contract contract.json -policy legal-confidential-v2
noetivela eligible -contract contract.json -policy legal-confidential-v2
noetivela chat -contract contract.json -prompt "hi" -policy legal-confidential-v2
noetivela decisions -limit 50
noetivela usage
noetivela replay -baseline fixed:gpt-x -policy legal-confidential-v2 -limit 1000
noetivela train -version v2-table
```

示例 InferenceContract JSON 见 [`cli/contract.example.json`](cli/contract.example.json)。

## 版本对齐

三仓库共享同一 semver minor 线；本仓库的 `contracts/` 是 API 规格的权威
发布点，core / ee 引用固定版本。Breaking change 必须先在本仓库标记
deprecated。

## License

Apache License 2.0 — 见 [LICENSE](LICENSE)。
