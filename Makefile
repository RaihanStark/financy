APP     := financy
PKG     := github.com/raihanstark/financy
# Current version is read from FyneApp.toml unless overridden: make build VERSION=0.2.0
VERSION ?= $(shell sed -n 's/^Version = "\(.*\)"/\1/p' FyneApp.toml)
LDFLAGS := -X $(PKG)/internal/core.Version=$(VERSION)

# Hugo for the docs site: use it from PATH, else fall back to the go-installed
# binary in GOPATH/bin (where `go install ... hugo` puts it).
HUGO := $(shell command -v hugo 2>/dev/null || echo $(shell go env GOPATH)/bin/hugo)

.PHONY: help run test vet check build shot set-version package release clean docs docs-build

help:
	@echo "Financy — make targets:"
	@echo "  make run                      run the app (go run .)"
	@echo "  make test                     run all tests"
	@echo "  make check                    build + vet + test"
	@echo "  make build                    build ./$(APP) with version $(VERSION)"
	@echo "  make shot                     regenerate UI screenshots into /tmp/financy-shots"
	@echo "  make docs                     serve the docs site locally (http://localhost:1313)"
	@echo "  make docs-build               build the docs to website/public"
	@echo "  make set-version VERSION=x.y.z  stamp version into code + FyneApp.toml"
	@echo "  make package                  package for THIS OS (needs the fyne CLI)"
	@echo "  make release VERSION=x.y.z    stamp, verify, build — then commit & tag"

run:
	go run .

test:
	go test ./...

vet:
	go vet ./...

check:
	go build ./...
	go vet ./...
	go test ./...

build:
	go build -ldflags "$(LDFLAGS)" -o $(APP) .

shot:
	go run . shot /tmp/financy-shots

# Serve the docs site locally with live reload at http://localhost:1313/ .
# Needs Hugo Extended: CGO_ENABLED=1 go install -tags extended github.com/gohugoio/hugo@latest
docs:
	@command -v $(HUGO) >/dev/null 2>&1 || { echo "Hugo Extended not found at '$(HUGO)'."; echo "Install: CGO_ENABLED=1 go install -tags extended github.com/gohugoio/hugo@latest"; exit 1; }
	$(HUGO) server --source website --baseURL http://localhost:1313/

# Build the docs to website/public (production-like check).
docs-build:
	@command -v $(HUGO) >/dev/null 2>&1 || { echo "Hugo Extended not found at '$(HUGO)'."; echo "Install: CGO_ENABLED=1 go install -tags extended github.com/gohugoio/hugo@latest"; exit 1; }
	$(HUGO) --source website --gc --minify

# Stamp the version into the in-app constant and the packaging metadata.
set-version:
	@test -n "$(VERSION)" || (echo "VERSION is required, e.g. make set-version VERSION=0.2.0"; exit 1)
	perl -pi -e 's/^var Version = .*/var Version = "$(VERSION)"/' internal/core/version.go
	perl -pi -e 's/^Version = .*/Version = "$(VERSION)"/' FyneApp.toml
	@echo "Stamped version $(VERSION)"

package: build
	fyne package

# Local release prep: stamp + verify + build. Then commit and tag to trigger CI.
release:
	@test -n "$(VERSION)" || (echo "VERSION is required, e.g. make release VERSION=0.2.0"; exit 1)
	$(MAKE) set-version VERSION=$(VERSION)
	$(MAKE) check
	$(MAKE) build VERSION=$(VERSION)
	@echo ""
	@echo "Release $(VERSION) is ready. Now:"
	@echo "  git commit -am \"Release v$(VERSION)\""
	@echo "  git tag v$(VERSION) && git push --tags   # CI builds & publishes the bundles"

clean:
	rm -f $(APP)
