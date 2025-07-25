package handler

import (
	"fmt"
	"sync"
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

	// 基準値の範囲を確認（定数を使用）
	baseInterval := session.RetryInterval
	minExpected := time.Duration(float64(baseInterval) * (1.0 - JitterPercentage)) // -30%
	maxExpected := time.Duration(float64(baseInterval) * (1.0 + JitterPercentage)) // +30%
	minAllowed := time.Duration(float64(baseInterval) * MinIntervalRatio)          // 最小値は基準値の50%

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
		minAllowed := time.Duration(float64(session.RetryInterval) * MinIntervalRatio)
		if interval < minAllowed {
			t.Errorf("Interval (%v) is less than minimum allowed (%v)", interval, minAllowed)
		}
	}
}

// TestSession_calculateRetryIntervalWithJitter_InvalidInput は無効な入力値に対するテスト
func TestSession_calculateRetryIntervalWithJitter_InvalidInput(t *testing.T) {
	tests := []struct {
		name          string
		retryInterval time.Duration
		expectDefault bool
	}{
		{
			name:          "Zero interval",
			retryInterval: 0,
			expectDefault: true,
		},
		{
			name:          "Negative interval",
			retryInterval: -1 * time.Second,
			expectDefault: true,
		},
		{
			name:          "Valid interval",
			retryInterval: 2 * time.Second,
			expectDefault: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{
				RetryInterval: tt.retryInterval,
			}

			interval := session.calculateRetryIntervalWithJitter()

			if tt.expectDefault {
				// デフォルト値（3秒）が返されることを確認
				if interval != 3*time.Second {
					t.Errorf("Expected default interval 3s, got %v", interval)
				}
			} else {
				// 有効な範囲内の値が返されることを確認
				minAllowed := time.Duration(float64(tt.retryInterval) * MinIntervalRatio)
				maxAllowed := time.Duration(float64(tt.retryInterval) * (1.0 + JitterPercentage))
				if interval < minAllowed || interval > maxAllowed {
					t.Errorf("Interval %v is outside expected range [%v, %v]", interval, minAllowed, maxAllowed)
				}
			}
		})
	}
}

// TestSession_calculateRetryIntervalWithJitter_Concurrency は並行アクセステスト
func TestSession_calculateRetryIntervalWithJitter_Concurrency(t *testing.T) {
	session := &Session{
		RetryInterval: 1 * time.Second,
	}

	const numGoroutines = 100
	const numIterations = 10

	var wg sync.WaitGroup
	results := make(chan time.Duration, numGoroutines*numIterations)

	// 複数のgoroutineから同時にアクセス
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				interval := session.calculateRetryIntervalWithJitter()
				results <- interval
			}
		}()
	}

	wg.Wait()
	close(results)

	// 結果を収集
	var intervals []time.Duration
	for interval := range results {
		intervals = append(intervals, interval)
	}

	// すべての結果が有効な範囲内にあることを確認
	baseInterval := session.RetryInterval
	minAllowed := time.Duration(float64(baseInterval) * MinIntervalRatio)
	maxAllowed := time.Duration(float64(baseInterval) * (1.0 + JitterPercentage))

	for i, interval := range intervals {
		if interval < minAllowed || interval > maxAllowed {
			t.Errorf("Interval %d (%v) is outside expected range [%v, %v]", i, interval, minAllowed, maxAllowed)
		}
	}

	// 結果の多様性を確認（すべて同じでないことを確認）
	uniqueValues := make(map[time.Duration]bool)
	for _, interval := range intervals {
		uniqueValues[interval] = true
	}

	// 少なくとも10個の異なる値があることを期待
	if len(uniqueValues) < 10 {
		t.Errorf("Expected at least 10 unique values, got %d", len(uniqueValues))
	}

	t.Logf("Generated %d unique values from %d total calls", len(uniqueValues), len(intervals))
}

// TestSession_calculateRetryIntervalWithJitter_ThreadSafety はスレッドセーフ性のテスト
func TestSession_calculateRetryIntervalWithJitter_ThreadSafety(t *testing.T) {
	session := &Session{
		RetryInterval: 500 * time.Millisecond,
	}

	const numGoroutines = 50
	const numIterations = 100

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numIterations)

	// 複数のgoroutineから同時に大量アクセス
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				interval := session.calculateRetryIntervalWithJitter()

				// 基本的な妥当性チェック
				if interval <= 0 {
					errors <- fmt.Errorf("goroutine %d iteration %d: got non-positive interval %v", routineID, j, interval)
					return
				}

				maxExpected := time.Duration(float64(session.RetryInterval) * (1.0 + JitterPercentage))
				if interval > maxExpected*2 { // 異常に大きな値でないことを確認
					errors <- fmt.Errorf("goroutine %d iteration %d: got unexpectedly large interval %v", routineID, j, interval)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// エラーがないことを確認
	for err := range errors {
		t.Error(err)
	}
}
