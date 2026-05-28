package dtypes

import "testing"

func TestSchemaIndexOf(t *testing.T) {
	schema := Schema{
		{Name: "id", Type: Int64},
		{Name: "city", Type: String},
	}
	if got := schema.IndexOf("city"); got != 1 {
		t.Fatalf("IndexOf(city) = %d, очікували 1", got)
	}
	if got := schema.IndexOf("missing"); got != -1 {
		t.Fatalf("IndexOf(missing) = %d, очікували -1", got)
	}
}
