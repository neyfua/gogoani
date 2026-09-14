package httpclient

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"sync"
	"time"

	"github.com/neyfua/gogoani/internal/logger"
)

// fallbackDNS are public resolvers used when the system resolver fails.
var fallbackDNS = []string{"8.8.8.8", "1.1.1.1"}

// resolver tries the system resolver first, then falls back to public DNS.
var resolver = &net.Resolver{
	PreferGo: false,
}

func dialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	// Try system resolver first
	ips, err := resolver.LookupHost(ctx, host)
	if err == nil && len(ips) > 0 {
		return dialFirst(ctx, network, ips, port)
	}
	logger.Log.Debug("system DNS failed, trying fallback resolvers", "host", host, "error", err)

	// Fallback to public resolvers
	for _, dns := range fallbackDNS {
		fb := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: 5 * time.Second}
				return d.DialContext(ctx, "udp", dns+":53")
			},
		}
		ips, err = fb.LookupHost(ctx, host)
		if err == nil && len(ips) > 0 {
			return dialFirst(ctx, network, ips, port)
		}
		logger.Log.Debug("fallback DNS failed", "dns", dns, "host", host, "error", err)
	}
	return nil, &net.DNSError{Err: "all DNS resolvers failed", Name: host, IsTemporary: true}
}

func dialFirst(ctx context.Context, network string, ips []string, port string) (net.Conn, error) {
	d := net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	for _, ip := range ips {
		conn, err := d.DialContext(ctx, network, net.JoinHostPort(ip, port))
		if err == nil {
			return conn, nil
		}
	}
	return nil, &net.DNSError{Err: "dial failed for all resolved IPs", Name: net.JoinHostPort(ips[0], port), IsTemporary: true}
}

var transport = &http.Transport{
	MaxIdleConns:        100,
	MaxIdleConnsPerHost: 10,
	IdleConnTimeout:     90 * time.Second,
	DisableCompression:  false,
	TLSHandshakeTimeout: 5 * time.Second,
	ForceAttemptHTTP2:   true,
	DialContext:         dialContext,
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

// Request makes an HTTP request with common headers, retrying on transient DNS errors.
func Request(ctx context.Context, method, url string, headers map[string]string, body io.Reader) (*http.Response, error) {
	var lastErr error
	for attempt := range 3 {
		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			logger.Log.Error("failed to create request", "url", url, "error", err)
			return nil, err
		}

		req.Header.Set("User-Agent", UserAgent)
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		logger.Log.Debug("making request", "method", method, "url", url, "attempt", attempt+1)
		resp, err := Client.Do(req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if attempt < 2 {
			logger.Log.Debug("request failed, retrying", "url", url, "error", err, "attempt", attempt+1)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt+1) * 500 * time.Millisecond):
			}
		}
	}
	return nil, lastErr
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
