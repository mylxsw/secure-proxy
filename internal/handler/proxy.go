package handler

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	stdLog "log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mylxsw/secure-proxy/config"
	"github.com/mylxsw/secure-proxy/internal/auth"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/go-utils/str"
)

const (
	OriginalHostHeader = "x-secure-proxy-host"
	AccountHostHeader  = "x-secure-proxy-account"
)

// ProxyHandler handles proxy requests
type ProxyHandler struct {
	requestTimeout  time.Duration
	logger          log.Logger
	option          *ProxyOptions
	author          auth.Auth
	transport       *http.Transport
	reverseProxies  map[string]*httputil.ReverseProxy
	reverseProxyMu  sync.RWMutex
}

// BackendProvider 后端服务提供者接口
type BackendProvider interface {
	Get(host string) (config.Backend, bool)
	GetAll() map[string]config.Backend
	ReloadIfNeeded()
}

// ProxyOptions defines proxy configuration options
type ProxyOptions struct {
	config          *config.Config
	backendProvider BackendProvider
	Timeout         time.Duration
	Director        func(req *http.Request)
	AuthHandler     func(req *http.Request) (*config.UserAuthInfo, error)
}

// DefaultOptions returns default proxy configuration options
func DefaultOptions(conf *config.Config) *ProxyOptions {
	return &ProxyOptions{
		config:   conf,
		Timeout:  60 * time.Second,
		Director: func(req *http.Request) {},
	}
}

// SetBackendProvider 设置后端服务提供者
func (o *ProxyOptions) SetBackendProvider(provider BackendProvider) {
	o.backendProvider = provider
}

// staticBackendProvider 静态后端服务提供者（用于配置文件模式）
type staticBackendProvider struct {
	backends map[string]config.Backend
}

func newStaticBackendProvider(conf *config.Config) *staticBackendProvider {
	backends := make(map[string]config.Backend)
	for _, backend := range conf.Backends {
		backends[backend.Host] = backend
	}
	return &staticBackendProvider{backends: backends}
}

func (p *staticBackendProvider) Get(host string) (config.Backend, bool) {
	b, ok := p.backends[host]
	return b, ok
}

func (p *staticBackendProvider) GetAll() map[string]config.Backend {
	return p.backends
}

func (p *staticBackendProvider) ReloadIfNeeded() {
	// 静态配置不需要重载
}

// NewProxyHandler creates a new proxy request handler
func NewProxyHandler(option *ProxyOptions, author auth.Auth, logger log.Logger) *ProxyHandler {
	// 如果没有设置 BackendProvider，使用静态配置
	if option.backendProvider == nil {
		option.backendProvider = newStaticBackendProvider(option.config)
	}

	// Create a custom transport for WebSocket-friendly long-lived connections
	// Based on http.DefaultTransport but with HTTP/2 disabled for WebSocket compatibility
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     false, // Disable HTTP/2 for WebSocket compatibility (RFC 6455 requires HTTP/1.1)
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,               // Increase from default 2 for better proxy performance
		IdleConnTimeout:       90 * time.Second, // Default value; WebSocket connections are hijacked and bypass this
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	ph := &ProxyHandler{
		transport:      transport,
		reverseProxies: make(map[string]*httputil.ReverseProxy),
		requestTimeout: option.Timeout,
		logger:         logger,
		option:         option,
		author:         author,
	}

	// 初始化反向代理
	ph.initReverseProxies()

	return ph
}

// initReverseProxies 初始化反向代理
func (ph *ProxyHandler) initReverseProxies() {
	backends := ph.option.backendProvider.GetAll()

	ph.reverseProxyMu.Lock()
	defer ph.reverseProxyMu.Unlock()

	for _, backend := range backends {
		ph.createReverseProxy(backend)
	}
}

// createReverseProxy 创建单个反向代理
func (ph *ProxyHandler) createReverseProxy(backend config.Backend) {
	gateway, _ := url.Parse(backend.GetUpstream())
	proxy := httputil.NewSingleHostReverseProxy(gateway)
	proxy.Transport = ph.transport
	proxy.ErrorLog = stdLog.New(LogWriter{logger: ph.logger}, "", stdLog.Flags())

	originalDirector := proxy.Director
	host := backend.Host
	rewriteHeaders := backend.RewriteHeaders
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = gateway.Host
		req.Header.Add(OriginalHostHeader, host)

		for _, h := range rewriteHeaders {
			if h.Key == "" {
				continue
			}

			if h.Value == "" {
				req.Header.Del(h.Key)
			} else {
				req.Header.Set(h.Key, h.Value)
			}
		}

		ph.option.Director(req)
	}

	ph.reverseProxies[backend.Host] = proxy
	log.With(backend).Debugf("add reverse proxy")
}

// getOrCreateReverseProxy 获取或创建反向代理
func (ph *ProxyHandler) getOrCreateReverseProxy(host string) (*httputil.ReverseProxy, config.Backend, bool) {
	// 触发后端配置的定期刷新检查
	ph.option.backendProvider.ReloadIfNeeded()

	// 获取 backend 配置
	backend, ok := ph.option.backendProvider.Get(host)
	if !ok {
		return nil, config.Backend{}, false
	}

	// 检查是否已有对应的反向代理
	ph.reverseProxyMu.RLock()
	rp, exists := ph.reverseProxies[host]
	ph.reverseProxyMu.RUnlock()

	if exists {
		return rp, backend, true
	}

	// 创建新的反向代理
	ph.reverseProxyMu.Lock()
	defer ph.reverseProxyMu.Unlock()

	// 双重检查
	if rp, exists = ph.reverseProxies[host]; exists {
		return rp, backend, true
	}

	ph.createReverseProxy(backend)
	return ph.reverseProxies[host], backend, true
}

