package handler

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestTimeoutManager_WithTimeout_Success(t *testing.T) {
	tm := DefaultTimeoutManager()
	ctx := context.Background()

	err := tm.WithTimeout(ctx, "test", 1*time.Second, func() error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestTimeoutManager_WithTimeout_OperationError(t *testing.T) {
	tm := DefaultTimeoutManager()
	ctx := context.Background()
	expectedErr := errors.New("operation failed")

	err := tm.WithTimeout(ctx, "test", 1*time.Second, func() error {
		return expectedErr
	})

	if err != expectedErr {
		t.Errorf("Expected %v, got %v", expectedErr, err)
	}
}

func TestTimeoutManager_WithTimeout_Timeout(t *testing.T) {
	tm := DefaultTimeoutManager()
	ctx := context.Background()

	start := time.Now()
	err := tm.WithTimeout(ctx, "test", 100*time.Millisecond, func() error {
		time.Sleep(200 * time.Millisecond)
		return nil
	})
	duration := time.Since(start)

	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	// タイムアウトが適切に機能していることを確認
	if duration > 150*time.Millisecond {
		t.Errorf("Operation took too long: %v", duration)
	}
}

func TestTimeoutManager_WithDiscoveryTimeout(t *testing.T) {
	tm := DefaultTimeoutManager()
	ctx := context.Background()

	called := false
	err := tm.WithDiscoveryTimeout(ctx, func() error {
		called = true
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !called {
		t.Error("Function was not called")
	}
}

func TestTimeoutManager_WithPropertyUpdateTimeout(t *testing.T) {
	tm := DefaultTimeoutManager()
	ctx := context.Background()

	called := false
	err := tm.WithPropertyUpdateTimeout(ctx, func() error {
		called = true
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !called {
		t.Error("Function was not called")
	}
}
