// Package simd provides thin wrappers over turboslice SIMD operations
// with graceful fallback to scalar implementations on non-AMD64
// architectures or when built without GOEXPERIMENT=simd.
//
// Build with SIMD acceleration:
//
//	GOEXPERIMENT=simd go build ./...
//
// Build without (auto-fallback):
//
//	go build ./...
package simd
