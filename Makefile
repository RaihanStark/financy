APP     := financy
PKG     := github.com/raihanstark/financy
# Current version is read from FyneApp.toml unless overridden: make build VERSION=0.2.0
VERSION ?= $(shell sed -n 's/^Version = "\(.*\)"/\1/p' FyneApp.toml)
LDFLAGS := -X $(PKG)/internal/core.Version=$(VERSION)

# Hugo for the docs site: use it from PATH, else fall back to the go-installed
# binary in GOPATH/bin (where `go install ... hugo` puts it).
HUGO := $(shell command -v hugo 2>/dev/null || echo $(shell go env GOPATH)/bin/hugo)

# Where the screenshot harness writes its PNGs before they're copied into the docs.
SHOTDIR := /tmp/financy-shots

.PHONY: help run run-dev test vet check build shot set-version package nfpm print-version release clean docs docs-build

help:
	@echo "Financy — make targets:"
	@echo "  make run                      run the app (go run .)"
	@echo "  make run-dev                  run with an isolated dev profile + db (won't touch prod state)"
	@echo "  make test                     run all tests"
	@echo "  make check                    build + vet + test"
	@echo "  make build                    build ./$(APP) with version $(VERSION)"
	@echo "  make shot                     regenerate UI screenshots + copy into the docs"
	@echo "  make docs                     serve the docs site locally (http://localhost:1313)"
	@echo "  make docs-build               build the docs to website/public"
	@echo "  make set-version VERSION=x.y.z  stamp version into code + FyneApp.toml"
	@echo "  make package                  package for THIS OS (needs the fyne CLI)"
	@echo "  make nfpm                     build the Linux .deb and .rpm into dist/ (needs the nfpm CLI)"
	@echo "  make release VERSION=x.y.z    stamp, verify, build — then commit & tag"

run:
	go run .

# Run against an ISOLATED dev profile so you never touch your real ("prod") data.
# Financy stores which file to reopen (prefs.json), the recent list, and the Fyne
# settings under the OS config dir. On Linux that dir is $XDG_CONFIG_HOME, so we
# repoint ONLY that at ./.devdata/config — giving dev its own prefs, its own
# recent-files list, and its own .financy database. Your prod ~/.config/financy
# is never read or written. We build with the normal Go cache first (fast; keeps
# the toolchain's cache/telemetry out of .devdata), then run the binary with the
# config redirect. First run shows the setup wizard; save the dev database
# somewhere like ./.devdata/ and it'll reopen next time.
run-dev: build
	@mkdir -p .devdata/config
	XDG_CONFIG_HOME=$(CURDIR)/.devdata/config ./$(APP)

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

# Regenerate every UI screenshot from the current code and copy them into the two
# doc locations (README uses docs/screenshots, the Hugo site uses website/static/img,
# which renames a few). Run this whenever the UI changes so the docs stay current.
shot:
	go run . shot $(SHOTDIR)
	@echo "Copying screenshots into docs/screenshots and website/static/img…"
	@for n in accounts analytics categories data-summary recurring reports setup transactions; do \
		cp "$(SHOTDIR)/$$n.png" "docs/screenshots/$$n.png"; \
	done
	@for n in accounts analytics categories reports transactions; do \
		cp "$(SHOTDIR)/$$n.png" "website/static/img/$$n.png"; \
	done
	@cp "$(SHOTDIR)/recurring.png"        website/static/img/recurring-screen.png
	@cp "$(SHOTDIR)/reconcile-dialog.png" website/static/img/reconcile-dialog.png
	@cp "$(SHOTDIR)/reconcile-result.png" website/static/img/reconcile-result.png
	@cp "$(SHOTDIR)/recurring-due.png"    website/static/img/recurring-due.png
	@echo "Screenshots updated."

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

# Print the version the build is stamped with (used by CI/nfpm). Handy on its own.
print-version:
	@echo $(VERSION)

# Build the Linux .deb and .rpm from the freshly built binary using nfpm.
# Install the CLI once: go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest
nfpm: build
	@command -v nfpm >/dev/null 2>&1 || { echo "nfpm not found. Install: go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest"; exit 1; }
	@mkdir -p dist
	VERSION=$(VERSION) nfpm package --config nfpm.yaml --packager deb --target dist/
	VERSION=$(VERSION) nfpm package --config nfpm.yaml --packager rpm --target dist/
	@echo "Built Linux packages in dist/:"
	@ls -1 dist/*.deb dist/*.rpm

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
