package policy

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type tokenKind int

const (
	tokIdent tokenKind = iota
	tokNumber
	tokString
	tokPunct
	tokEOF
)

type token struct {
	kind tokenKind
	text string
	pos  int
}

type lexer struct {
	src string
	pos int
}

func (l *lexer) next() (token, error) {
	for l.pos < len(l.src) {
		c := l.src[l.pos]
		if unicode.IsSpace(rune(c)) {
			l.pos++
			continue
		}
		if c == '#' {
			// line comment
			for l.pos < len(l.src) && l.src[l.pos] != '\n' {
				l.pos++
			}
			continue
		}
		break
	}
	if l.pos >= len(l.src) {
		return token{kind: tokEOF, pos: l.pos}, nil
	}
	start := l.pos
	c := l.src[l.pos]
	switch {
	case c == '"' || c == '\'':
		quote := c
		l.pos++
		var sb strings.Builder
		for l.pos < len(l.src) && l.src[l.pos] != quote {
			sb.WriteByte(l.src[l.pos])
			l.pos++
		}
		if l.pos >= len(l.src) {
			return token{}, fmt.Errorf("unterminated string at %d", start)
		}
		l.pos++ // closing quote
		return token{kind: tokString, text: sb.String(), pos: start}, nil
	case unicode.IsDigit(rune(c)) || (c == '.' && l.pos+1 < len(l.src) && unicode.IsDigit(rune(l.src[l.pos+1]))):
		for l.pos < len(l.src) && (unicode.IsDigit(rune(l.src[l.pos])) || l.src[l.pos] == '.') {
			l.pos++
		}
		return token{kind: tokNumber, text: l.src[start:l.pos], pos: start}, nil
	case c == '_' || unicode.IsLetter(rune(c)):
		for l.pos < len(l.src) && (unicode.IsLetter(rune(l.src[l.pos])) || unicode.IsDigit(rune(l.src[l.pos])) || l.src[l.pos] == '_' || l.src[l.pos] == '-') {
			l.pos++
		}
		return token{kind: tokIdent, text: l.src[start:l.pos], pos: start}, nil
	default:
		if l.pos+1 < len(l.src) {
			two := l.src[l.pos : l.pos+2]
			switch two {
			case "==", "!=", ">=", "<=":
				l.pos += 2
				return token{kind: tokPunct, text: two, pos: start}, nil
			}
		}
		l.pos++
		return token{kind: tokPunct, text: string(c), pos: start}, nil
	}
}

type parser struct {
	toks []token
	pos  int
}

func Parse(input string) (*Policy, error) {
	var l lexer
	l.src = input
	var toks []token
	for {
		t, err := l.next()
		if err != nil {
			return nil, err
		}
		toks = append(toks, t)
		if t.kind == tokEOF {
			break
		}
	}
	p := &parser{toks: toks}
	return p.parsePolicy()
}

func (p *parser) peek() token { return p.toks[p.pos] }

func (p *parser) take() token {
	t := p.toks[p.pos]
	p.pos++
	return t
}

func (p *parser) expectPunct(s string) error {
	t := p.take()
	if t.kind != tokPunct || t.text != s {
		return fmt.Errorf("expected %q at %d, got %q", s, t.pos, t.text)
	}
	return nil
}

func (p *parser) expectIdent() (token, error) {
	t := p.take()
	if t.kind != tokIdent {
		return token{}, fmt.Errorf("expected identifier at %d, got %q", t.pos, t.text)
	}
	return t, nil
}

func (p *parser) parsePolicy() (*Policy, error) {
	if _, err := p.expectIdent(); err != nil {
		return nil, fmt.Errorf("expected 'policy': %w", err)
	}
	nameTok, err := p.expectIdent()
	if err != nil {
		return nil, fmt.Errorf("expected policy name: %w", err)
	}
	if err := p.expectPunct("{"); err != nil {
		return nil, err
	}

	pol := &Policy{Name: nameTok.text}
	for p.peek().kind != tokEOF && p.peek().text != "}" {
		kw, err := p.expectIdent()
		if err != nil {
			return nil, err
		}
		switch kw.text {
		case "require":
			expr, err := p.parseOr()
			if err != nil {
				return nil, err
			}
			switch expr.(type) {
			case *CompareExpr, *InExpr:
			default:
				return nil, fmt.Errorf("require must be a comparison or 'in' expression")
			}
			pol.Requires = append(pol.Requires, Require{Expr: expr})
		case "prefer":
			feat, err := p.expectIdent()
			if err != nil {
				return nil, err
			}
			if _, err := p.expectIdent(); err != nil { // 'weight'
				return nil, err
			}
			w, err := p.parseNumber()
			if err != nil {
				return nil, err
			}
			pol.Prefers = append(pol.Prefers, Prefer{Feature: feat.text, Weight: w})
		case "optimize":
			opt := &Optimize{Weights: map[Dim]float64{}}
			first, _ := p.expectIdent()
			if first.text == "lexicographic" {
				opt.Lexicographic = true
				for p.peek().kind != tokEOF && p.peek().text != "}" && p.peek().text != "fallback" && p.peek().text != "switch" && p.peek().text != "require" && p.peek().text != "prefer" {
					dim, _ := p.expectIdent()
					opt.Order = append(opt.Order, Dim(dim.text))
				}
			} else {
				if err := p.parseDimWeight(opt, first); err != nil {
					return nil, err
				}
				for isDim(p.peek().text) {
					dim, _ := p.expectIdent()
					if err := p.parseDimWeight(opt, dim); err != nil {
						return nil, err
					}
				}
			}
			pol.Optimize = opt
		case "fallback":
			cls, err := p.expectIdent()
			if err != nil {
				return nil, err
			}
			fb := &Fallback{Class: FallbackClass(cls.text)}
			if p.peek().text == "max_attempts" {
				_ = p.take()
				n, err := p.parseInteger()
				if err != nil {
					return nil, err
				}
				fb.MaxAttempts = n
			}
			pol.Fallback = fb
		case "switch":
			if _, err := p.expectIdent(); err != nil { // 'only_if'
				return nil, err
			}
			if _, err := p.expectIdent(); err != nil { // 'expected_gain'
				return nil, err
			}
			if err := p.expectPunct(">="); err != nil {
				return nil, err
			}
			gain, err := p.parseNumber()
			if err != nil {
				return nil, err
			}
			sw := &Switch{MinGain: gain}
			if p.peek().text == "min_hold_seconds" {
				_ = p.take()
				secs, err := p.parseInteger()
				if err != nil {
					return nil, err
				}
				sw.MinHoldSeconds = secs
			}
			pol.Switch = sw
		default:
			return nil, fmt.Errorf("unknown clause %q at %d", kw.text, kw.pos)
		}
	}
	if err := p.expectPunct("}"); err != nil {
		return nil, err
	}
	return pol, nil
}