func (ph *ProxyHandler) RegisterHandlers() {
	http.Handle("/", ph)
}

// ServeHTTP handles HTTP requests
func (ph *ProxyHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	startTs := time.Now()

	host := request.Host

	// Check if this is a protocol upgrade request (e.g., WebSocket)
	// Look for Connection: Upgrade header and Upgrade header presence
	connectionHeader := strings.ToLower(request.Header.Get("Connection"))
	upgradeHeader := request.Header.Get("Upgrade")
	isUpgradeRequest := strings.Contains(connectionHeader, "upgrade") && upgradeHeader != ""

	// For upgrade requests, check if the original writer supports hijacking
	if isUpgradeRequest {
		if _, ok := writer.(http.Hijacker); !ok {
			// Original writer doesn't support hijacking, return error
			http.Error(writer, "WebSocket upgrade not supported by this server", http.StatusNotImplemented)
			return
		}
	}

	var targetResponse *ResponseWriter
	if !isUpgradeRequest {
		targetResponse = &ResponseWriter{
			Headers:        make(http.Header),
			body:           bytes.NewBuffer([]byte{}),
			StatusCode:     0,
			CreatedAt:      time.Now(),
			originalWriter: writer,
		}
	}

	var clientIP string
	if ph.option.config.ClientRealIPHeader != "" {
		// Get real IP from X-Forwarded-For header, first IP is the real client IP
		clientIP = strings.Split(request.Header.Get(ph.option.config.ClientRealIPHeader), ",")[0]
	}
	if clientIP == "" {
		clientIP = strings.Split(request.RemoteAddr, ":")[0]
	}

	defer func() {
		if err := recover(); err != nil {
			statusCode := 0
			if targetResponse != nil {
				statusCode = targetResponse.StatusCode
			}
			log.F(log.M{
				"elapse":      time.Since(startTs).Microseconds(),
				"host":        host,
				"url":         request.RequestURI,
				"method":      request.Method,
				"ua":          request.UserAgent(),
				"remote":      clientIP,
				"referer":     request.Referer(),
				"status_code": statusCode,
				"upgrade":     isUpgradeRequest,
			}).Errorf("proxy handler panic: %v", err)
		}
	}()

	userAuthInfo, err := ph.option.AuthHandler(request)
	if err != nil {
		if !isUpgradeRequest {
			targetResponse.StatusCode = http.StatusSeeOther
		}
		http.Redirect(writer, request, "/secure-proxy/auth", http.StatusSeeOther)
		return
	}

	// Exclude static resources from logging
	if ph.option.config.Verbose || !str.HasSuffixes(request.URL.Path, []string{".js", ".css", ".jpeg", ".bmp", ".jpg", ".png", ".gif", ".svg", ".font", ".ico", ".woff2", ".ttf"}) {
		defer func() {
			statusCode := 0
			if targetResponse != nil {
				statusCode = targetResponse.StatusCode
			}
			ph.logger.WithFields(log.Fields{
				"elapse":      time.Since(startTs).Seconds(),
				"host":        host,
				"url":         request.RequestURI,
				"method":      request.Method,
				"ua":          request.UserAgent(),
				"remote":      clientIP,
				"referer":     request.Referer(),
				"status_code": statusCode,
				"auth":        userAuthInfo,
				"upgrade":     isUpgradeRequest,
			}).Debugf("request")
		}()
	}

	// Get backend configuration and reverse proxy
	rp, backend, ok := ph.getOrCreateReverseProxy(request.Host)
	if !ok {
		if !isUpgradeRequest {
			targetResponse.StatusCode = http.StatusBadRequest
		}

		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(fmt.Sprintf("Domain %s is not available", request.Host)))

		return
	}

	// Check if user has permission to access backend service
	if !backend.HasPrivilege(userAuthInfo) {
		if !isUpgradeRequest {
			targetResponse.StatusCode = http.StatusForbidden
		}

		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		writer.WriteHeader(http.StatusForbidden)
		_, _ = writer.Write([]byte(fmt.Sprintf("User %s does not have access permission for domain %s. Please contact administrator. If permission has been granted, please visit <a href='http://%s/secure-proxy'>http://%s/secure-proxy</a> to logout and login again",
			userAuthInfo.Name,
			request.Host,
			request.Host,
			request.Host,
		)))

		return
	}

	request.Header.Set(AccountHostHeader, userAuthInfo.Account)

	if isUpgradeRequest {
		// For upgrade requests (like WebSocket), use the original writer directly
		// to support hijacking - we've already verified it supports hijacking above
		// WebSocket connections are long-lived, don't apply timeout
		rp.ServeHTTP(writer, request)
		return
	}

	// Only apply timeout to regular HTTP requests
	ctx, cancel := context.WithTimeout(context.Background(), ph.requestTimeout)
	defer cancel()

	// For regular HTTP requests, use our custom ResponseWriter for buffering
	requestBody := ExtractBodyFromRequest(request)
	rp.ServeHTTP(targetResponse, request.WithContext(ctx))
	request.Body = ioutil.NopCloser(bytes.NewBuffer(requestBody))

	body := targetResponse.body.Bytes()
	for k, v := range targetResponse.Header() {
		for _, vv := range v {
			writer.Header().Add(k, vv)
		}
	}

	writer.Header().Set("Content-Length", strconv.Itoa(len(body)))
	if targetResponse.StatusCode > 0 {
		writer.WriteHeader(targetResponse.StatusCode)
	}

	_, _ = writer.Write(body)
}

// LogWriter rewrites HTTP package request error logs to standard format
type LogWriter struct {
	logger log.Logger
}

// Write implements io.Writer interface
func (l LogWriter) Write(p []byte) (n int, err error) {
	l.logger.Errorf("%s", string(p)[20:])
	return len(p), nil
}
