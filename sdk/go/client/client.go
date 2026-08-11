// Package client is the Apache-2.0 Go SDK for NOETIVELA.
//
// Applications submit Inference Contracts instead of hard-coding a provider
// model ID. Provider credentials and endpoint URLs never appear in this
// client: everything flows through the governed gateway.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)
// Contract mirrors the authoritative InferenceContract JSON Schema
// (contracts/schema/inference-contract.json).
type Contract struct {
	Task     string   `json:"task"`
	Domain   string   `json:"domain,omitempty"`
	Language []string `json:"language,omitempty"`
	Modality []string `json:"modality"`

	ResponseSchema string    `json:"response_schema,omitempty"`
	Quality        *Quality  `json:"quality,omitempty"`
	Latency        *Latency  `json:"latency,omitempty"`
	Cost           *Cost     `json:"cost,omitempty"`
	Context        *Context  `json:"context,omitempty"`
	Policy         *Policy   `json:"policy,omitempty"`
	Reliability    *Reliability `json:"reliability,omitempty"`
}

type Quality struct {
	MinimumEvalGrade string `json:"minimum_eval_grade,omitempty"`
	GroundingRequired bool  `json:"grounding_required,omitempty"`
}

type Latency struct {
	TTFTMs    int  `json:"ttft_ms,omitempty"`
	P95MS     int  `json:"p95_ms,omitempty"`
	DeadlineMS int `json:"deadline_ms,omitempty"`
	Streaming bool `json:"streaming,omitempty"`
}

type Cost struct {
	CeilingPerRequest string `json:"ceiling_per_request,omitempty"`
	BudgetAccount     string `json:"budget_account,omitempty"`
}

type Context struct {
	SessionID          string `json:"session_id,omitempty"`
	CacheKey           string `json:"cache_key,omitempty"`
	PreferCacheLocality bool  `json:"prefer_cache_locality,omitempty"`
}

type Policy struct {
	DataClassification string   `json:"data_classification,omitempty"`
	AllowedRegions     []string `json:"allowed_regions,omitempty"`
	DeniedProviders    []string `json:"denied_providers,omitempty"`
	ProviderTraining   string   `json:"provider_training,omitempty"`
	Retention          string   `json:"retention,omitempty"`
}

type Reliability struct {
	RetryBudget int    `json:"retry_budget,omitempty"`
	Fallback    string `json:"fallback,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Usage struct {
	InputTokens     int64 `json:"input_tokens"`
	OutputTokens    int64 `json:"output_tokens"`
	CachedTokens    int64 `json:"cached_tokens"`
	ReasoningTokens int64 `json:"reasoning_tokens"`
}

type TCoS struct {
	DirectInference float64 `json:"direct_inference"`
	RouterOverhead  float64 `json:"router_overhead"`
	ContextMovement float64 `json:"context_movement"`
	CacheLoss       float64 `json:"cache_loss"`
	ReliabilityCost float64 `json:"reliability_cost"`
	Total           float64 `json:"total"`
}

type RoutingEvidence struct {
	DecisionID  string `json:"decision_id"`
	ModelRef    string `json:"model_ref"`
	EndpointRef string `json:"endpoint_ref"`
	TCoS        TCoS   `json:"tcos"`
}

type ChatResponse struct {
	Content      string           `json:"content"`
	FinishReason string           `json:"finish_reason"`
	Usage        Usage            `json:"usage"`
	Noetivela    RoutingEvidence  `json:"noetivela"`
}

type ChatChunk struct {
	Delta        string `json:"delta"`
	FinishReason string `json:"finish_reason"`
	Usage        *Usage `json:"usage,omitempty"`
}

type EligibleCandidate struct {
	ModelRef     string   `json:"model_ref"`
	EndpointRef  string   `json:"endpoint_ref"`
	Eligible     bool     `json:"eligible"`
	FilterReasons []string `json:"filter_reasons,omitempty"`
	Score        float64  `json:"score,omitempty"`
}

type UnsatisfiableError struct {
	Type        string   `json:"type"`
	Message     string   `json:"message"`
	FailedGates []string `json:"failed_gates,omitempty"`
}

func (e *UnsatisfiableError) Error() string { return e.Message }

type Error struct {
	StatusCode int
	Body       string
}

func (e *Error) Error() string { return fmt.Sprintf("noetivela: HTTP %d: %s", e.StatusCode, e.Body) }

// Client talks to a NOETIVELA gateway.
type Client struct {
	BaseURL    string
	HTTP       *http.Client
	Contract   *Contract
	PolicyName string
	Tenant     string
	Principal  string
}

func New(baseURL string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP:    &http.Client{Timeout: 120 * time.Second},
	}
}

// WithContract returns a clone bound to a default contract for convenience.
func (c *Client) WithContract(ct *Contract) *Client {
	cp := *c
	cp.Contract = ct
	return &cp
}

func (c *Client) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Contract != nil {
		raw, err := json.Marshal(c.Contract)
		if err != nil {
			return nil, err
		}
		req.Header.Set("X-Noetivela-Contract", string(raw))
	}
	if c.PolicyName != "" {
		req.Header.Set("X-Noetivela-Policy", c.PolicyName)
	}
	if c.Tenant != "" {
		req.Header.Set("X-Noetivela-Tenant", c.Tenant)
	}
	if c.Principal != "" {
		req.Header.Set("X-Noetivela-Principal", c.Principal)
	}
	return req, nil
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	req, err := c.newRequest(ctx, method, path, body)
	if err != nil {
		return err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &Error{StatusCode: resp.StatusCode, Body: string(raw)}
	}
	if out != nil {
		return json.Unmarshal(raw, out)
	}
	return nil
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream,omitempty"`
}

