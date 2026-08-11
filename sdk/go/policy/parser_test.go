package policy

import (
	"math"
	"reflect"
	"testing"
)

const sample = `
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
`

func TestParseSamplePolicy(t *testing.T) {
	p, err := Parse(sample)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "legal-confidential-v2" {
		t.Fatalf("unexpected name %q", p.Name)
	}
	if len(p.Requires) != 4 {
		t.Fatalf("expected 4 requires, got %d", len(p.Requires))
	}

	first, ok := p.Requires[0].Expr.(*InExpr)
	if !ok {
		t.Fatalf("expected InExpr, got %T", p.Requires[0].Expr)
	}
	path, ok := first.Operand.(*PathExpr)
	if !ok || path.Namespace != "data" || path.Segments[0] != "region" {
		t.Fatalf("unexpected first require operand: %+v", first.Operand)
	}
	if !reflect.DeepEqual(first.Items, []string{"sg", "cn-north"}) {
		t.Fatalf("unexpected items %v", first.Items)
	}

	call := p.Requires[1].Expr.(*CompareExpr).Left.(*CallExpr)
	if call.Namespace != "model" || call.Name != "eval" || call.Args[0] != "legal_clause_v3" {
		t.Fatalf("unexpected call expr: %+v", call)
	}
	if p.Requires[1].Expr.(*CompareExpr).Op != OpGe {
		t.Fatalf("expected >= op, got %s", p.Requires[1].Expr.(*CompareExpr).Op)
	}

	if len(p.Prefers) != 2 || p.Prefers[0].Feature != "session_affinity" || p.Prefers[0].Weight != 0.25 {
		t.Fatalf("unexpected prefers: %+v", p.Prefers)
	}

	opt := p.Optimize
	if math.Abs(opt.Weights[DimQuality]-0.45) > 1e-9 || math.Abs(opt.Weights[DimLatency]-0.20) > 1e-9 {
		t.Fatalf("unexpected optimize weights: %+v", opt.Weights)
	}
	if opt.Lexicographic {
		t.Fatal("expected weighted optimize, not lexicographic")
	}

	if p.Fallback == nil || p.Fallback.Class != FallbackSameOrHigher || p.Fallback.MaxAttempts != 2 {
		t.Fatalf("unexpected fallback: %+v", p.Fallback)
	}
	if p.Switch == nil || math.Abs(p.Switch.MinGain-0.08) > 1e-9 || p.Switch.MinHoldSeconds != 300 {
		t.Fatalf("unexpected switch: %+v", p.Switch)
	}
}

func TestLexicographicOptimize(t *testing.T) {
	p, err := Parse(`policy p { optimize lexicographic quality safety latency tcos }`)
	if err != nil {
		t.Fatal(err)
	}
	if !p.Optimize.Lexicographic {
		t.Fatal("expected lexicographic")
	}
	want := []Dim{DimQuality, "safety", DimLatency, DimTCoS}
	if !reflect.DeepEqual(p.Optimize.Order, want) {
		t.Fatalf("unexpected order %v", p.Optimize.Order)
	}
}

func TestCommentsAndWhitespace(t *testing.T) {
	p, err := Parse(`# Example routing policy
policy demo { # inline too
  require data.region in ["sg"] # trailing
  optimize quality 0.5 tcos 0.5
}`)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "demo" || len(p.Requires) != 1 {
		t.Fatalf("unexpected parse %+v", p)
	}
}

func TestParseErrors(t *testing.T) {
	cases := []string{
		`policy { }`,                         // missing name
		`policy p { require data.region }`,   // dangling operand
		`policy p { prefer x 0.5 }`,          // missing 'weight'
		`policy p { bogus data.region in ["x"] }`, // unknown clause
	}
	for _, c := range cases {
		if _, err := Parse(c); err == nil {
			t.Errorf("expected parse error for %q", c)
		}
	}
}
