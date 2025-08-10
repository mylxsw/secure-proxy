package handler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"time"
)

// ResponseWriter implements http.ResponseWriter interface to store proxy response data,
// avoiding direct transmission to client
type ResponseWriter struct {
	Headers    http.Header
	body       *bytes.Buffer
	Body       []byte
	StatusCode int
	CreatedAt  time.Time

	// originalWriter holds reference to the original ResponseWriter for hijacking support
	originalWriter http.ResponseWriter
}

// Header implements http.ResponseWriter interface
func (rw *ResponseWriter) Header() http.Header {
	return rw.Headers
}

// Write implements http.ResponseWriter interface
func (rw *ResponseWriter) Write(bytes []byte) (int, error) {
	return rw.body.Write(bytes)
}

// WriteHeader implements http.ResponseWriter interface
func (rw *ResponseWriter) WriteHeader(statusCode int) {
	rw.StatusCode = statusCode
}

// Serialize converts the response to JSON
func (rw *ResponseWriter) Serialize() []byte {
	if rw.Body == nil && rw.body != nil {
		rw.Body = rw.body.Bytes()
	}

	data, _ := json.Marshal(rw)
	return data
}

// UnSerialize deserializes from JSON
func (rw *ResponseWriter) UnSerialize(data []byte) error {
	if err := json.Unmarshal(data, rw); err != nil {
		return err
	}

	rw.body = bytes.NewBuffer(rw.Body)
	return nil
}

// Hijack implements http.Hijacker interface by delegating to the original writer
func (rw *ResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := rw.originalWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}
