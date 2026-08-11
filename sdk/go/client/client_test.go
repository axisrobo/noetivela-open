package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/axisrobo/noetivela-open/sdk/go/client"
)

// mockGateway mimics the NOETIVELA data-plane surface the SDK talks to. It
// is contract-driven and returns a RoutingDecision each time.
func mockGateway(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/chat/completions":
			if r.Header.Get("X-Noetivela-Contract") == "" {
				t.Errorf("contract header missing")
			}
			var body struct {
				Stream bool `json:"stream"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Stream {
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = w.Write([]byte("data: {\"delta\":\"Hel\",\"finish_reason\":\"\"}\n\n"))
				_, _ = w.Write([]byte("data: {\"delta\":\"lo\",\"finish_reason\":\"stop\",\"usage\":{\"input_tokens\":10,\"output_tokens\":2,\"cached_tokens\":3,\"reasoning_tokens\":0}}\n\n"))
				_, _ = w.Write([]byte("data: {\"done\":true}\n\n"))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
			  "choices":[{"message":{"content":"ok"},"finish_reason":"stop"}],
			  "usage":{"input_tokens":100,"output_tokens":20,"cached_tokens":40,"reasoning_tokens":5},
			  "noetivela":{"decision_id":"abc123","model_ref":"m1","endpoint_ref":"e1",
			    "tcos":{"direct_inference":0.0002,"total":0.0002}}
			}`))
		case "/v1/contracts/validate":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"valid":true,"satisfiable":true,"policy":"legal-confidential-v2",
			  "candidates":[{"model_ref":"m1","endpoint_ref":"e1","eligible":true,"score":0.8}]}`))
		case "/v1/eligible":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ranked":[{"model_ref":"m1","endpoint_ref":"e1","eligible":true,"score":0.8}]}`))
		case "/v1/embeddings":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":[[0.1,0.2],[0.3,0.4]],"dimensions":2,"usage":{"input_tokens":4,"output_tokens":0,"cached_tokens":0,"reasoning_tokens":0},"noetivela":{"decision_id":"emb1","model_ref":"m1","endpoint_ref":"e1","tcos":{"total":0.0001}}}`))
		case "/v1/replay":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"records":10,"baseline":{"strategy":"fixed:exp","quality_adjusted_tcos":0.02,"served_count":10,"denied_count":0},"routed":{"strategy":"policy_routed","quality_adjusted_tcos":0.01,"served_count":10,"denied_count":0},"savings_percent":50.0,"verdict":"PASS"}`))
		case "/v1/train":
			var body struct {
				VersionID  string              `json:"version_id"`
				MinSamples int                 `json:"min_samples"`
				Samples    []client.TrainSample `json:"samples"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.VersionID != "v2" {
				t.Errorf("missing version_id, got %q", body.VersionID)
			}
			if len(body.Samples) != 2 {
				t.Errorf("expected 2 dataset samples, got %d", len(body.Samples))
			}
			if body.Samples[0].CandidateRef != "ep-openai-sg/v1" || body.Samples[0].Features["task"] != "contract_clause_extraction" {
				t.Errorf("unexpected dataset sample %+v", body.Samples[0])
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version_id":"v2","trained_samples":2,"validate_mae":0.12,"min_confidence":0.6}`))
		case "/v1/decisions/abc123":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"decision_id":"abc123","candidates":[],"selection":{"endpoint_ref":"e1"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestChat(t *testing.T) {
	srv := mockGateway(t)
	defer srv.Close()

	c := client.New(srv.URL).WithContract(&client.Contract{
		Task: "contract_clause_extraction", Modality: []string{"text"},
		Policy: &client.Policy{AllowedRegions: []string{"sg"}},
	})
	resp, err := c.Chat(context.Background(), "auto", []client.Message{{Role: "user", Content: "hi"}})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "ok" {
		t.Fatalf("unexpected content %q", resp.Content)
	}
	if resp.Noetivela.EndpointRef != "e1" || resp.Noetivela.DecisionID != "abc123" {
		t.Fatalf("unexpected routing evidence %+v", resp.Noetivela)
	}
	if resp.Usage.CachedTokens != 40 || resp.Usage.ReasoningTokens != 5 {
		t.Fatalf("unexpected usage %+v", resp.Usage)
	}
}

func TestChatStream(t *testing.T) {
	srv := mockGateway(t)
	defer srv.Close()

	c := client.New(srv.URL).WithContract(&client.Contract{Task: "chat", Modality: []string{"text"}})
	ch, errCh := c.ChatStream(context.Background(), "auto", []client.Message{{Role: "user", Content: "hi"}})

	var got strings.Builder
	var gotUsage *int64
	for chunk := range ch {
		got.WriteString(chunk.Delta)
		if chunk.Usage != nil {
			u := chunk.Usage.OutputTokens
			gotUsage = &u
		}
	}
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatal(err)
		}
	default:
	}
	if got.String() != "Hello" {
		t.Fatalf("unexpected streamed content %q", got.String())
	}
	if gotUsage == nil || *gotUsage != 2 {
		t.Fatalf("unexpected streamed usage %v", gotUsage)
	}
}

