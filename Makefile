.PHONY: build test clean cross install release-patch release-minor release-major

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

build:
	docker compose run --rm dev go build -ldflags="$(LDFLAGS)" -o bin/ctx .

test:
	docker compose run --rm dev go test ./... -v

clean:
	rm -rf bin/

UNAME_S := $(shell uname -s)
ifeq ($(findstring MINGW,$(UNAME_S)),MINGW)
  GOOS ?= windows
else ifeq ($(findstring MSYS,$(UNAME_S)),MSYS)
  GOOS ?= windows
else ifeq ($(findstring Darwin,$(UNAME_S)),Darwin)
  GOOS ?= darwin
else
  GOOS ?= linux
endif
GOARCH ?= $(if $(filter arm64 aarch64,$(shell uname -m)),arm64,amd64)
EXT := $(if $(filter windows,$(GOOS)),.exe,)
BINARY := bin/ctx$(EXT)

ifeq ($(filter windows,$(GOOS)),windows)
  INSTALL_DIR := $(LOCALAPPDATA)/Programs/ctx
else
  INSTALL_DIR := $(HOME)/bin
endif
install:
	docker compose run --rm dev sh -c "CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags='$(LDFLAGS)' -o $(BINARY) ./cmd"
	mkdir -p "$(INSTALL_DIR)"
	cp $(BINARY) "$(INSTALL_DIR)/ctx$(EXT)"
	@echo "installed ctx $(VERSION) ($(GOOS)/$(GOARCH)) to $(INSTALL_DIR)/ctx$(EXT)"

# --- Release helpers ---
CURRENT_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null || echo v0.0.0)
MAJOR := $(shell echo $(CURRENT_TAG) | sed 's/^v//' | cut -d. -f1)
MINOR := $(shell echo $(CURRENT_TAG) | sed 's/^v//' | cut -d. -f2)
PATCH := $(shell echo $(CURRENT_TAG) | sed 's/^v//' | cut -d. -f3)

release-patch:
	@NEXT=v$(MAJOR).$(MINOR).$(shell echo $$(($(PATCH)+1))); \
	echo "$(CURRENT_TAG) -> $$NEXT"; \
	git tag $$NEXT && git push origin $$NEXT && echo "released $$NEXT"

release-minor:
	@NEXT=v$(MAJOR).$(shell echo $$(($(MINOR)+1))).0; \
	echo "$(CURRENT_TAG) -> $$NEXT"; \
	git tag $$NEXT && git push origin $$NEXT && echo "released $$NEXT"

release-major:
	@NEXT=v$(shell echo $$(($(MAJOR)+1))).0.0; \
	echo "$(CURRENT_TAG) -> $$NEXT"; \
	git tag $$NEXT && git push origin $$NEXT && echo "released $$NEXT"

cross:
	docker compose run --rm dev sh -c "\
		CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='$(LDFLAGS)' -o bin/ctx-linux-amd64 ./cmd && \
		CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags='$(LDFLAGS)' -o bin/ctx-darwin-amd64 ./cmd && \
		CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags='$(LDFLAGS)' -o bin/ctx-darwin-arm64 ./cmd && \
		CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags='$(LDFLAGS)' -o bin/ctx-windows-amd64.exe ./cmd"
