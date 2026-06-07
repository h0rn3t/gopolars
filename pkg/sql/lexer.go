package sql

import (
	"fmt"
	"strings"
)

// tokenKind enumerates the lexical token categories produced by the lexer.
type tokenKind int

const (
	tokEOF    tokenKind = iota
	tokIdent            // bare or double-quoted identifier (case preserved)
	tokNumber           // numeric literal
	tokString           // single-quoted string literal (unquoted content)
	tokOp               // operator: = <> != < <= > >= + - * / % :: (text holds the operator)
	tokLParen           // (
	tokRParen           // )
	tokComma            // ,
	tokDot              // .
)

// token is a single lexical unit. text holds the literal slice (operator text,
// identifier name, raw number, or unquoted string content). quoted marks a
// double-quoted identifier so keyword matching can be skipped for it.
type token struct {
	kind   tokenKind
	text   string
	quoted bool
}

// lex tokenizes a SQL expression fragment. It understands SQL string literals
// (single quotes, ” escape), double-quoted identifiers, multi-character
// operators (<= >= <> != ::), and the punctuation used by the expression
// grammar. It does not interpret keywords — the parser matches identifier text
// case-insensitively against keywords where it needs to.
func lex(input string) ([]token, error) {
	tokens := make([]token, 0, len(input)/2+1)
	runes := []rune(input)
	i := 0
	n := len(runes)
	for i < n {
		ch := runes[i]
		switch {
		case ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r':
			i++
		case ch == '(':
			tokens = append(tokens, token{kind: tokLParen, text: "("})
			i++
		case ch == ')':
			tokens = append(tokens, token{kind: tokRParen, text: ")"})
			i++
		case ch == ',':
			tokens = append(tokens, token{kind: tokComma, text: ","})
			i++
		case ch == '\'':
			s, next, err := lexString(runes, i)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, token{kind: tokString, text: s})
			i = next
		case ch == '"':
			s, next, err := lexQuotedIdent(runes, i)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, token{kind: tokIdent, text: s, quoted: true})
			i = next
		case isIdentStart(ch):
			j := i + 1
			for j < n && isIdentPart(runes[j]) {
				j++
			}
			tokens = append(tokens, token{kind: tokIdent, text: string(runes[i:j])})
			i = j
		case ch >= '0' && ch <= '9':
			j := i + 1
			for j < n && (runes[j] >= '0' && runes[j] <= '9') {
				j++
			}
			if j < n && runes[j] == '.' {
				j++
				for j < n && (runes[j] >= '0' && runes[j] <= '9') {
					j++
				}
			}
			// optional exponent
			if j < n && (runes[j] == 'e' || runes[j] == 'E') {
				k := j + 1
				if k < n && (runes[k] == '+' || runes[k] == '-') {
					k++
				}
				if k < n && runes[k] >= '0' && runes[k] <= '9' {
					j = k
					for j < n && (runes[j] >= '0' && runes[j] <= '9') {
						j++
					}
				}
			}
			tokens = append(tokens, token{kind: tokNumber, text: string(runes[i:j])})
			i = j
		case ch == '.':
			// could be a number like .5; otherwise it is a dot separator
			if i+1 < n && runes[i+1] >= '0' && runes[i+1] <= '9' {
				j := i + 1
				for j < n && (runes[j] >= '0' && runes[j] <= '9') {
					j++
				}
				tokens = append(tokens, token{kind: tokNumber, text: string(runes[i:j])})
				i = j
			} else {
				tokens = append(tokens, token{kind: tokDot, text: "."})
				i++
			}
		case ch == '<':
			if i+1 < n && runes[i+1] == '=' {
				tokens = append(tokens, token{kind: tokOp, text: "<="})
				i += 2
			} else if i+1 < n && runes[i+1] == '>' {
				tokens = append(tokens, token{kind: tokOp, text: "!="})
				i += 2
			} else {
				tokens = append(tokens, token{kind: tokOp, text: "<"})
				i++
			}
		case ch == '>':
			if i+1 < n && runes[i+1] == '=' {
				tokens = append(tokens, token{kind: tokOp, text: ">="})
				i += 2
			} else {
				tokens = append(tokens, token{kind: tokOp, text: ">"})
				i++
			}
		case ch == '=':
			tokens = append(tokens, token{kind: tokOp, text: "="})
			i++
		case ch == '!':
			if i+1 < n && runes[i+1] == '=' {
				tokens = append(tokens, token{kind: tokOp, text: "!="})
				i += 2
			} else {
				return nil, fmt.Errorf("unexpected character '!' in SQL expression")
			}
		case ch == ':':
			if i+1 < n && runes[i+1] == ':' {
				tokens = append(tokens, token{kind: tokOp, text: "::"})
				i += 2
			} else {
				return nil, fmt.Errorf("unexpected character ':' in SQL expression")
			}
		case ch == '+' || ch == '-' || ch == '*' || ch == '/' || ch == '%':
			tokens = append(tokens, token{kind: tokOp, text: string(ch)})
			i++
		default:
			return nil, fmt.Errorf("unexpected character %q in SQL expression", string(ch))
		}
	}
	tokens = append(tokens, token{kind: tokEOF})
	return tokens, nil
}

func lexString(runes []rune, start int) (string, int, error) {
	var b strings.Builder
	i := start + 1
	n := len(runes)
	for i < n {
		ch := runes[i]
		if ch == '\'' {
			// '' is an escaped single quote
			if i+1 < n && runes[i+1] == '\'' {
				b.WriteRune('\'')
				i += 2
				continue
			}
			return b.String(), i + 1, nil
		}
		b.WriteRune(ch)
		i++
	}
	return "", 0, fmt.Errorf("unterminated string literal")
}

func lexQuotedIdent(runes []rune, start int) (string, int, error) {
	var b strings.Builder
	i := start + 1
	n := len(runes)
	for i < n {
		ch := runes[i]
		if ch == '"' {
			if i+1 < n && runes[i+1] == '"' {
				b.WriteRune('"')
				i += 2
				continue
			}
			return b.String(), i + 1, nil
		}
		b.WriteRune(ch)
		i++
	}
	return "", 0, fmt.Errorf("unterminated quoted identifier")
}

func isIdentStart(ch rune) bool {
	return ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isIdentPart(ch rune) bool {
	return isIdentStart(ch) || (ch >= '0' && ch <= '9')
}
