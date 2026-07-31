package httpclient

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/cookiejar"
	"sync"
	"time"

	"github.com/neyfua/gogoani/internal/logger"
)

var transport = &http.Transport{
	MaxIdleConns:        100,
	MaxIdleConnsPerHost: 10,
	IdleConnTimeout:     90 * time.Second,
	DisableCompression:  false,
	TLSHandshakeTimeout: 5 * time.Second,
	ForceAttemptHTTP2:   true,
}

var jar, _ = cookiejar.New(nil)

var Client = &http.Client{
	Timeout:   15 * time.Second,
	Transport: transport,
	Jar:       jar,
}

const UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:150.0) Gecko/20100101 Firefox/150.0"

var bufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

// Request makes an HTTP request with common headers
func Request(ctx context.Context, method, url string, headers map[string]string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		logger.Log.Error("failed to create request", "url", url, "error", err)
		return nil, err
	}

	req.Header.Set("User-Agent", UserAgent)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	logger.Log.Debug("making request", "method", method, "url", url)
	return Client.Do(req)
}

// GetBuf returns a pooled buffer
func GetBuf() *bytes.Buffer {
	return bufPool.Get().(*bytes.Buffer)
}

// PutBuf returns a buffer to the pool
func PutBuf(b *bytes.Buffer) {
	b.Reset()
	bufPool.Put(b)
}
