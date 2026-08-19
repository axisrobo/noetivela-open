// Command noetivela is the CLI for the NOETIVELA Inference Fabric.
// It talks to a running gateway/controller through the Apache-licensed SDK.
//
//   noetivela contract validate -contract contract.json [-policy p]
//   noetivela eligible -contract contract.json [-policy p]
//   noetivela chat -contract contract.json [-policy p] -prompt "..."
//   noetivela decisions [-limit n] [-offset n]
//   noetivela usage
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/axisrobo/noetivela-open/sdk/go/client"
)

var baseURL = flag.String("url", defaultURL(), "NOETIVELA gateway base URL (default $NOETIVELA_URL or http://localhost:8080)")

// version is injected at build time (goreleaser ldflags). Default aligns with
// the NOETIVELA release milestone.
var version = "1.0.0-rc"

func defaultURL() string {
	if v := os.Getenv("NOETIVELA_URL"); v != "" {
		return v
	}
	return "http://localhost:8080"
}

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	c := client.New(*baseURL)

	var err error
	switch args[0] {
	case "version":
		fmt.Println("noetivela " + version)
		return
	case "contract", "validate":
		err = runValidate(ctx, c, args)
	case "eligible":
		err = runEligible(ctx, c, args)
	case "chat":
		err = runChat(ctx, c, args)
	case "decisions":
		err = runDecisions(ctx, c, args)
	case "replay":
		err = runReplay(ctx, c, args)
	case "train":
		err = runTrain(ctx, c, args)
	case "usage":
		err = runUsage(ctx, c)
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `usage: noetivela <command> [flags]

commands:
  contract validate -contract file.json [-policy name]   check feasibility (no tokens)
  eligible -contract file.json [-policy name]            ranked eligible candidates
  chat -contract file.json [-prompt text]                governed inference request
  decisions [-limit n] [-offset n] [-task t] [-model m]   list routing decisions (filtered)
  replay -baseline endpoint [-policy name] [-limit n]    counterfactual replay vs fixed baseline
  train -version v [-min-samples n] [-tcos-weight w]     fit/register the learned router
  usage                                               show TCoS ledger
`)
}

func loadContract(path string) (*client.Contract, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c client.Contract
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("parse contract %s: %w", path, err)
	}
	return &c, nil
}

func runValidate(ctx context.Context, c *client.Client, args []string) error {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	contractPath := fs.String("contract", "", "path to InferenceContract JSON")
	policyName := fs.String("policy", "", "RoutingPolicy name")
	_ = fs.Parse(args[1:])
	if *contractPath == "" {
		return fmt.Errorf("-contract is required")
	}
	ct, err := loadContract(*contractPath)
	if err != nil {
		return err
	}
	res, err := c.ValidateContract(ctx, ct, *policyName)
	if err != nil {
		return err
	}
	out, _ := json.MarshalIndent(res, "", "  ")
	fmt.Println(string(out))
	return nil
}

func runEligible(ctx context.Context, c *client.Client, args []string) error {
	fs := flag.NewFlagSet("eligible", flag.ExitOnError)
	contractPath := fs.String("contract", "", "path to InferenceContract JSON")
	policyName := fs.String("policy", "", "RoutingPolicy name")
	_ = fs.Parse(args[1:])
	if *contractPath == "" {
		return fmt.Errorf("-contract is required")
	}
	ct, err := loadContract(*contractPath)
	if err != nil {
		return err
	}
	cands, err := c.EligibleCandidates(ctx, ct, *policyName)
	if err != nil {
		return err
	}
	for _, cand := range cands {
		status := "eligible"
		if !cand.Eligible {
			status = "ineligible"
		}
		fmt.Printf("%-24s %-20s %s  score=%.3f %v\n", cand.ModelRef, cand.EndpointRef, status, cand.Score, cand.FilterReasons)
	}
	return nil
}

