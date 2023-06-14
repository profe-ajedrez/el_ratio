package el_ratio

import (
	"testing"
	"time"
)

func TestRatio(t *testing.T) {
	l := NewLeakyBucketLimiter(1, time.Second)

	prev := time.Now()

	for i := 0; i <= 9; i++ {
		now := l.Wait()

		if i > 0 {
			ellapsed := now.Sub(prev).Round(time.Millisecond * 2)
			t.Logf("round: %d  delay: %s ", i, ellapsed)

			if ellapsed != time.Second {
				t.Logf("expected 1 second of delay, got %s", ellapsed)
				t.FailNow()
			}
		}

		prev = now
	}
}

func BenchmarkRatio(b *testing.B) {
	l := NewLeakyBucketLimiter(1, time.Second)

	prev := time.Now()

	b.StopTimer()
	for k := 0; k < b.N; k++ {
		for i := 0; i <= 4; i++ {
			b.StartTimer()
			now := l.Wait()
			b.StopTimer()

			if i > 0 {
				_ = now.Sub(prev).Round(time.Millisecond * 2)
			}

			prev = now
		}
	}
}
