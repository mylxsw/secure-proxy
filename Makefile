BIN := secure-proxy
LDFLAGS := -s -w -X main.Version=$(shell date "+%Y%m%d%H%M") -X main.GitCommit=$(shell git rev-parse --short HEAD)

run: build
	./build/debug/$(BIN) --conf ./secure-proxy.yaml

build-tool:
	go build -race -ldflags "$(LDFLAGS)"  -o build/debug/$(BIN)-tool cmd/tool/main.go

build: build-static
	go build -race -ldflags "$(LDFLAGS)" -o build/debug/$(BIN) cmd/server/main.go

build-release: build-static
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o build/release/$(BIN) cmd/server/main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o build/release/$(BIN)-tool cmd/tool/main.go

build-static:
	#go get -u github.com/mjibson/esc
	esc -pkg handler -o internal/handler/static.go -prefix=assets assets

clean:
	rm -fr build/debug/ build/release/

.PHONY: run build build-release clean build-static deploy build-tool
