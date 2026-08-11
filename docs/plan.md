# NOETIVELA-open 内容规划

## 1. contracts/（权威 API 规格）

| 文件 | 内容 | 优先级 |
|---|---|---|
| `openapi/data-plane.yaml` | chat/responses、streaming、embeddings、errors、usage | Phase 0 |
| `openapi/control-plane.yaml` | model/endpoint CRUD、policy CRUD、budget/quota、lifecycle | Phase 0 |
| `openapi/decision.yaml` | validate contract、eligible candidates、route/explain、replay | Phase 1 |
| `schema/inference-contract.json` | InferenceContract JSON Schema | Phase 0 |
| `schema/model-profile.json` | ModelIdentity/Version/Profile/CapabilityVector | Phase 0 |
| `schema/routing-decision.json` | RoutingDecision（候选、过滤原因、得分、版本证据） | Phase 0 |
| `dsl/routing-policy.md` + `dsl/grammar.ebnf` | Policy DSL 语法与示例 | Phase 1 |
| `proto/` | 内部高性能接口（gateway ↔ routing ↔ registry） | Phase 1 |

发布规则：本目录是唯一权威；core/ee 通过 Go module / submodule 引用固定 tag。

## 2. sdk/

| SDK | Phase | 关键能力 |
|---|---|---|
| `sdk/go` | 0 | contract builder、streaming、decision trace 读取 |
| `sdk/python` | 0 | 同上 + PRAXOVELA/ORCHADYN 集成首选 |
| `sdk/typescript` | 1 | 同上 |
| `sdk/java`、`sdk/rust` | 2–3 | 按需 |

SDK 设计原则：
- 只暴露 Contract / alias / profile，**不暴露 provider credential 与 model ID**。
- 返回对象始终含 `routing_decision`（reason codes、policy version、候选摘要）。
- OpenAI-compatible 迁移模式：一行 base_url 替换即可走 gateway。

## 3. examples/

| 示例 | 对应场景（产品计划 §14） | Phase |
|---|---|---|
| `quickstart/` | 最小 chat + contract + decision trace | 0 |
| `contract-review-agent/` | 企业合同审查（confidential、session affinity、fallback 同质量） | 1 |
| `multi-agent-swe/` | 多 Agent 软件工程（分类→低价模型、架构→高能力模型、TCoS 报告） | 2 |
| `edge-robotics/` | 边缘自治（deadline、offline、edge endpoint 优先） | 2 |
| `model-retirement/` | 退役迁移演练（无应用改码迁移默认候选） | 2 |

## 4. cli/

`noetivela` CLI（Go，随 core release 构建）：
- `noetivela models list / endpoints status`
- `noetivela contract validate -f contract.yaml`
- `noetivela route explain --policy legal-confidential-v2`
- `noetivela replay --baseline fixed:gpt-x --policy p.yaml --dataset ...`

binary 通过 GitHub Releases 发布（linux/darwin/windows, amd64/arm64）。

## 5. docs/（公开文档）

- 概念：Inference Fabric、Contract、hard gates、TCoS、context locality
- 快速开始、部署（self-host 单节点）、迁移指南（从 OpenAI SDK / LiteLLM）
- 市场能力矩阵（LiteLLM / Portkey / OpenRouter / Foundry Router / Bedrock / NIM 对比）
- 开源边界与商业版说明（链接 core 的 07-open-core-split）
