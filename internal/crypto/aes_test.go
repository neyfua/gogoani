package crypto

import (
	"encoding/base64"
	"fmt"
	"testing"
)

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
