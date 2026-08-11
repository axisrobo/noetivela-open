// Package extension defines the NOETIVELA extension-point interfaces that
// enterprise (NOETIVELA-ee) and third parties implement WITHOUT importing the
// AGPL core. Apache-2.0: this is the open integration surface.
//
// Dependency direction (see NOETIVELA docs/07-open-core-split.md):
//
//	open (extension) <- core (implements)   core <- ee (extends)
//
// core provides concrete implementations of these interfaces; ee layers
// tenancy, audit, adaptive scoring etc. on top by wrapping/implementing them.
package extension

import (
	"context"
	"errors"
)

// ---- registry / store ----

// ModelIdentity is the stable enterprise model object (provider model names
// are never stable; see ModelProfile in contracts/schema).
type ModelIdentity struct {
	ID       string   `json:"id"`
	Family   string   `json:"family"`
	Provider string   `json:"provider"`
}

// Endpoint is the subset of InferenceEndpoint extension points need.
type Endpoint struct {
	ID           string  `json:"id"`
	Region       string  `json:"region"`
	Provider     string  `json:"provider"`
	Retention    string  `json:"retention"`
	Health       string  `json:"health"`
	PricePerMTok float64 `json:"price_per_mtok"`
}

// Store is the registry access surface. Core implements it; EE wraps it with
// tenancy/audit/federation without reimplementing storage.
type Store interface {
	GetModel(id string) (ModelIdentity, error)
	ListModels() ([]ModelIdentity, error)
	GetEndpoint(id string) (Endpoint, error)
	ListEndpoints() ([]Endpoint, error)
}

// ErrNotFound is returned by stores when a key is absent.
var ErrNotFound = errors.New("extension: not found")

// ---- routing / scoring ----

// Sample is one evidence-feedback record used to train a learned router.
// Features are prompt-free (task/domain/length/schema/safety only).
type Sample struct {
	Tenant       string  `json:"tenant,omitempty"`
	Task         string  `json:"task"`
	Domain       string  `json:"domain,omitempty"`
	CandidateRef string  `json:"candidate_ref"`
	Quality      float64 `json:"quality"`
	TCoS         float64 `json:"tcos"`
}

// NotFoundError is the typed "not found" error for stores and resolvers.
type NotFoundError struct{ Msg string }

func (e *NotFoundError) Error() string { return "extension: not found: " + e.Msg }

// Scorer scores an eligible candidate. This is the adaptive-routing
// extension point: EE supplies learned/bandit scorers; core supplies the
// rule baseline. Higher is better.
type Scorer interface {
	Score(s Sample, ep Endpoint) float64
}

// ---- credentials ----

// CredentialResolver resolves provider credentials for a tenant+binding.
// EE backs it with a secret manager and short-lived tokens.
type CredentialResolver interface {
	Resolve(ctx context.Context, tenant, bindingRef string) (string, error)
}

// ---- telemetry ----

// Sink receives normalized usage records (TCoS, latency, quality signal).
// EE implements audit and FinOps sinks.
type Sink interface {
	Record(record UsageRecord) error
}

// UsageRecord is the normalized usage surface extension points consume.
type UsageRecord struct {
	DecisionID  string  `json:"decision_id"`
	Tenant      string  `json:"tenant"`
	Task        string  `json:"task"`
	EndpointRef string  `json:"endpoint_ref"`
	TotalTCoS   float64 `json:"total_tcos"`
	LatencyMS   float64 `json:"latency_ms"`
	Error       string  `json:"error,omitempty"`
}