type chatResponseEnvelope struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage     Usage           `json:"usage"`
	Noetivela RoutingEvidence `json:"noetivela"`
}

// Chat sends a governed inference request. model is a NOETIVELA alias or
// "auto" for contract-driven routing — never a provider model ID.
func (c *Client) Chat(ctx context.Context, model string, messages []Message) (*ChatResponse, error) {
	var env chatResponseEnvelope
	err := c.do(ctx, http.MethodPost, "/v1/chat/completions", &chatRequest{Model: model, Messages: messages}, &env)
	if err != nil {
		return nil, mapError(err)
	}
	if len(env.Choices) == 0 {
		return nil, errors.New("noetivela: empty response")
	}
	return &ChatResponse{
		Content:      env.Choices[0].Message.Content,
		FinishReason: env.Choices[0].FinishReason,
		Usage:        env.Usage,
		Noetivela:    env.Noetivela,
	}, nil
}

// ChatStream streams chunks via SSE; the returned channel closes after the
// final event. Errors from the stream surface as InvokeErrorKind on the
// channel.
func (c *Client) ChatStream(ctx context.Context, model string, messages []Message) (<-chan ChatChunk, <-chan error) {
	ch := make(chan ChatChunk)
	errCh := make(chan error, 1)
	go func() {
		defer close(ch)
		req, err := c.newRequest(ctx, http.MethodPost, "/v1/chat/completions", &chatRequest{Model: model, Messages: messages, Stream: true})
		if err != nil {
			errCh <- err
			return
		}
		resp, err := c.HTTP.Do(req)
		if err != nil {
			errCh <- err
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			errCh <- &Error{StatusCode: resp.StatusCode, Body: string(b)}
			return
		}
		scanner := newSSEScanner(resp.Body)
		for {
			event, ok := scanner.Next()
			if !ok {
				break
			}
			var done struct {
				Done bool `json:"done"`
			}
			if err := json.Unmarshal([]byte(event), &done); err == nil && done.Done {
				break
			}
			var chunk ChatChunk
			if err := json.Unmarshal([]byte(event), &chunk); err != nil {
				errCh <- err
				return
			}
			ch <- chunk
		}
	}()
	return ch, errCh
}

