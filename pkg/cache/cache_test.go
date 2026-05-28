package cache

import "testing"

func TestChunkCacheSetGet(t *testing.T) {
	c := NewChunkCache()
	c.Set("chunk-a", []byte{1, 2, 3})
	got, ok := c.Get("chunk-a")
	if !ok {
		t.Fatal("очікували наявність ключа chunk-a")
	}
	if len(got) != 3 || got[0] != 1 {
		t.Fatalf("неочікуване значення: %v", got)
	}
	if _, ok := c.Get("missing"); ok {
		t.Fatal("відсутній ключ не повинен знаходитися")
	}
}

func TestPlanCacheSetGet(t *testing.T) {
	c := NewPlanCache()
	c.Set("plan-1", "select * from t")
	got, ok := c.Get("plan-1")
	if !ok || got != "select * from t" {
		t.Fatalf("неочікуване значення плану: %q ok=%v", got, ok)
	}
}
