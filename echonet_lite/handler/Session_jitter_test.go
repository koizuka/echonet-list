package handler

import (
	"testing"
	"time"
)

func TestSession_calculateRetryIntervalWithJitter(t *testing.T) {
	session := &Session{
		RetryInterval: 3 * time.Second,
	}

	// ジッタが適用されることを確認するために複数回実行
	intervals := make([]time.Duration, 10)
	for i := 0; i < 10; i++ {
		intervals[i] = session.calculateRetryIntervalWithJitter()
	}

	// 基準値の範囲を確認（±30%のジッタ）
	baseInterval := session.RetryInterval
	minExpected := time.Duration(float64(baseInterval) * 0.7) // -30%
	maxExpected := time.Duration(float64(baseInterval) * 1.3) // +30%
	minAllowed := baseInterval / 2                            // 最小値は基準値の50%

	// すべての値が期待される範囲内にあることを確認
	for i, interval := range intervals {
		if interval < minAllowed {
			t.Errorf("Interval %d (%v) is less than minimum allowed (%v)", i, interval, minAllowed)
		}
		if interval < minExpected || interval > maxExpected {
			t.Errorf("Interval %d (%v) is outside expected range [%v, %v]", i, interval, minExpected, maxExpected)
		}
	}

	// すべての値が同じでないことを確認（ジッタが機能していることの確認）
	allSame := true
	for i := 1; i < len(intervals); i++ {
		if intervals[i] != intervals[0] {
			allSame = false
			break
		}
	}
	if allSame {
		t.Error("All intervals are the same, jitter is not working")
	}

	// 値の分散を確認
	var sum time.Duration
	for _, interval := range intervals {
		sum += interval
	}
	average := sum / time.Duration(len(intervals))

	// 平均値が基準値に近いことを確認（±10%以内）
	if average < time.Duration(float64(baseInterval)*0.9) || average > time.Duration(float64(baseInterval)*1.1) {
		t.Errorf("Average interval (%v) deviates too much from base interval (%v)", average, baseInterval)
	}
}

func TestSession_calculateRetryIntervalWithJitter_MinimumValue(t *testing.T) {
	session := &Session{
		RetryInterval: 100 * time.Millisecond, // 非常に小さい基準値
	}

	// 最小値の保証を確認
	for i := 0; i < 100; i++ {
		interval := session.calculateRetryIntervalWithJitter()
		minAllowed := session.RetryInterval / 2
		if interval < minAllowed {
			t.Errorf("Interval (%v) is less than minimum allowed (%v)", interval, minAllowed)
		}
	}
}
