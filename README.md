# NOETIVELA

**Model, Routing & Inference Fabric**
AxisRobo Architecture & Research Program

> **中文文档:** [README.zh-CN.md](README.zh-CN.md)

## What is NOETIVELA?

NOETIVELA is an **enterprise inference fabric** that treats model selection as a
**governed decision**, not a guess. It catalogs the models and endpoints you
already run (cloud APIs, private NIM/vLLM, edge), turns each task requirement into
an **Inference Contract**, and routes every request across that catalog using
policy, quality, latency, reliability, locality and total-cost evidence.

It is **not** an OpenAI-compatible proxy, and it is **not** a "cheap-model auto
switcher". Those tools add a layer; NOETIVELA adds a control plane.

### The problems it solves

- **Provider fragmentation.** Teams chase the best model across cloud, private
  and edge endpoints — and end up hard-coding model names into applications,
  locking them to a single vendor.
- **No governance in the routing path.** Quality/cost heuristics happily route
  requests into regions, providers or data classes that policy forbids — because
  scoring has no idea what the *hard gates* are.
- **Opaque decisions.** When the wrong model answers, nobody can explain why it
  was chosen — no audit trail, no replay, no fix.
- **Runaway cost.** Without per-task cost accounting, spend on frontier models
  is invisible until the invoice arrives.
- **Credential sprawl.** Applications hold provider keys directly; tenant
  isolation, rotation and blast radius are an afterthought.
- **Model churn.** Canary, shadow, deprecate and retire are manual, risky
  rituals — so stale models stay in production.

### Core features

| Capability | What it does |
|---|---|
| **Model & endpoint lifecycle** | Models, versions, deployments and endpoints are first-class, governed objects — canaried, shadowed, deprecated and retired like code. |
| **Inference Contract** | The request declares the *need* (`task`, `modality`, regions, data classification, session) — not the model. Applications never hard-code a model name. |
| **Hard-gate eligibility** | Security, residency, freshness and compatibility gates are evaluated first. A violation **never** gets compensated by weighted scoring; ineligible candidates are simply excluded. |
| **Multi-objective routing** | Among the eligible set, routing optimizes quality, latency, reliability, locality and total cost — via a readable Routing Policy DSL or learned models. |
| **TCoS evidence** | Every decision carries a **total cost of serving** breakdown and quality-adjusted economics, so routing is provably cheaper than a fixed frontier baseline. |
| **Explainable RoutingDecision** | Each request produces an auditable decision trace: why this candidate, what the alternatives were, and what it cost. Replay the ledger to test any policy counterfactually. |
| **Governance & security** | Tenant-scoped credential isolation (apps never hold keys), budget/quota ceilings that are never routed around, and deny-by-default authorization. |

> For a deeper walkthrough of the routing pipeline, object model and economics,
> see the [NOETIVELA core README](../NOETIVELA).

## NOETIVELA-open — this repository

NOETIVELA is built as a three-repository family. This repository is its **open
integration surface**, licensed under **Apache-2.0** so that any product
(including closed-source commercial ones) can integrate it freely.

| Repository | Role | License |
|---|---|---|
| **NOETIVELA-open (this repo)** | SDKs, examples, authoritative API specs, CLI/binaries, public docs | Apache-2.0 |
| [NOETIVELA](../NOETIVELA) | Core engine implementation (registry, routing, gateway, telemetry) | AGPL-3.0 |
| [NOETIVELA-ee](../NOETIVELA-ee) | Enterprise edition: multi-tenancy, HA, governance, learned routing | Commercial |

### What's in this repository

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

## CLI & Binaries

The `noetivela` CLI talks to a running gateway/controller and drives the whole
governed pipeline: contract → validate → eligible → chat → decision trace.

**Install** — download the prebuilt binary for your platform from
[GitHub Releases](../../releases) (linux / darwin / windows × amd64 / arm64,
with `checksums.txt`), or build from source:

```bash
git clone https://github.com/axisrobo/noetivela-open.git
cd noetivela-open/cli
go build -o noetivela .      # then place it on your PATH
```

**Usage** (requires a running gateway — see Quickstart):

```bash
export NOETIVELA_URL=http://localhost:8080   # default: http://localhost:8080

noetivela version
noetivela contract validate -contract contract.json -policy legal-confidential-v2
noetivela eligible -contract contract.json -policy legal-confidential-v2
noetivela chat -contract contract.json -prompt "hi" -policy legal-confidential-v2
noetivela decisions -limit 50
noetivela usage
noetivela replay -baseline fixed:gpt-x -policy legal-confidential-v2 -limit 1000
noetivela train -version v2-table
```

An example InferenceContract JSON is at [`cli/contract.example.json`](cli/contract.example.json).

## Version alignment

The three repositories share the same semver minor line. This repository's
`contracts/` is the authoritative release point for the API specs; core and ee
pin fixed versions. Breaking changes must first be marked deprecated here.

## License

Apache License 2.0 — see [LICENSE](LICENSE).
