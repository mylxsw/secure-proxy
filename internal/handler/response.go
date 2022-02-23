package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

// ResponseWriter http.ResponseWriter 实现，用于存储代理请求返回数据，避免直接发送给客户端
type ResponseWriter struct {
	Headers    http.Header
	body       *bytes.Buffer
	Body       []byte
	StatusCode int
	CreatedAt  time.Time
}

// Header 实现 http.ResponseWriter 接口
func (rw *ResponseWriter) Header() http.Header {
	return rw.Headers
}

// Write 实现 http.ResponseWriter 接口
func (rw *ResponseWriter) Write(bytes []byte) (int, error) {
	return rw.body.Write(bytes)
}

// WriteHeader 实现 http.ResponseWriter 接口
func (rw *ResponseWriter) WriteHeader(statusCode int) {
	rw.StatusCode = statusCode
}

// Serialize 序列化为 Json
func (rw *ResponseWriter) Serialize() []byte {
	if rw.Body == nil && rw.body != nil {
		rw.Body = rw.body.Bytes()
	}

	data, _ := json.Marshal(rw)
	return data
}

// UnSerialize 从 Json 反序列化
func (rw *ResponseWriter) UnSerialize(data []byte) error {
	if err := json.Unmarshal(data, rw); err != nil {
		return err
	}

	rw.body = bytes.NewBuffer(rw.Body)
	return nil
}