// ValidateContract checks feasibility without consuming any tokens.
func (c *Client) ValidateContract(ctx context.Context, ct *Contract, policyName string) (*ValidateResult, error) {
	var out ValidateResult
	cp := *c
	cp.Contract = ct
	cp.PolicyName = policyName
	err := cp.do(ctx, http.MethodPost, "/v1/contracts/validate", nil, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type ValidateResult struct {
	Valid       bool               `json:"valid"`
	Satisfiable bool               `json:"satisfiable"`
	Candidates  []EligibleCandidate `json:"candidates"`
	Policy      string             `json:"policy"`
}

// EligibleCandidates returns the ranked eligible set for a contract without
// invoking any endpoint.
func (c *Client) EligibleCandidates(ctx context.Context, ct *Contract, policyName string) ([]EligibleCandidate, error) {
	var out struct {
		Ranked []EligibleCandidate `json:"ranked"`
	}
	cp := *c
	cp.Contract = ct
	cp.PolicyName = policyName
	err := cp.do(ctx, http.MethodPost, "/v1/eligible", nil, &out)
	if err != nil {
		return nil, err
	}
	return out.Ranked, nil
}

// GetDecision retrieves the stored RoutingDecision for a prior request.
func (c *Client) GetDecision(ctx context.Context, decisionID string) (json.RawMessage, error) {
	var out json.RawMessage
	err := c.do(ctx, http.MethodGet, "/v1/decisions/"+decisionID, nil, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type DecisionSummary struct {
	ID        string          `json:"decision_id"`
	Task      string          `json:"task,omitempty"`
	Policy    string          `json:"policy_version"`
	Selection json.RawMessage `json:"selection,omitempty"`
}

type DecisionsOptions struct {
	Limit  int
	Offset int
	Task   string
	Model  string
}

// Decisions lists routed decisions with pagination and optional filters.
func (c *Client) Decisions(ctx context.Context, opts DecisionsOptions) ([]DecisionSummary, error) {
	q := fmt.Sprintf("limit=%d&offset=%d", opts.Limit, opts.Offset)
	if opts.Task != "" {
		q += "&task=" + opts.Task
	}
	if opts.Model != "" {
		q += "&model=" + opts.Model
	}
	var out struct {
		Decisions []DecisionSummary `json:"decisions"`
	}
	err := c.do(ctx, http.MethodGet, "/v1/decisions?"+q, nil, &out)
	if err != nil {
		return nil, err
	}
	return out.Decisions, nil
}

type TrainResult struct {
	VersionID      string  `json:"version_id"`
	TrainedSamples int     `json:"trained_samples"`
	ValidateMAE    float64 `json:"validate_mae,omitempty"`
	MinConfidence  float64 `json:"min_confidence"`
}

// TrainSample is one evidence-feedback record uploaded for training.
// Features are prompt-free (task/domain/length/schema/safety only).
type TrainSample struct {
	Features     map[string]any `json:"features"`
	CandidateRef string         `json:"candidate_ref"`
	Quality      float64        `json:"quality"`
	TCoS         float64        `json:"tcos"`
}

// TrainOptions controls the training loop.
type TrainOptions struct {
	VersionID   string
	MinSamples  int
	TCoSWeight  float64
	HoldOutFrac float64
	// Samples is an optional explicit dataset (TEKMOVELA offline evidence);
	// when nil the gateway's online collector is used.
	Samples []TrainSample
}

// Train fits the learned router on the gateway (online collector or the
// uploaded dataset) and registers a new router model version.
func (c *Client) Train(ctx context.Context, opts TrainOptions) (*TrainResult, error) {
	var out TrainResult
	body := map[string]any{
		"version_id":     opts.VersionID,
		"min_samples":    opts.MinSamples,
		"tcos_weight":    opts.TCoSWeight,
		"hold_out_frac":  opts.HoldOutFrac,
	}
	if len(opts.Samples) > 0 {
		body["samples"] = opts.Samples
	}
	err := c.do(ctx, http.MethodPost, "/v1/train", body, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type StrategyResult struct {
	Strategy       string  `json:"strategy"`
	TotalTCoS      float64 `json:"total_tcos"`
	TotalQuality   float64 `json:"total_quality"`
	QualityAdjTCoS float64 `json:"quality_adjusted_tcos"`
	ServedCount    int     `json:"served_count"`
	DeniedCount    int     `json:"denied_count"`
}

type ReplayResult struct {
	Records        int            `json:"records"`
	Baseline       StrategyResult `json:"baseline"`
	Routed         StrategyResult `json:"routed"`
	SavingsPercent float64        `json:"savings_percent"`
	Verdict        string         `json:"verdict"`
}

// Replay counterfactually re-routes recorded usage through the gateway's
// policy router and compares against a fixed baseline endpoint. This is the
// acceptance evidence for quality-adjusted TCoS savings.
func (c *Client) Replay(ctx context.Context, baselineEndpoint, policyName string, limit int) (*ReplayResult, error) {
	var out ReplayResult
	err := c.do(ctx, http.MethodPost, "/v1/replay", map[string]any{
		"baseline_endpoint": baselineEndpoint,
		"policy":            policyName,
		"limit":             limit,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type EmbeddingResponse struct {
	Vectors    [][]float32 `json:"data"`
	Dimensions int         `json:"dimensions"`
	Usage      Usage       `json:"usage"`
	Noetivela  RoutingEvidence `json:"noetivela"`
}

type embeddingRequest struct {
	Model      string   `json:"model"`
	Input      []string `json:"input"`
	Dimensions int      `json:"dimensions,omitempty"`
}

// Embeddings sends a governed embedding request. Only embedding-capable
// endpoints are eligible; the chosen endpoint is a routing decision.
func (c *Client) Embeddings(ctx context.Context, model string, input []string, dimensions int) (*EmbeddingResponse, error) {
	var out struct {
		Data       [][]float32     `json:"data"`
		Dimensions int             `json:"dimensions"`
		Usage      Usage           `json:"usage"`
		Noetivela  RoutingEvidence `json:"noetivela"`
	}
	err := c.do(ctx, http.MethodPost, "/v1/embeddings", &embeddingRequest{Model: model, Input: input, Dimensions: dimensions}, &out)
	if err != nil {
		return nil, mapError(err)
	}
	return &EmbeddingResponse{Vectors: out.Data, Dimensions: out.Dimensions, Usage: out.Usage, Noetivela: out.Noetivela}, nil
}

// Usage lists the gateway's TCoS ledger.
func (c *Client) Usage(ctx context.Context) ([]json.RawMessage, error) {
	var out []json.RawMessage
	err := c.do(ctx, http.MethodGet, "/v1/usage", nil, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func mapError(err error) error {
	var e *Error
	if !errors.As(err, &e) {
		return err
	}
	var wrapped struct {
		Error UnsatisfiableError `json:"error"`
	}
	if json.Unmarshal([]byte(e.Body), &wrapped) == nil && wrapped.Error.Type == "unsatisfiable_contract" {
		ue := wrapped.Error
		return &ue
	}
	return e
}
