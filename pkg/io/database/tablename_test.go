package database

import "testing"

func TestParseTableName(t *testing.T) {
	cases := []struct {
		name               string
		in                 string
		cat, schema, table string
		wantErr            bool
	}{
		{name: "bare", in: "tbl", table: "tbl"},
		{name: "schema.table", in: "sch.tbl", schema: "sch", table: "tbl"},
		{name: "catalog.schema.table", in: "cat.sch.tbl", cat: "cat", schema: "sch", table: "tbl"},
		{name: "quoted dot in name", in: `"my.tbl"`, table: "my.tbl"},
		{name: "quoted schema and table", in: `"sch"."my.tbl"`, schema: "sch", table: "my.tbl"},
		{name: "backtick quoting", in: "`sch`.`tbl`", schema: "sch", table: "tbl"},
		{name: "escaped doubled quote", in: `"a""b"`, table: `a"b`},
		{name: "whitespace trimmed (unquoted)", in: "  sch  .  tbl  ", schema: "sch", table: "tbl"},
		{name: "whitespace around quotes ignored", in: ` "sch" . "tbl" `, schema: "sch", table: "tbl"},
		{name: "too many parts", in: "a.b.c.d", wantErr: true},
		{name: "empty table", in: "sch.", wantErr: true},
		{name: "unterminated quote", in: `"tbl`, wantErr: true},
		{name: "text after closing quote", in: `"sch"x.tbl`, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cat, schema, table, err := parseTableName(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseTableName(%q): expected error, got cat=%q schema=%q table=%q", tc.in, cat, schema, table)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseTableName(%q): unexpected error: %v", tc.in, err)
			}
			if cat != tc.cat || schema != tc.schema || table != tc.table {
				t.Fatalf("parseTableName(%q) = (%q, %q, %q), want (%q, %q, %q)",
					tc.in, cat, schema, table, tc.cat, tc.schema, tc.table)
			}
		})
	}
}

// TestQuoteIdentifier checks the quoting and that splitIdentifier reads every
// quoted name back as one part equal to the input.
func TestQuoteIdentifier(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{name: "tbl", want: `"tbl"`},
		{name: "my tbl", want: `"my tbl"`},
		{name: `my"tbl`, want: `"my""tbl"`},
		{name: "select", want: `"select"`},
		{name: "my.tbl", want: `"my.tbl"`},
		{name: " padded ", want: `" padded "`},
		{name: "", want: `""`},
		{name: `""`, want: `""""""`},
		{name: "back`tick", want: "\"back`tick\""},
		{name: "таблиця", want: `"таблиця"`},
		{name: `a" AS SELECT 'injected' AS secret --`, want: `"a"" AS SELECT 'injected' AS secret --"`},
		{name: `x"; COPY (SELECT 1) TO 'out.csv'; --`, want: `"x""; COPY (SELECT 1) TO 'out.csv'; --"`},
	}
	for _, tc := range cases {
		got := QuoteIdentifier(tc.name)
		if got != tc.want {
			t.Errorf("QuoteIdentifier(%q) = %q, want %q", tc.name, got, tc.want)
		}
		parts, err := splitIdentifier(got)
		if err != nil || len(parts) != 1 || parts[0] != tc.name {
			t.Errorf("splitIdentifier(%q) = %q, %v, want [%q], nil", got, parts, err, tc.name)
		}
	}
}

func TestIngestMode(t *testing.T) {
	cases := []struct {
		in      IfTableExists
		want    string
		wantErr bool
	}{
		{in: "", want: "adbc.ingest.mode.create"},
		{in: IfTableExistsFail, want: "adbc.ingest.mode.create"},
		{in: IfTableExistsAppend, want: "adbc.ingest.mode.create_append"},
		{in: IfTableExistsReplace, want: "adbc.ingest.mode.replace"},
		{in: "bogus", wantErr: true},
	}
	for _, tc := range cases {
		got, err := ingestMode(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ingestMode(%q): expected error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("ingestMode(%q): unexpected error: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ingestMode(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
