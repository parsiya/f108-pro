//go:build darwin

package aula

import (
	"errors"
	"testing"
)

func TestRetryDarwinHIDOperationReturnsImmediatelyOnSuccess(t *testing.T) {
	attempts := 0
	n, err := retryDarwinHIDOperation(func() (int, error) {
		attempts++
		return 65, nil
	})
	if err != nil {
		t.Fatalf("retryDarwinHIDOperation: %v", err)
	}
	if n != 65 {
		t.Fatalf("result = %d, want 65", n)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestRetryDarwinHIDOperationRetriesTransientErrors(t *testing.T) {
	transientErr := errors.New("transient IOKit error")
	attempts := 0
	n, err := retryDarwinHIDOperation(func() (int, error) {
		attempts++
		if attempts < hidRetryAttempts {
			return 0, transientErr
		}
		return 65, nil
	})
	if err != nil {
		t.Fatalf("retryDarwinHIDOperation: %v", err)
	}
	if n != 65 {
		t.Fatalf("result = %d, want 65", n)
	}
	if attempts != hidRetryAttempts {
		t.Fatalf("attempts = %d, want %d", attempts, hidRetryAttempts)
	}
}

func TestRetryDarwinHIDOperationReturnsLastError(t *testing.T) {
	transientErr := errors.New("transient IOKit error")
	attempts := 0
	n, err := retryDarwinHIDOperation(func() (int, error) {
		attempts++
		return attempts, transientErr
	})
	if !errors.Is(err, transientErr) {
		t.Fatalf("error = %v, want %v", err, transientErr)
	}
	if n != hidRetryAttempts {
		t.Fatalf("result = %d, want %d", n, hidRetryAttempts)
	}
	if attempts != hidRetryAttempts {
		t.Fatalf("attempts = %d, want %d", attempts, hidRetryAttempts)
	}
}
