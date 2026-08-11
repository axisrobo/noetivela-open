# Routing Policy DSL

NOETIVELA Routing Policy 定义 hard gates（`require`）与候选集内偏好
（`prefer` / `optimize`）。**任何 `require` 违反都不能被加权优化抵消。**

## 示例

```
policy legal-confidential-v2 {
  require data.region in ["sg", "cn-north"]
  require model.eval("legal_clause_v3") >= "A-"
  require endpoint.retention == "none"
  require model.lifecycle in ["approved", "default"]

  prefer session_affinity weight 0.25
  prefer provider_diversity weight 0.05

  optimize quality 0.45 latency 0.20 tcos 0.25 reliability 0.10

  fallback same_or_higher_quality max_attempts 2
  switch only_if expected_gain >= 0.08 min_hold_seconds 300
}
```

## 语义

| 子句 | 类型 | 语义 |
|---|---|---|
| `require <expr>` | hard gate | 不可违反；全部通过才进入候选集 |
| `prefer <feature> weight <w>` | soft | 加分项（session affinity、provider 多样性等） |
| `optimize <dim> <w>...` | soft | 多目标加权；权重之和应为 1.0 |
| `fallback <class> max_attempts <n>` | 执行 | none / same_or_higher_quality / any_compliant / degrade_allowed |
| `switch only_if expected_gain >= <g> [min_hold_seconds <s>]` | 稳定性 | route hysteresis，防 flapping |

可优化维度：`quality`、`latency`、`tcos`、`reliability`、`energy`、`capacity`。
高风险任务可声明 `optimize lexicographic quality safety latency tcos`
（按顺序字典序优化，而非加权）。

## 可引用命名空间

- `data.*` — region、classification
- `model.*` — family、version、eval("<task>")、lifecycle、age_days、capability(...)
- `endpoint.*` — region、provider、retention、health、capacity_util、protocol
- `contract.*` — task、domain、modality、deadline_ms

## 治理

- 每个 policy 有不可变版本号；RoutingDecision 必须记录 `policy_version`。
- DSL 解析器在 NOETIVELA-open（Apache），执行器在 core（AGPL），
  simulation / impact 预览在 ee。