func TestValidateContract(t *testing.T) {
	srv := mockGateway(t)
	defer srv.Close()

	c := client.New(srv.URL)
	res, err := c.ValidateContract(context.Background(), &client.Contract{Task: "t", Modality: []string{"text"}}, "legal-confidential-v2")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Valid || !res.Satisfiable {
		t.Fatalf("expected satisfiable contract, got %+v", res)
	}
	if res.Policy != "legal-confidential-v2" {
		t.Fatalf("unexpected policy echo %q", res.Policy)
	}
}

func TestEligibleCandidates(t *testing.T) {
	srv := mockGateway(t)
	defer srv.Close()

	c := client.New(srv.URL)
	cands, err := c.EligibleCandidates(context.Background(), &client.Contract{Task: "t", Modality: []string{"text"}}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != 1 || cands[0].EndpointRef != "e1" {
		t.Fatalf("unexpected candidates %+v", cands)
	}
}

func TestGetDecision(t *testing.T) {
	srv := mockGateway(t)
	defer srv.Close()

	c := client.New(srv.URL)
	raw, err := c.GetDecision(context.Background(), "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"selection"`) {
		t.Fatalf("unexpected decision payload %s", raw)
	}
}

func TestEmbeddings(t *testing.T) {
	srv := mockGateway(t)
	defer srv.Close()

	c := client.New(srv.URL)
	resp, err := c.Embeddings(context.Background(), "auto", []string{"a", "b"}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Vectors) != 2 || resp.Dimensions != 2 {
		t.Fatalf("unexpected embedding response %+v", resp)
	}
	if resp.Noetivela.EndpointRef != "e1" || resp.Usage.InputTokens != 4 {
		t.Fatalf("unexpected embedding evidence %+v usage %+v", resp.Noetivela, resp.Usage)
	}
}

func TestReplay(t *testing.T) {
	srv := mockGateway(t)
	defer srv.Close()

	c := client.New(srv.URL)
	res, err := c.Replay(context.Background(), "exp-ep", "legal", 0)
	if err != nil {
		t.Fatal(err)
	}
	if res.Records != 10 || res.Verdict != "PASS" {
		t.Fatalf("unexpected replay result %+v", res)
	}
	if res.SavingsPercent != 50.0 || res.Routed.QualityAdjTCoS >= res.Baseline.QualityAdjTCoS {
		t.Fatalf("unexpected replay economics %+v", res)
	}
}

func TestTrainWithDataset(t *testing.T) {
	srv := mockGateway(t)
	defer srv.Close()

	c := client.New(srv.URL)
	res, err := c.Train(context.Background(), client.TrainOptions{
		VersionID:  "v2",
		MinSamples: 10,
		Samples: []client.TrainSample{
			{Features: map[string]any{"task": "contract_clause_extraction", "input_tokens_est": 1200},
				CandidateRef: "ep-openai-sg/v1", Quality: 0.9, TCoS: 0.002},
			{Features: map[string]any{"task": "code_review", "input_tokens_est": 3000},
				CandidateRef: "ep-groq-sg/v1", Quality: 0.4, TCoS: 0.0001},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.VersionID != "v2" || res.TrainedSamples != 2 || res.MinConfidence != 0.6 {
		t.Fatalf("unexpected train result %+v", res)
	}
}
