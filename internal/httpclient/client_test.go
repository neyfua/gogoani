package httpclient

import (
	"testing"
)

func TestGetBufPutBuf(t *testing.T) {
	buf1 := GetBuf()
	if buf1 == nil {
		t.Fatal("GetBuf() returned nil")
	}

	// Write some data
	buf1.WriteString("test data")
	if buf1.String() != "test data" {
		t.Errorf("Buffer contains %q, want %q", buf1.String(), "test data")
	}

	// Return to pool
	PutBuf(buf1)

	// Get another buffer - should be reset
	buf2 := GetBuf()
	if buf2 == nil {
		t.Fatal("GetBuf() returned nil on second call")
	}

	// Buffer should be empty after being returned to pool
	if buf2.Len() != 0 {
		t.Errorf("Pooled buffer has length %d, want 0", buf2.Len())
	}

	PutBuf(buf2)
}

func TestUserAgent(t *testing.T) {
	if UserAgent == "" {
		t.Error("UserAgent constant is empty")
	}
}

func TestClientNotNil(t *testing.T) {
	if Client == nil {
		t.Error("Client is nil after package initialization")
	}
}
