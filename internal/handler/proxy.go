package handler

import (
	"bytes"
	"context"
	"fmt"
	"github.com/mylxsw/secure-proxy/config"
	"github.com/mylxsw/secure-proxy/internal/auth"
	"io/ioutil"
	stdLog "log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mylxsw/asteria/log"
	"github.com/mylxsw/go-utils/str"
)

const (
	OriginalHostHeader = "x-secure-proxy-host"
	AccountHostHeader  = "x-secure-proxy-account"
)

// ProxyHandler 代理请求处理器
type ProxyHandler struct {
	requestTimeout time.Duration
	logger         log.Logger
	option         *ProxyOptions
	author         auth.Auth
	reverseProxies map[string]*httputil.ReverseProxy
}

// ProxyOptions 代理配置选项
type ProxyOptions struct {
	config      *config.Config
	backends    map[string]config.Backend
	Timeout     time.Duration
	Director    func(req *http.Request)
	AuthHandler func(req *http.Request) (*config.UserAuthInfo, error)
}

// DefaultOptions 默认代理配置选项
func DefaultOptions(conf *config.Config) *ProxyOptions {
	backendsMap := make(map[string]config.Backend)
	for _, backend := range conf.Backends {
		backendsMap[backend.Host] = backend
	}

	return &ProxyOptions{
		config:   conf,
		backends: backendsMap,
		Timeout:  60 * time.Second,
		Director: func(req *http.Request) {},
	}
}

// NewProxyHandler 创建一个代理请求处理器
func NewProxyHandler(option *ProxyOptions, author auth.Auth, logger log.Logger) *ProxyHandler {
	reverseProxies := make(map[string]*httputil.ReverseProxy)
	for _, backend := range option.backends {
		gateway, _ := url.Parse(backend.GetUpstream())
		proxy := httputil.NewSingleHostReverseProxy(gateway)
		proxy.ErrorLog = stdLog.New(LogWriter{logger: logger}, "", stdLog.Flags())

		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			req.Host = gateway.Host
			req.Header.Add(OriginalHostHeader, backend.Host)
			option.Director(req)
		}

		reverseProxies[backend.Host] = proxy
		log.With(backend).Debugf("add reverse proxy")
	}

	return &ProxyHandler{reverseProxies: reverseProxies, requestTimeout: option.Timeout, logger: logger, option: option, author: author}
}

func (ph *ProxyHandler) RegisterHandlers() {
	http.Handle("/", ph)
}

// ServeHTTP 请求处理
func (ph *ProxyHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	startTs := time.Now()

	host := request.Host
	targetResponse := &ResponseWriter{
		Headers:    make(http.Header),
		body:       bytes.NewBuffer([]byte{}),
		StatusCode: 0,
		CreatedAt:  time.Now(),
	}

	var clientIP string
	if ph.option.config.ClientRealIPHeader != "" {
		// 如从请求头 X-Forwarded-For 中获取真实 IP，列表中第一个 IP 为真实的客户端 IP 地址
		clientIP = strings.Split(request.Header.Get(ph.option.config.ClientRealIPHeader), ",")[0]
	}
	if clientIP == "" {
		clientIP = strings.Split(request.RemoteAddr, ":")[0]
	}

	defer func() {
		if err := recover(); err != nil {
			log.F(log.M{
				"elapse":      time.Since(startTs).Microseconds(),
				"host":        host,
				"url":         request.RequestURI,
				"method":      request.Method,
				"ua":          request.UserAgent(),
				"remote":      clientIP,
				"referer":     request.Referer(),
				"status_code": targetResponse.StatusCode,
			}).Errorf("proxy handler panic: %v", err)
		}
	}()

	userAuthInfo, err := ph.option.AuthHandler(request)
	if err != nil {
		targetResponse.StatusCode = http.StatusSeeOther
		http.Redirect(writer, request, "/secure-proxy/auth", http.StatusSeeOther)
		return
	}

	// 排除掉静态资源，不记录日志
	if ph.option.config.Verbose || !str.HasSuffixes(request.URL.Path, []string{".js", ".css", ".jpeg", ".bmp", ".jpg", ".png", ".gif", ".svg", ".font", ".ico", ".woff2", ".ttf"}) {
		defer func() {
			ph.logger.WithFields(log.Fields{
				"elapse":      time.Since(startTs).Seconds(),
				"host":        host,
				"url":         request.RequestURI,
				"method":      request.Method,
				"ua":          request.UserAgent(),
				"remote":      clientIP,
				"referer":     request.Referer(),
				"status_code": targetResponse.StatusCode,
				"auth":        userAuthInfo,
			}).Debugf("request")
		}()
	}

	// 检查是否有权限访问后端服务
	backend, ok := ph.option.backends[request.Host]
	if !ok || !backend.HasPrivilege(userAuthInfo) {
		targetResponse.StatusCode = http.StatusForbidden

		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		writer.WriteHeader(http.StatusForbidden)
		_, _ = writer.Write([]byte(fmt.Sprintf("当前用户 %s 没有对域名 %s 的访问权限，请联系管理员。如管理员已经赋予相应权限，请访问 <a href='http://%s/secure-proxy'>http://%s/secure-proxy</a> 退出重新登陆后再试",
			userAuthInfo.Name,
			request.Host,
			request.Host,
			request.Host,
		)))

		return
	}

	// 获取被反向代理的后端服务
	rp, ok := ph.reverseProxies[request.Host]
	if !ok {
		targetResponse.StatusCode = http.StatusBadRequest

		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(fmt.Sprintf("当前域名 %s 不可用", request.Host)))

		return
	}

	request.Header.Set(AccountHostHeader, userAuthInfo.Account)
	requestBody := ExtractBodyFromRequest(request)

	ctx, cancel := context.WithTimeout(context.Background(), ph.requestTimeout)
	defer cancel()

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

// LogWriter 用于重写http包输出的请求错误日志为标准格式
type LogWriter struct {
	logger log.Logger
}

// Write 实现 io.Writer 接口
func (l LogWriter) Write(p []byte) (n int, err error) {
	l.logger.Errorf("%s", string(p)[20:])
	return len(p), nil
}
