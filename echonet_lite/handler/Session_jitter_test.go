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

	// 統計的に十分なサンプル数で実行
	const sampleSize = 1000
	intervals := make([]time.Duration, sampleSize)
	for i := 0; i < sampleSize; i++ {
		intervals[i] = session.calculateRetryIntervalWithJitter(0) // retryCount = 0
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

	// 少なくとも10個の異なる値があることを確認（ジッタが機能していることの確認）
	uniqueValues := make(map[time.Duration]bool)
	for _, interval := range intervals {
		uniqueValues[interval] = true
	}
	if len(uniqueValues) < 10 {
		t.Errorf("Expected at least 10 unique values, got %d", len(uniqueValues))
	}

	// 大数の法則により、平均値が基準値に収束することを確認（±3%以内）
	var sum time.Duration
	for _, interval := range intervals {
		sum += interval
	}
	average := sum / time.Duration(len(intervals))

	// サンプルサイズが大きいため、より厳しい許容範囲を使用
	tolerance := 0.03 // 3%
	minAverage := time.Duration(float64(baseInterval) * (1.0 - tolerance))
	maxAverage := time.Duration(float64(baseInterval) * (1.0 + tolerance))

	if average < minAverage || average > maxAverage {
		t.Errorf("Average interval (%v) deviates too much from base interval (%v), expected within [%v, %v]",
			average, baseInterval, minAverage, maxAverage)
	}
}

func TestSession_calculateRetryIntervalWithJitter_MinimumValue(t *testing.T) {
	session := &Session{
		RetryInterval: 100 * time.Millisecond, // 非常に小さい基準値
	}

	// 最小値の保証を確認
	for i := 0; i < 100; i++ {
		interval := session.calculateRetryIntervalWithJitter(0) // retryCount = 0
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

			interval := session.calculateRetryIntervalWithJitter(0) // retryCount = 0

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
				interval := session.calculateRetryIntervalWithJitter(0) // retryCount = 0
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
				interval := session.calculateRetryIntervalWithJitter(0) // retryCount = 0

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

// TestSession_calculateRetryIntervalWithJitter_ExponentialBackoff はexponential backoffのテスト
func TestSession_calculateRetryIntervalWithJitter_ExponentialBackoff(t *testing.T) {
	session := &Session{
		RetryInterval: 1 * time.Second,
	}

	// 各リトライ回数での期待される基準間隔を計算
	expectedBaseIntervals := []time.Duration{
		1 * time.Second,  // retryCount = 0: 1s
		2 * time.Second,  // retryCount = 1: 1s * 2
		4 * time.Second,  // retryCount = 2: 1s * 2^2
		8 * time.Second,  // retryCount = 3: 1s * 2^3
		16 * time.Second, // retryCount = 4: 1s * 2^4
		32 * time.Second, // retryCount = 5: 1s * 2^5
		60 * time.Second, // retryCount = 6: MaxRetryInterval (60s)
		60 * time.Second, // retryCount = 7: MaxRetryInterval (60s)
	}

	for retryCount, expectedBase := range expectedBaseIntervals {
		t.Run(fmt.Sprintf("retryCount=%d", retryCount), func(t *testing.T) {
			// 複数回実行して範囲を確認
			for i := 0; i < 10; i++ {
				interval := session.calculateRetryIntervalWithJitter(retryCount)

				// 最大値を超えないことを確認
				if expectedBase > MaxRetryInterval {
					expectedBase = MaxRetryInterval
				}

				// ジッタを考慮した期待範囲
				minExpected := time.Duration(float64(expectedBase) * (1.0 - JitterPercentage))
				maxExpected := time.Duration(float64(expectedBase) * (1.0 + JitterPercentage))
				minAllowed := time.Duration(float64(expectedBase) * MinIntervalRatio)

				// 範囲内にあることを確認
				if interval < minAllowed {
					t.Errorf("Interval (%v) is less than minimum allowed (%v) for retryCount=%d", interval, minAllowed, retryCount)
				}
				if interval < minExpected || interval > maxExpected {
					t.Errorf("Interval (%v) is outside expected range [%v, %v] for retryCount=%d", interval, minExpected, maxExpected, retryCount)
				}
			}
		})
	}
}

// TestSession_calculateRetryIntervalWithJitter_MaxRetryInterval は最大値制限のテスト
func TestSession_calculateRetryIntervalWithJitter_MaxRetryInterval(t *testing.T) {
	session := &Session{
		RetryInterval: 10 * time.Second, // 大きめの初期値
	}

	// 高いリトライ回数で最大値を超えないことを確認
	for retryCount := 5; retryCount < 10; retryCount++ {
		for i := 0; i < 5; i++ {
			interval := session.calculateRetryIntervalWithJitter(retryCount)

			// MaxRetryIntervalにジッタを加えた値を超えないことを確認
			maxAllowed := time.Duration(float64(MaxRetryInterval) * (1.0 + JitterPercentage))
			if interval > maxAllowed {
				t.Errorf("Interval (%v) exceeds maximum allowed (%v) for retryCount=%d", interval, maxAllowed, retryCount)
			}
		}
	}
}
