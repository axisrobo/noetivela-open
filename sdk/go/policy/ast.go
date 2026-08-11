// Package policy implements the NOETIVELA Routing Policy DSL.
//
// This package is Apache-2.0 licensed and lives in NOETIVELA-open: the DSL
// surface is open integration surface. The enforcement engine that interprets
// these policies inside the routing pipeline lives in NOETIVELA core (AGPL).
// Core depends on this package as the authoritative grammar implementation.
package policy

// AST nodes for the Routing Policy DSL (see contracts/dsl/grammar.ebnf).

type Dim string

const (
	DimQuality     Dim = "quality"
	DimLatency     Dim = "latency"
	DimTCoS        Dim = "tcos"
	DimReliability Dim = "reliability"
	DimEnergy      Dim = "energy"
	DimCapacity    Dim = "capacity"
)

type FallbackClass string

const (
	FallbackNone              FallbackClass = "none"
	FallbackSameOrHigher      FallbackClass = "same_or_higher_quality"
	FallbackAnyCompliant      FallbackClass = "any_compliant"
	FallbackDegradeAllowed    FallbackClass = "degrade_allowed"
)

type BinOp string

const (
	OpEq  BinOp = "=="
	OpNeq BinOp = "!="
	OpGe  BinOp = ">="
	OpLe  BinOp = "<="
	OpGt  BinOp = ">"
	OpLt  BinOp = "<"
	OpIn  BinOp = "in"
)

type Expr interface{ exprNode() }

type PathExpr struct {
	Namespace string   `json:"namespace"` // data | model | endpoint | contract
	Segments  []string `json:"segments"`
}

type CallExpr struct {
	Namespace string   `json:"namespace"` // model.eval(...) / model.capability(...)
	Name      string   `json:"name"`
	Args      []string `json:"args"`
}

type LiteralExpr struct {
	Kind  string `json:"kind"` // string | number | boolean
	Value string `json:"value"`
}

type InExpr struct {
	Operand Expr     `json:"operand"`
	Items   []string `json:"items"`
}

type CompareExpr struct {
	Left  Expr  `json:"left"`
	Op    BinOp `json:"op"`
	Right Expr  `json:"right"`
}

type NotExpr struct{ Inner Expr `json:"inner"` }

type AndExpr struct{ Left, Right Expr }

type OrExpr struct{ Left, Right Expr }

func (*PathExpr) exprNode()     {}
func (*CallExpr) exprNode()     {}
func (*LiteralExpr) exprNode()  {}
func (*InExpr) exprNode()       {}
func (*CompareExpr) exprNode()  {}
func (*NotExpr) exprNode()      {}
func (*AndExpr) exprNode()      {}
func (*OrExpr) exprNode()       {}

type Require struct {
	Expr Expr `json:"expr"`
}

type Prefer struct {
	Feature string  `json:"feature"`
	Weight  float64 `json:"weight"`
}

type Optimize struct {
	Lexicographic bool               `json:"lexicographic"`
	Weights       map[Dim]float64    `json:"weights"`
	Order         []Dim              `json:"order,omitempty"`
}

type Fallback struct {
	Class       FallbackClass `json:"class"`
	MaxAttempts int           `json:"max_attempts,omitempty"`
}

type Switch struct {
	MinGain       float64 `json:"min_expected_gain"`
	MinHoldSeconds int    `json:"min_hold_seconds,omitempty"`
}

type Policy struct {
	Name     string     `json:"name"`
	Requires []Require  `json:"requires"`
	Prefers  []Prefer   `json:"prefers"`
	Optimize *Optimize  `json:"optimize,omitempty"`
	Fallback *Fallback  `json:"fallback,omitempty"`
	Switch   *Switch    `json:"switch,omitempty"`
}
