package util

import "testing"

func BenchmarkUuid(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewId()
	}
}
