package logger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitDefault(t *testing.T) {
	// Just verify Init doesn't panic
	err := Init(false)
	if err != nil {
		t.Errorf("Init(false) returned error: %v", err)
	}
	if Log == nil {
		t.Errorf("Init(false) left Log as nil")
	}
}

func TestLogIsNotNil(t *testing.T) {
	// Log should be initialized at package init
	if Log == nil {
		t.Errorf("Log is nil after package initialization")
	}
}

func TestLogCanWrite(t *testing.T) {
	// Verify we can call logging methods without panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Log methods panicked: %v", r)
		}
	}()

	Log.Debug("test message")
	Log.Info("test message")
	Log.Warn("test message")
	Log.Error("test message")
}

func TestDebugLogDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := Init(true); err != nil {
		t.Fatal(err)
	}
	Log.Debug("test")
	if _, err := os.Stat(filepath.Join(home, ".cache", "gogoani", "gogoani.log")); err != nil {
		t.Fatal(err)
	}
}
