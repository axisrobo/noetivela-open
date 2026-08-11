# NOETIVELA-open

**Model, Routing & Inference Fabric — Open Integration Surface**
AxisRobo Architecture & Research Program

> **中文文档:** [README.zh-CN.md](README.zh-CN.md)

NOETIVELA is an enterprise inference fabric that catalogs models and endpoints,
converts task requirements into **governed inference contracts**, and routes each
request across cloud, private and edge intelligence using policy, quality,
latency, reliability, locality and total-cost evidence.

This repository is NOETIVELA's **open integration surface**, licensed under
**Apache-2.0** so that any product (including closed-source commercial ones) can
integrate it freely.

| Repository | Role | License |
|---|---|---|
| **NOETIVELA-open (this repo)** | SDKs, examples, authoritative API specs, CLI/binaries, public docs | Apache-2.0 |
| [NOETIVELA](../NOETIVELA) | Core engine implementation (registry, routing, gateway, telemetry) | AGPL-3.0 |
| [NOETIVELA-ee](../NOETIVELA-ee) | Enterprise edition: multi-tenancy, HA, governance, learned routing | Commercial |

## What's in this repository

| Directory | Contents |
|---|---|
| `contracts/` | **Authoritative API specs**: OpenAPI (data/control plane), JSON Schema for the InferenceContract / ModelProfile / RoutingDecision, Policy DSL grammar |
| `sdk/` | Official SDKs: Go / Python / TypeScript (Java / Rust planned) |
| `examples/` | End-to-end examples — quickstart (contract → validate → eligible → chat → decision trace); more planned (contract-review-agent, multi-agent-swe, edge-robotics, model-retirement) |
| `cli/` | `noetivela` CLI source and release notes for prebuilt binaries |

Public user documentation (concepts, quickstart, migration guide, capability
matrix) is under active development and will be published on the docs site.
Internal design and development docs live in the NOETIVELA-ee repository.

## What's NOT in this repository

- The core engine implementation (in NOETIVELA core, AGPL).
- Enterprise features (multi-tenancy, HA, TCoS/FinOps, learned routing, etc.,
  in NOETIVELA-ee).
- Any provider credential or customer configuration.

## Quickstart

```bash
# 1. Start the control plane (bootstrap a sample catalog) and the data plane
#    (a sample routing policy)
NOETIVELA_BOOTSTRAP_FILE=../NOETIVELA/deploy/bootstrap/manifest.example.json \
    go run ../NOETIVELA/backend/cmd/noetivela-controller &   # :8081
NOETIVELA_POLICY_FILE=../NOETIVELA/deploy/policies/legal.confidential.npl \
    go run ../NOETIVELA/backend/cmd/noetivela-gateway &       # :8080

# 2. Submit an Inference Contract via the Go SDK (no hardcoded model names)
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

// Token-free satisfiability check + ranking
res, _ := c.ValidateContract(ctx, contract, "legal-confidential-v2")
ranked, _ := c.EligibleCandidates(ctx, contract, "legal-confidential-v2")

// Governed inference request; returns result + RoutingDecision + TCoS
chat, _ := c.Chat(ctx, "auto", []client.Message{{Role: "user", Content: "..."}})
fmt.Println(chat.Noetivela.EndpointRef, chat.Noetivela.TCoS.Total)
```

## Version alignment

The three repositories share the same semver minor line. This repository's
`contracts/` is the authoritative release point for the API specs; core and ee
pin fixed versions. Breaking changes must first be marked deprecated here.

## License

Apache License 2.0 — see [LICENSE](LICENSE).
