BINARIO  := elitechlog
DIST     := dist
VERSION  ?= dev
COMMIT   := $(shell git rev-parse --short HEAD 2>/dev/null || echo desconocido)
FECHA    := $(shell date -u +%Y-%m-%d)
LDFLAGS  := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(FECHA)

export CGO_ENABLED = 0

.PHONY: all test vet fmt build windows linux darwin darwin-amd64 darwin-arm64 clean

all: test build

test:
	go test ./... -race

vet:
	go vet ./...

fmt:
	gofmt -w .

build: windows linux darwin

# Todos los binarios se compilan desde cualquier sistema: el proyecto solo usa
# la librería estándar, así que no hace falta tener Windows ni Mac a mano.
windows:
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARIO).exe ./cmd/$(BINARIO)

linux:
	GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARIO) ./cmd/$(BINARIO)

darwin: darwin-amd64 darwin-arm64

darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARIO)_darwin_amd64 ./cmd/$(BINARIO)

darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARIO)_darwin_arm64 ./cmd/$(BINARIO)

clean:
	rm -rf $(DIST)
