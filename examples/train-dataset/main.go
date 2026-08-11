// Command train-dataset uploads an explicit offline evidence dataset
// (TEKMOVELA: judged answer quality + measured TCoS) to fit a new
// learned-router version on the gateway, instead of relying on the online
// collector. Features are prompt-free (task/domain/length/schema/safety).
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

	res, err := c.Train(ctx, client.TrainOptions{
		VersionID:   "v2",
		MinSamples:  10,
		TCoSWeight:  0.5,
		HoldOutFrac: 0.0,
		Samples: []client.TrainSample{
			{
				Features: map[string]any{
					"task":              "contract_clause_extraction",
					"domain":            "legal",
					"input_tokens_est":  1200,
					"has_session":       true,
					"tool_use_required": false,
					"structured_output": false,
					"has_deadline":      true,
					"has_latency_p95":   true,
				},
				CandidateRef: "ep-openai-sg/v1",
				Quality:      0.9, // judged answer quality (0..1)
				TCoS:         0.002, // measured total cost of service
			},
			{
				Features: map[string]any{
					"task":             "code_review",
					"domain":           "software",
					"input_tokens_est": 3000,
				},
				CandidateRef: "ep-groq-sg/v1",
				Quality:      0.4,
				TCoS:         0.0001,
			},
		},
	})
	if err != nil {
		fatal("train", err)
	}
	fmt.Printf("version=%s trained_samples=%d min_confidence=%.3f validate_mae=%.3f\n",
		res.VersionID, res.TrainedSamples, res.MinConfidence, res.ValidateMAE)
}

func fatal(step string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", step, err)
	os.Exit(1)
}