func runChat(ctx context.Context, c *client.Client, args []string) error {
	fs := flag.NewFlagSet("chat", flag.ExitOnError)
	contractPath := fs.String("contract", "", "path to InferenceContract JSON")
	policyName := fs.String("policy", "", "RoutingPolicy name")
	prompt := fs.String("prompt", "", "user message")
	_ = fs.Parse(args[1:])
	if *contractPath == "" {
		return fmt.Errorf("-contract is required")
	}
	ct, err := loadContract(*contractPath)
	if err != nil {
		return err
	}
	cc := c.WithContract(ct)
	cc.PolicyName = *policyName
	resp, err := cc.Chat(ctx, "auto", []client.Message{{Role: "user", Content: *prompt}})
	if err != nil {
		return err
	}
	fmt.Printf("model: %s  endpoint: %s  decision: %s\n", resp.Noetivela.ModelRef, resp.Noetivela.EndpointRef, resp.Noetivela.DecisionID)
	fmt.Printf("content: %s\n", resp.Content)
	fmt.Printf("usage: %+v  tcos: %.6f\n", resp.Usage, resp.Noetivela.TCoS.Total)
	return nil
}

func runDecisions(ctx context.Context, c *client.Client, args []string) error {
	fs := flag.NewFlagSet("decisions", flag.ExitOnError)
	limit := fs.Int("limit", 50, "page size")
	offset := fs.Int("offset", 0, "page offset")
	task := fs.String("task", "", "filter by task")
	model := fs.String("model", "", "filter by selected model ref")
	_ = fs.Parse(args[1:])
	decisions, err := c.Decisions(ctx, client.DecisionsOptions{
		Limit: *limit, Offset: *offset, Task: *task, Model: *model,
	})
	if err != nil {
		return err
	}
	for _, d := range decisions {
		sel := "<none>"
		if len(d.Selection) > 0 {
			sel = string(d.Selection)
		}
		fmt.Printf("%s  task=%s policy=%s  selection=%s\n", d.ID, d.Task, d.Policy, sel)
	}
	return nil
}

func runTrain(ctx context.Context, c *client.Client, args []string) error {
	fs := flag.NewFlagSet("train", flag.ExitOnError)
	version := fs.String("version", "", "new router model version, e.g. v2-table")
	minSamples := fs.Int("min-samples", 10, "coverage before predictions are trusted")
	tcosWeight := fs.Float64("tcos-weight", 0.5, "quality vs cost balance in learned score")
	holdOut := fs.Float64("hold-out", 0.0, "fraction reserved for validation")
	_ = fs.Parse(args[1:])
	if *version == "" {
		return fmt.Errorf("-version is required")
	}
	res, err := c.Train(ctx, client.TrainOptions{
		VersionID: *version, MinSamples: *minSamples, TCoSWeight: *tcosWeight, HoldOutFrac: *holdOut,
	})
	if err != nil {
		return err
	}
	fmt.Printf("trained router %s on %d samples (min_confidence=%.3f)", res.VersionID, res.TrainedSamples, res.MinConfidence)
	if res.ValidateMAE != 0 {
		fmt.Printf(" validate_mae=%.4f", res.ValidateMAE)
	}
	fmt.Println()
	return nil
}

func runUsage(ctx context.Context, c *client.Client) error {
	usage, err := c.Usage(ctx)
	if err != nil {
		return err
	}
	for _, u := range usage {
		fmt.Println(string(u))
	}
	return nil
}

func runReplay(ctx context.Context, c *client.Client, args []string) error {
	fs := flag.NewFlagSet("replay", flag.ExitOnError)
	baseline := fs.String("baseline", "", "fixed baseline endpoint ref")
	policyName := fs.String("policy", "", "RoutingPolicy name")
	limit := fs.Int("limit", 0, "max recorded requests to replay (0 = all)")
	_ = fs.Parse(args[1:])
	if *baseline == "" {
		return fmt.Errorf("-baseline is required")
	}
	res, err := c.Replay(ctx, *baseline, *policyName, *limit)
	if err != nil {
		return err
	}
	fmt.Printf("records: %d  savings: %.2f%%  verdict: %s\n", res.Records, res.SavingsPercent, res.Verdict)
	fmt.Printf("  baseline: quality-adjusted TCoS=%.4f served=%d denied=%d\n",
		res.Baseline.QualityAdjTCoS, res.Baseline.ServedCount, res.Baseline.DeniedCount)
	fmt.Printf("  routed:   quality-adjusted TCoS=%.4f served=%d denied=%d\n",
		res.Routed.QualityAdjTCoS, res.Routed.ServedCount, res.Routed.DeniedCount)
	return nil
}
