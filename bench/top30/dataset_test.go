package top30

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDatasetDeterministic(t *testing.T) {
	ds1 := GenerateDataset(1000, 42)
	ds2 := GenerateDataset(1000, 42)

	if len(ds1.G) != len(ds2.G) {
		t.Fatalf("g length mismatch")
	}
	for i := range ds1.G {
		if ds1.G[i] != ds2.G[i] {
			t.Fatalf("g mismatch at %d: %q vs %q", i, ds1.G[i], ds2.G[i])
		}
		if ds1.V[i] != ds2.V[i] {
			t.Fatalf("v mismatch at %d: %v vs %v", i, ds1.V[i], ds2.V[i])
		}
		if (ds1.N[i] == nil) != (ds2.N[i] == nil) {
			t.Fatalf("n nil mismatch at %d", i)
		}
		if ds1.N[i] != nil && ds2.N[i] != nil && *ds1.N[i] != *ds2.N[i] {
			t.Fatalf("n value mismatch at %d: %v vs %v", i, *ds1.N[i], *ds2.N[i])
		}
		if ds1.I[i] != ds2.I[i] {
			t.Fatalf("i mismatch at %d: %v vs %v", i, ds1.I[i], ds2.I[i])
		}
	}
}

func TestDatasetArrowIPCBinaryStable(t *testing.T) {
	dir := t.TempDir()
	p1 := filepath.Join(dir, "a.arrow")
	p2 := filepath.Join(dir, "b.arrow")

	ds := GenerateDataset(1000, 42)
	if err := WriteArrowIPC(p1, ds); err != nil {
		t.Fatalf("write first arrow ipc: %v", err)
	}
	if err := WriteArrowIPC(p2, ds); err != nil {
		t.Fatalf("write second arrow ipc: %v", err)
	}

	b1, err := os.ReadFile(p1)
	if err != nil {
		t.Fatalf("read first file: %v", err)
	}
	b2, err := os.ReadFile(p2)
	if err != nil {
		t.Fatalf("read second file: %v", err)
	}
	if len(b1) != len(b2) {
		t.Fatalf("file size mismatch: %d vs %d", len(b1), len(b2))
	}
	for i := range b1 {
		if b1[i] != b2[i] {
			t.Fatalf("byte mismatch at offset %d", i)
		}
	}
}

func TestDatasetArrowIPCMixedTypes(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "data.arrow")

	ds := GenerateDataset(100, 7)
	if err := WriteArrowIPC(p, ds); err != nil {
		t.Fatalf("write arrow ipc: %v", err)
	}

	info, err := os.Stat(p)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("arrow ipc file is empty")
	}
}
