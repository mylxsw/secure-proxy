package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/mylxsw/go-toolkit/jsonutils"
)

type Request struct {
	Method     string
	RequestURI string
	Header     http.Header
	Body       []byte
	Query      string
}

func NewRequest(req *http.Request) *Request {
	return &Request{
		Method:     req.Method,
		RequestURI: req.URL.Path,
		Header:     req.Header,
		Query:      req.URL.Query().Encode(),
		Body:       ExtractBodyFromRequest(req),
	}
}

// ExtractBodyFromRequest 从请求中获取body，可重复使用
func ExtractBodyFromRequest(req *http.Request) []byte {
	body, _ := ioutil.ReadAll(req.Body)
	_ = req.Body.Close()
	req.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	return body
}

func (req *Request) Serialize() string {
	data, _ := json.Marshal(req)
	return string(data)
}

func (req *Request) UnSerialize(data string) {
	_ = json.Unmarshal([]byte(data), req)
}

func (req *Request) HttpRequest(gatewayURL string) *http.Request {
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	request, _ := http.NewRequestWithContext(
		ctx,
		req.Method,
		strings.TrimRight(gatewayURL, "/")+req.RequestURI,
		bytes.NewBuffer(req.Body),
	)
	request.URL.RawQuery = req.Query

	if req.Header != nil {
		request.Header = req.Header
	}

	return request
}

type KvPairs []jsonutils.KvPair

func (k KvPairs) Len() int {
	return len(k)
}

func (k KvPairs) Less(i, j int) bool {
	return k[i].Key < k[j].Key
}

func (k KvPairs) Swap(i, j int) {
	k[i], k[j] = k[j], k[i]
}
