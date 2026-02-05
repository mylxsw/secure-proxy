package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/secure-proxy/config"
)

// mockBackendProvider 模拟后端服务提供者
type mockBackendProvider struct {
	backends map[string]config.Backend
}

func newMockBackendProvider() *mockBackendProvider {
	return &mockBackendProvider{
		backends: make(map[string]config.Backend),
	}
}

func (p *mockBackendProvider) Get(host string) (config.Backend, bool) {
	b, ok := p.backends[host]
	return b, ok
}

func (p *mockBackendProvider) GetAll() map[string]config.Backend {
	return p.backends
}

func (p *mockBackendProvider) ReloadIfNeeded() {}

func (p *mockBackendProvider) AddBackend(b config.Backend) {
	p.backends[b.Host] = b
}

func TestDefaultOptions(t *testing.T) {
	conf := &config.Config{
		Listen: ":8080",
	}

	options := DefaultOptions(conf)

	if options.config != conf {
		t.Error("config should be set")
	}
	if options.Timeout != 60*time.Second {
		t.Errorf("expected timeout 60s, got %v", options.Timeout)
	}
	if options.Director == nil {
		t.Error("Director should not be nil")
	}
}

func TestSetBackendProvider(t *testing.T) {
	conf := &config.Config{}
	options := DefaultOptions(conf)

	provider := newMockBackendProvider()
	options.SetBackendProvider(provider)

	if options.backendProvider != provider {
		t.Error("backendProvider should be set")
	}
}

func TestStaticBackendProvider(t *testing.T) {
	conf := &config.Config{
		Backends: []config.Backend{
			{Host: "test1.example.com", Upstream: "http://localhost:8001", Privilege: "internal"},
			{Host: "test2.example.com", Upstream: "http://localhost:8002", Privilege: "private"},
		},
	}

	provider := newStaticBackendProvider(conf)

	// Test Get
	b1, ok := provider.Get("test1.example.com")
	if !ok {
		t.Error("test1.example.com should exist")
	}
	if b1.Upstream != "http://localhost:8001" {
		t.Errorf("expected upstream http://localhost:8001, got %s", b1.Upstream)
	}

	// Test Get non-existent
	_, ok = provider.Get("nonexistent.example.com")
	if ok {
		t.Error("nonexistent.example.com should not exist")
	}

	// Test GetAll
	all := provider.GetAll()
	if len(all) != 2 {
		t.Errorf("expected 2 backends, got %d", len(all))
	}

	// Test ReloadIfNeeded (should not panic)
	provider.ReloadIfNeeded()
}

