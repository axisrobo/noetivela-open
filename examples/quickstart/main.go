// Command quickstart is the NOETIVELA end-to-end example: submit an
// Inference Contract, get an explainable routing decision, call the model,
// and read back the TCoS ledger.
//
// Prereq: a running gateway, e.g.
//
//	NOETIVELA_POLICY_FILE=deploy/policies/legal.confidential.npl \
//	    go run ./cmd/noetivela-gateway
//
//	NOETIVELA_BOOTSTRAP_FILE=deploy/bootstrap/manifest.example.json \
//	    go run ./cmd/noetivela-controller
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/axisrobo/noetivela-open/sdk/go/client"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	baseURL := os.Getenv("NOETIVELA_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	c := client.New(baseURL)

	// 1. Declare what intelligence the task needs — never which model.
	contract := &client.Contract{
		Task:     "contract_clause_extraction",
		Domain:   "legal",
		Modality: []string{"text"},
		ResponseSchema: "legal_clause_v3",
		Quality: &client.Quality{MinimumEvalGrade: "A-"},
		Latency: &client.Latency{P95MS: 3500},
		Policy: &client.Policy{
			DataClassification: "confidential",
			AllowedRegions:     []string{"sg", "cn-north"},
			ProviderTraining:   "prohibited",
			Retention:          "none",
		},
		Context: &client.Context{SessionID: "case-8821", PreferCacheLocality: true},
	}

	// 2. Check feasibility without consuming any tokens.
	res, err := c.ValidateContract(ctx, contract, "legal-confidential-v2")
	if err != nil {
		fatal("validate", err)
	}
	fmt.Printf("contract satisfiable: %v (policy %q)\n", res.Satisfiable, res.Policy)
	for _, cand := range res.Candidates {
		fmt.Printf("  candidate model=%s endpoint=%s eligible=%v\n", cand.ModelRef, cand.EndpointRef, cand.Eligible)
	}

	// 3. Inspect the ranked eligible set before sending traffic.
	ranked, err := c.EligibleCandidates(ctx, contract, "legal-confidential-v2")
	if err != nil {
		fatal("eligible", err)
	}
	fmt.Println("ranked eligible candidates:")
	for _, cand := range ranked {
		fmt.Printf("  %s @ %s score=%.3f\n", cand.ModelRef, cand.EndpointRef, cand.Score)
	}

	// 4. Send the governed request (routing is a policy decision, not a model name).
	cc := c.WithContract(contract)
	cc.PolicyName = "legal-confidential-v2"
	chat, err := cc.Chat(ctx, "auto", []client.Message{
		{Role: "system", Content: "You extract clauses from contracts."},
		{Role: "user", Content: "Extract the indemnity clause from this contract: ..."},
	})
	if err != nil {
		fatal("chat", err)
	}
	fmt.Printf("model=%s endpoint=%s decision=%s\n", chat.Noetivela.ModelRef, chat.Noetivela.EndpointRef, chat.Noetivela.DecisionID)
	fmt.Printf("content: %s\n", chat.Content)
	fmt.Printf("usage: %+v\n", chat.Usage)
	fmt.Printf("tcos: %.6f\n", chat.Noetivela.TCoS.Total)

	// 5. Read the explainable decision trace back.
	trace, err := c.GetDecision(ctx, chat.Noetivela.DecisionID)
	if err != nil {
		fatal("decision", err)
	}
	fmt.Printf("decision trace: %s\n", trace)
}

func fatal(step string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", step, err)
	os.Exit(1)
}
