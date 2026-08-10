BINARIO  := elitechlog
DIST     := dist
VERSION  ?= dev
COMMIT   := $(shell git rev-parse --short HEAD 2>/dev/null || echo desconocido)
FECHA    := $(shell date -u +%Y-%m-%d)
LDFLAGS  := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(FECHA)

export CGO_ENABLED = 0

.PHONY: all test vet fmt build windows linux clean

all: test build

test:
	go test ./... -race

vet:
	go vet ./...

fmt:
	gofmt -w .

build: windows linux

# El .exe se compila desde cualquier sistema: el proyecto solo usa la
# librería estándar, así que no hace falta un Windows para generarlo.
windows:
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARIO).exe ./cmd/$(BINARIO)

linux:
	GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARIO) ./cmd/$(BINARIO)

clean:
	rm -rf $(DIST)
