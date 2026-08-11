# Changelog

All notable changes to NOETIVELA-open (SDKs, contracts, examples, CLI).
Format follows [Keep a Changelog](https://keepachangelog.com/), versioning follows semver,
aligned with NOETIVELA core / ee minor lines.

## [Unreleased]

### Added
- (next milestone — see 0.1.0-alpha)

## [0.1.0-alpha] - 2026-08-10

### Added
- Repository plan: contracts (authoritative API specs), SDK layout (Go/Python/TS),
  examples, CLI, public docs (docs/plan.md).
- Authoritative contracts v0.1: InferenceContract / ModelProfile /
  RoutingDecision JSON Schemas, Routing Policy DSL (spec + EBNF grammar),
  OpenAI-compatible data-plane OpenAPI skeleton with TCoS breakdown and
  unsatisfiable-contract semantics.
- `sdk/go/policy`: Apache-licensed Routing Policy DSL implementation
  (lexer, recursive-descent parser, AST, validate) with tests. Core (AGPL)
  depends on this package as the authoritative grammar.
- `sdk/go/client`: Apache-licensed gateway client — Chat, ChatStream (SSE),
  ValidateContract, EligibleCandidates, GetDecision, Usage; contract bound
  via `WithContract` and `X-Noetivela-*` headers; unsatisfiable contracts
  surfaced as typed errors. Tested against a mock gateway.
- `examples/quickstart`: end-to-end contract → validate → eligible → chat →
  decision-trace program (go module, builds standalone).
- `sdk/go/client.Embeddings`: governed embedding inference; OpenAPI updated
  with the full `/v1/embeddings` surface.
- `sdk/go/client.Decisions`: paginated routing-decision listing.
- `cli/`: `noetivela` command — `contract validate`, `eligible`, `chat`,
  `decisions`, `usage` against a running gateway; `contract.example.json`;
- `contracts/openapi/control-plane.yaml`: authoritative control-plane spec
  (registry CRUD, deployment, lifecycle transitions + plans) with the
  lifecycle state machine documented.
- `sdk/go/client.Replay`: counterfactual replay (quality-adjusted TCoS vs a
  fixed baseline); CLI `replay` command and `version` subcommand;
  `.goreleaser.yaml` for multi-platform CLI releases.
- `sdk/python`: zero-dependency Python client (stdlib urllib) — Contract
  dataclasses, chat/chat_stream/validate/eligible/embeddings/decisions/
  get_decision/usage/replay, typed UnsatisfiableError; pytest suite against
  a mock gateway; pyproject + README.
- `sdk/typescript`: zero-dependency TS client (global fetch, Node ≥ 18) —
  typed Contract/response models, chat/chatStream/validate/eligible/
  embeddings/decisions/getDecision/usage/replay; node:test suite; CI jobs
  for both SDKs added.
- `sdk/go`, `sdk/python`, `sdk/typescript`: `Decisions`/`decisions` support
  task/model filters; `Train`/`train` added; CLI gains `train` and
  `decisions -task/-model` filters.
- `.github/workflows/ci.yml`: sdk-go + examples jobs.
- `contracts/go/extension`: Apache-licensed enterprise extension interfaces
  (Store, Scorer, CredentialResolver, Sink, Sample/UsageRecord) — the EE
  layering surface that never imports AGPL core.
- SDK parity: `Train`/`train` in Go/Python/TS accept an explicit evidence
  dataset upload (`samples`) for TEKMOVELA offline training; versions
  aligned to `0.1.0-alpha` (Go modules keep `v0.1.0`).

[0.1.0-alpha]: https://github.com/axisrobo/NOETIVELA-open/releases/tag/v0.1.0-alpha