func isDim(s string) bool {
	switch Dim(s) {
	case DimQuality, DimLatency, DimTCoS, DimReliability, DimEnergy, DimCapacity:
		return true
	}
	return false
}

func (p *parser) parseDimWeight(opt *Optimize, dim token) error {
	w, err := p.parseNumber()
	if err != nil {
		return err
	}
	opt.Weights[Dim(dim.text)] = w
	return nil
}

func (p *parser) parseNumber() (float64, error) {
	t := p.take()
	if t.kind != tokNumber {
		return 0, fmt.Errorf("expected number at %d, got %q", t.pos, t.text)
	}
	return strconv.ParseFloat(t.text, 64)
}

func (p *parser) parseInteger() (int, error) {
	t := p.take()
	if t.kind != tokNumber {
		return 0, fmt.Errorf("expected integer at %d, got %q", t.pos, t.text)
	}
	n, err := strconv.ParseFloat(t.text, 64)
	return int(n), err
}

func (p *parser) parseOr() (Expr, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.peek().text == "or" {
		_ = p.take()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &OrExpr{Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseAnd() (Expr, error) {
	left, err := p.parseCmp()
	if err != nil {
		return nil, err
	}
	for p.peek().text == "and" {
		_ = p.take()
		right, err := p.parseCmp()
		if err != nil {
			return nil, err
		}
		left = &AndExpr{Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseCmp() (Expr, error) {
	left, err := p.parseOperand()
	if err != nil {
		return nil, err
	}
	t := p.peek()
	if t.kind == tokIdent {
		switch t.text {
		case "in":
			_ = p.take()
			if err := p.expectPunct("["); err != nil {
				return nil, err
			}
			var items []string
			for p.peek().text != "]" {
				lit, err := p.parseOperand()
				if err != nil {
					return nil, err
				}
				litExpr, ok := lit.(*LiteralExpr)
				if !ok {
					return nil, fmt.Errorf("'in' list items must be literals")
				}
				items = append(items, litExpr.Value)
				if p.peek().text == "," {
					_ = p.take()
				}
			}
			_ = p.take() // ']'
			return &InExpr{Operand: left, Items: items}, nil
		}
	}
	if t.kind == tokPunct {
		switch t.text {
		case "==", "!=", ">=", "<=", ">", "<":
			_ = p.take()
			right, err := p.parseOperand()
			if err != nil {
				return nil, err
			}
			return &CompareExpr{Left: left, Op: BinOp(t.text), Right: right}, nil
		}
	}
	return left, nil
}

func (p *parser) parseOperand() (Expr, error) {
	t := p.take()
	switch t.kind {
	case tokString:
		return &LiteralExpr{Kind: "string", Value: t.text}, nil
	case tokNumber:
		return &LiteralExpr{Kind: "number", Value: t.text}, nil
	case tokIdent:
		switch t.text {
		case "true", "false":
			return &LiteralExpr{Kind: "boolean", Value: t.text}, nil
		case "not":
			inner, err := p.parseOperand()
			if err != nil {
				return nil, err
			}
			return &NotExpr{Inner: inner}, nil
		default:
			// dotted path: ns.seg(.seg)* or ns.func(...)
			ns := t.text
			next := p.peek()
			if next.text != "." {
				return nil, fmt.Errorf("expected '.' after %q at %d", ns, next.pos)
			}
			_ = p.take() // '.'
			name, err := p.expectIdent()
			if err != nil {
				return nil, err
			}
			if p.peek().text == "(" {
				_ = p.take()
				var args []string
				for p.peek().text != ")" {
					lit, err := p.parseOperand()
					if err != nil {
						return nil, err
					}
					if le, ok := lit.(*LiteralExpr); ok {
						args = append(args, le.Value)
					}
					if p.peek().text == "," {
						_ = p.take()
					}
				}
				_ = p.take() // ')'
				return &CallExpr{Namespace: ns, Name: name.text, Args: args}, nil
			}
			segs := []string{name.text}
			for p.peek().text == "." {
				_ = p.take()
				s, err := p.expectIdent()
				if err != nil {
					return nil, err
				}
				segs = append(segs, s.text)
			}
			return &PathExpr{Namespace: ns, Segments: segs}, nil
		}
	default:
		return nil, fmt.Errorf("unexpected token %q at %d", t.text, t.pos)
	}
}
