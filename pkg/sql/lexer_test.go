package sql

import "testing"

func TestLexTokenKinds(t *testing.T) {
	toks, err := lex("a.b >= 'it''s' + 3.5 :: INT, (x)")
	if err != nil {
		t.Fatalf("lex: %v", err)
	}
	want := []struct {
		kind tokenKind
		text string
	}{
		{tokIdent, "a"},
		{tokDot, "."},
		{tokIdent, "b"},
		{tokOp, ">="},
		{tokString, "it's"},
		{tokOp, "+"},
		{tokNumber, "3.5"},
		{tokOp, "::"},
		{tokIdent, "INT"},
		{tokComma, ","},
		{tokLParen, "("},
		{tokIdent, "x"},
		{tokRParen, ")"},
		{tokEOF, ""},
	}
	if len(toks) != len(want) {
		t.Fatalf("token count = %d, want %d (%v)", len(toks), len(want), toks)
	}
	for i, w := range want {
		if toks[i].kind != w.kind || toks[i].text != w.text {
			t.Fatalf("token[%d] = {%v %q}, want {%v %q}", i, toks[i].kind, toks[i].text, w.kind, w.text)
		}
	}
}

func TestLexQuotedIdentifier(t *testing.T) {
	toks, err := lex(`"Mixed Case"`)
	if err != nil {
		t.Fatalf("lex: %v", err)
	}
	if toks[0].kind != tokIdent || toks[0].text != "Mixed Case" || !toks[0].quoted {
		t.Fatalf("quoted ident = {%v %q quoted=%v}", toks[0].kind, toks[0].text, toks[0].quoted)
	}
}

func TestLexErrors(t *testing.T) {
	for _, src := range []string{"'unterminated", `"unterminated`, "a ! b", "a : b"} {
		if _, err := lex(src); err == nil {
			t.Fatalf("lex(%q) expected error", src)
		}
	}
}
