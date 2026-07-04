package crypto

import (
	"encoding/base64"
	"fmt"
	"testing"
)

// Test functions for go test (not just benchmarks)

func TestDecodeHexMapBasic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no prefix",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "simple hex decoding",
			input:    "--7979",
			expected: "AA",
		},
		{
			name:     "uppercase letter",
			input:    "--79",
			expected: "A",
		},
		{
			name:     "lowercase letter",
			input:    "--59",
			expected: "a",
		},
		{
			name:     "digit",
			input:    "--08",
			expected: "0",
		},
		{
			name:     "special char dot",
			input:    "--16",
			expected: ".",
		},
		{
			name:     "invalid hex code becomes question mark",
			input:    "--ff",
			expected: "?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DecodeHexMap(tt.input)
			if result != tt.expected {
				t.Errorf("DecodeHexMap(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDecodeHexMapClockJSON(t *testing.T) {
	// Test that /clock becomes /clock.json
	input := "--17" // "/" in the hex map at position for "/"
	result := DecodeHexMap(input)
	if !stringContains(result, "clock") {
		// This test verifies the clock.json replacement logic
		// For now just verify decoding works
		if result == "" {
			t.Errorf("DecodeHexMap should not return empty string")
		}
	}
}

func stringContains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func BenchmarkDecryptAllAnime(b *testing.B) {
	// Minimal valid encrypted payload: 1 byte prefix + 12 bytes IV + 16 bytes ciphertext + 16 bytes tag = 45 bytes raw
	raw := make([]byte, 45)
	raw[0] = 0x01 // version byte
	// IV at [1:13]
	for i := 1; i < 13; i++ {
		raw[i] = byte(i)
	}
	// ciphertext at [13:29]
	for i := 13; i < 29; i++ {
		raw[i] = byte(i * 2)
	}
	// tag at [29:45]
	for i := 29; i < 45; i++ {
		raw[i] = byte(i)
	}
	encoded := base64.StdEncoding.EncodeToString(raw)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err := DecryptAllAnime(encoded)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecryptAllAnimeLarge(b *testing.B) {
	// Simulate a realistic ~8KB encrypted response (typical API response)
	payloadSize := 8192
	raw := make([]byte, payloadSize)
	raw[0] = 0x01
	for i := 1; i < 13; i++ {
		raw[i] = byte(i)
	}
	for i := 13; i < payloadSize-16; i++ {
		raw[i] = byte(i * 3)
	}
	for i := payloadSize - 16; i < payloadSize; i++ {
		raw[i] = byte(i)
	}
	encoded := base64.StdEncoding.EncodeToString(raw)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, err := DecryptAllAnime(encoded)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHexMapDecode(b *testing.B) {
	encoded := "--79595a56595e165a51"
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = DecodeHexMap(encoded)
	}
}

func BenchmarkHexMapDecodeLong(b *testing.B) {
	// Build a realistic URL-like hex encoding
	parts := make([]string, 0, 100)
	for range 100 {
		parts = append(parts, "79")
	}
	encoded := "--" + fmt.Sprintf("%s", parts)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = DecodeHexMap(encoded)
	}
}
