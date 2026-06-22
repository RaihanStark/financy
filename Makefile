APP     := financy
PKG     := github.com/raihanstark/financy
# Current version is read from FyneApp.toml unless overridden: make build VERSION=0.2.0
VERSION ?= $(shell sed -n 's/^Version = "\(.*\)"/\1/p' FyneApp.toml)
LDFLAGS := -X $(PKG)/internal/core.Version=$(VERSION)

.PHONY: help run test vet check build shot set-version package release clean

help:
	@echo "Financy — make targets:"
	@echo "  make run                      run the app (go run .)"
	@echo "  make test                     run all tests"
	@echo "  make check                    build + vet + test"
	@echo "  make build                    build ./$(APP) with version $(VERSION)"
	@echo "  make shot                     regenerate UI screenshots into /tmp/financy-shots"
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
