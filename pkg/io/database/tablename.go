package database

import (
	"fmt"
	"strings"
)

// parseTableName splits a (possibly schema-qualified) table name into its
// catalog, schema and table parts. Parts are assigned from the right — matching
// Python polars' table-name unpacking — so "tbl" -> table, "sch.tbl" ->
// schema.table, "cat.sch.tbl" -> catalog.schema.table. Dots inside quoted
// identifiers ("my.tbl") are not separators. More than three parts is an error.
func parseTableName(name string) (catalog, schema, table string, err error) {
	parts, err := splitIdentifier(name)
	if err != nil {
		return "", "", "", err
	}
	switch len(parts) {
	case 1:
		table = parts[0]
	case 2:
		schema, table = parts[0], parts[1]
	case 3:
		catalog, schema, table = parts[0], parts[1], parts[2]
	default:
		return "", "", "", fmt.Errorf("table name %q has more than 3 dot-separated parts", name)
	}
	if strings.TrimSpace(table) == "" {
		return "", "", "", fmt.Errorf("empty table name in %q", name)
	}
	return catalog, schema, table, nil
}

// splitIdentifier splits a dotted identifier on unquoted '.' separators. Each
// part may be double-quote or backtick quoted; a doubled quote inside a quoted
// span is an escaped literal quote (`"a""b"` -> a"b). Unquoted parts are
// whitespace-trimmed; a part that mixes a quoted span with surrounding non-
// whitespace text, or that leaves a quote unterminated, is an error.
func splitIdentifier(name string) ([]string, error) {
	var parts []string
	var quoted, unquoted strings.Builder
	hadQuote := false
	var quote rune

	flush := func() error {
		var val string
		if hadQuote {
			if strings.TrimSpace(unquoted.String()) != "" {
				return fmt.Errorf("invalid identifier %q: unquoted text around a quoted part", name)
			}
			val = quoted.String()
		} else {
			val = strings.TrimSpace(unquoted.String())
		}
		parts = append(parts, val)
		quoted.Reset()
		unquoted.Reset()
		hadQuote = false
		return nil
	}

	runes := []rune(name)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		switch {
		case quote != 0:
			if r == quote {
				if i+1 < len(runes) && runes[i+1] == quote {
					quoted.WriteRune(quote) // escaped literal quote
					i++
				} else {
					quote = 0
				}
			} else {
				quoted.WriteRune(r)
			}
		case r == '"' || r == '`':
			quote = r
			hadQuote = true
		case r == '.':
			if err := flush(); err != nil {
				return nil, err
			}
		default:
			unquoted.WriteRune(r)
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quoted identifier in %q", name)
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return parts, nil
}
