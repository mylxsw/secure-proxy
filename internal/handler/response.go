package handler

import (
	"bytes"
	"encoding/json"
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
