package tests

import (
	"testing"

	"ad-targeting-engine/internal/engine"
)

func BenchmarkMatch(b *testing.B) {
	eng := engine.NewEngine()
	// Skipping snapshot build for brevity; in real test, set snapshot directly
	req := engine.MatchRequest{AppID: "com.app", Country: "IN", OS: "android"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = eng.Match(nil, req)
	}
}