func TestProxyHandler_NoBackend(t *testing.T) {
	// Create a mock upstream server
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer upstream.Close()

	conf := &config.Config{
		Backends: []config.Backend{},
	}

	options := DefaultOptions(conf)
	options.AuthHandler = func(req *http.Request) (*config.UserAuthInfo, error) {
		return &config.UserAuthInfo{
			Account: "testuser",
			Name:    "Test User",
			Groups:  []string{"admin"},
		}, nil
	}

	handler := NewProxyHandler(options, nil, nil)

	// Test request to non-existent backend
	req := httptest.NewRequest("GET", "http://nonexistent.example.com/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestProxyHandler_NoAuth(t *testing.T) {
	conf := &config.Config{
		Backends: []config.Backend{
			{Host: "test.example.com", Upstream: "http://localhost:8001", Privilege: "internal"},
		},
	}

	options := DefaultOptions(conf)
	options.AuthHandler = func(req *http.Request) (*config.UserAuthInfo, error) {
		return nil, http.ErrNoCookie
	}

	handler := NewProxyHandler(options, nil, nil)

	req := httptest.NewRequest("GET", "http://test.example.com/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("expected redirect status, got %d", rr.Code)
	}
}

func TestProxyHandler_NoPermission(t *testing.T) {
	conf := &config.Config{
		Backends: []config.Backend{
			{
				Host:      "test.example.com",
				Upstream:  "http://localhost:8001",
				Privilege: "private",
				Groups:    []string{"admin"},
			},
		},
	}

	options := DefaultOptions(conf)
	options.AuthHandler = func(req *http.Request) (*config.UserAuthInfo, error) {
		return &config.UserAuthInfo{
			Account: "testuser",
			Name:    "Test User",
			Groups:  []string{"guest"}, // Not in admin group
		}, nil
	}

	handler := NewProxyHandler(options, nil, nil)

	req := httptest.NewRequest("GET", "http://test.example.com/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, rr.Code)
	}
}

func TestProxyHandler_ProxyRequest(t *testing.T) {
	// Create a mock upstream server
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get(OriginalHostHeader) == "" {
			t.Error("original host header should be set")
		}
		if r.Header.Get(AccountHostHeader) != "testuser" {
			t.Errorf("expected account header 'testuser', got '%s'", r.Header.Get(AccountHostHeader))
		}

		w.Header().Set("X-Test-Response", "success")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello from upstream"))
	}))
	defer upstream.Close()

	conf := &config.Config{
		Backends: []config.Backend{
			{
				Host:      "test.example.com",
				Upstream:  upstream.URL,
				Privilege: "internal",
			},
		},
	}

	options := DefaultOptions(conf)
	options.AuthHandler = func(req *http.Request) (*config.UserAuthInfo, error) {
		return &config.UserAuthInfo{
			Account: "testuser",
			Name:    "Test User",
			Groups:  []string{"admin"},
		}, nil
	}

	handler := NewProxyHandler(options, nil, nil)

	req := httptest.NewRequest("GET", "http://test.example.com/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	body, _ := io.ReadAll(rr.Body)
	if string(body) != "Hello from upstream" {
		t.Errorf("expected body 'Hello from upstream', got '%s'", string(body))
	}
}

func TestProxyHandler_RewriteHeaders(t *testing.T) {
	var receivedHeaders http.Header

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	conf := &config.Config{
		Backends: []config.Backend{
			{
				Host:      "test.example.com",
				Upstream:  upstream.URL,
				Privilege: "internal",
				RewriteHeaders: []config.Header{
					{Key: "X-Custom-Header", Value: "custom-value"},
					{Key: "X-Remove-Header", Value: ""}, // Remove header
				},
			},
		},
	}

	options := DefaultOptions(conf)
	options.AuthHandler = func(req *http.Request) (*config.UserAuthInfo, error) {
		return &config.UserAuthInfo{
			Account: "testuser",
			Name:    "Test User",
			Groups:  []string{"admin"},
		}, nil
	}

	handler := NewProxyHandler(options, nil, nil)

	req := httptest.NewRequest("GET", "http://test.example.com/test", nil)
	req.Header.Set("X-Remove-Header", "should-be-removed")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if receivedHeaders.Get("X-Custom-Header") != "custom-value" {
		t.Errorf("expected X-Custom-Header 'custom-value', got '%s'", receivedHeaders.Get("X-Custom-Header"))
	}
	if receivedHeaders.Get("X-Remove-Header") != "" {
		t.Errorf("X-Remove-Header should be removed, got '%s'", receivedHeaders.Get("X-Remove-Header"))
	}
}

func TestProxyHandler_DynamicBackendProvider(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer upstream.Close()

	conf := &config.Config{}
	options := DefaultOptions(conf)

	// Use dynamic backend provider
	provider := newMockBackendProvider()
	provider.AddBackend(config.Backend{
		Host:      "dynamic.example.com",
		Upstream:  upstream.URL,
		Privilege: "internal",
	})
	options.SetBackendProvider(provider)

	options.AuthHandler = func(req *http.Request) (*config.UserAuthInfo, error) {
		return &config.UserAuthInfo{
			Account: "testuser",
			Name:    "Test User",
			Groups:  []string{"admin"},
		}, nil
	}

	handler := NewProxyHandler(options, nil, nil)

	req := httptest.NewRequest("GET", "http://dynamic.example.com/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestLogWriter(t *testing.T) {
	lw := LogWriter{logger: log.Module("test")}

	// Test Write method
	testData := []byte("12345678901234567890test message")
	n, err := lw.Write(testData)
	if err != nil {
		t.Errorf("Write should not return error: %v", err)
	}
	if n != len(testData) {
		t.Errorf("expected n=%d, got %d", len(testData), n)
	}
}
