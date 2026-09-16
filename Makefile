# Copyright 2019 The Skaffold Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
GOPATH ?= $(shell go env GOPATH)
GOBIN ?= $(or $(shell go env GOBIN),$(GOPATH)/bin)
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
BUILD_DIR ?= ./out
ORG = github.com/lucky-tools
PROJECT = devloop
REPOPATH ?= $(ORG)/$(PROJECT)

SUPPORTED_PLATFORMS = linux-amd64 darwin-amd64 windows-amd64.exe linux-arm64 darwin-arm64
BUILD_PACKAGE = $(REPOPATH)/cmd/devloop

DEVLOOP_TEST_PACKAGES = ./pkg/devloop/... ./cmd/... ./hack/... ./pkg/webhook/...
GO_FILES = $(shell find . -type f -name '*.go' -not -path "./pkg/diag/*")

VERSION_PACKAGE = $(REPOPATH)/pkg/devloop/version
COMMIT = $(shell git rev-parse HEAD)

ifeq "$(strip $(VERSION))" ""
	override VERSION = $(shell git describe --always --tags --dirty)
endif

DATE_FMT = +%Y-%m-%dT%H:%M:%SZ
ifdef SOURCE_DATE_EPOCH
    BUILD_DATE ?= $(shell date -u -d "@$(SOURCE_DATE_EPOCH)" "$(DATE_FMT)" 2>/dev/null || date -u -r "$(SOURCE_DATE_EPOCH)" "$(DATE_FMT)" 2>/dev/null || date -u "$(DATE_FMT)")
else
    BUILD_DATE ?= $(shell date "$(DATE_FMT)")
endif

GO_LDFLAGS = -X $(VERSION_PACKAGE).version=$(VERSION)
GO_LDFLAGS += -X $(VERSION_PACKAGE).buildDate=$(BUILD_DATE)
GO_LDFLAGS += -X $(VERSION_PACKAGE).gitCommit=$(COMMIT)
GO_LDFLAGS += -s -w

GO_BUILD_TAGS = timetzdata

GO_BUILD_TAGS_linux = osusergo netgo static_build release
LDFLAGS_linux = -static

GO_BUILD_TAGS_windows = release

GO_BUILD_TAGS_darwin = release

ifneq "$(strip $(LOCAL))" "true"
	override EMBEDDED_FILES_CHECK = fs/assets/check.txt
endif

# when build for local development (`LOCAL=true make install` can skip license check)
$(BUILD_DIR)/$(PROJECT): $(EMBEDDED_FILES_CHECK) $(GO_FILES) $(BUILD_DIR)
	$(eval ldflags = $(GO_LDFLAGS) $(patsubst %,-extldflags \"%\",$(LDFLAGS_$(GOOS))))
	$(eval tags = $(GO_BUILD_TAGS) $(GO_BUILD_TAGS_$(GOOS)) $(GO_BUILD_TAGS_$(GOOS)_$(GOARCH)))
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=1 \
	    go build -mod="vendor" -gcflags="all=-N -l" -tags "$(tags)" -ldflags "$(ldflags)" -o $@ $(BUILD_PACKAGE)
ifeq ($(GOOS),darwin)
	codesign --force --deep --sign - $@
endif

.PHONY: install
install: $(BUILD_DIR)/$(PROJECT)
	mkdir -p $(GOPATH)/bin
	rm -f $(GOBIN)/$(PROJECT)
	cp $(BUILD_DIR)/$(PROJECT) $(GOBIN)/$(PROJECT)

# Optional: compress the built binary with UPX (https://upx.github.io/) to cut
# the ~100MB binary down to roughly a third. Not on by default because UPX-packed
# Windows binaries are often flagged by antivirus and startup is slightly slower.
.PHONY: compress
compress: $(BUILD_DIR)/$(PROJECT)
	@command -v upx >/dev/null 2>&1 || { echo "UPX not found: install from https://upx.github.io/"; exit 1; }
	@test -f $(BUILD_DIR)/$(PROJECT) && upx --best $(BUILD_DIR)/$(PROJECT) || true
	@test -f $(BUILD_DIR)/$(PROJECT).exe && upx --best $(BUILD_DIR)/$(PROJECT).exe || true

.PRECIOUS: $(foreach platform, $(SUPPORTED_PLATFORMS), $(BUILD_DIR)/$(PROJECT)-$(platform))

.PHONY: cross
cross: $(foreach platform, $(SUPPORTED_PLATFORMS), $(BUILD_DIR)/$(PROJECT)-$(platform))

$(BUILD_DIR)/$(PROJECT)-%: $(EMBEDDED_FILES_CHECK) $(GO_FILES) $(BUILD_DIR)
	$(eval os = $(firstword $(subst -, ,$*)))
	$(eval arch = $(lastword $(subst -, ,$(subst .exe,,$*))))
	$(eval ldflags = $(GO_LDFLAGS) $(patsubst %,-extldflags \"%\",$(LDFLAGS_$(os))))
	$(eval tags = $(GO_BUILD_TAGS) $(GO_BUILD_TAGS_$(os)) $(GO_BUILD_TAGS_$(os)_$(arch)))
	GOOS=$(os) GOARCH=$(arch) CGO_ENABLED=1 go build -mod="vendor" -tags "$(tags)" -ldflags "$(ldflags)" -o $@ ./cmd/devloop
	(cd `dirname $@`; shasum -a 256 `basename $@`) | tee $@.sha256
	file $@ || true

.PHONY: $(BUILD_DIR)/VERSION
$(BUILD_DIR)/VERSION: $(BUILD_DIR)
	@ echo $(VERSION) > $@

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

.PHONY: test
test: $(BUILD_DIR)
	@ ./hack/gotest.sh -count=1 -race -short -timeout=90s $(DEVLOOP_TEST_PACKAGES)
	@ ./hack/checks.sh
	@ ./hack/linters.sh

.PHONY: unit-tests
unit-tests: $(BUILD_DIR)
	@ ./hack/gotest.sh -count=1 -race -short -timeout=90s $(DEVLOOP_TEST_PACKAGES)

.PHONY: coverage
coverage: $(BUILD_DIR)
    # https://go-review.git.corp.google.com/c/go/+/569575
	@ ./hack/gotest.sh -count=1 -race -cover -short -timeout=90s -coverprofile=out/coverage.txt -coverpkg="./pkg/...,./cmd/..." $(DEVLOOP_TEST_PACKAGES)
	@- curl -s https://codecov.io/bash > $(BUILD_DIR)/upload_coverage && bash $(BUILD_DIR)/upload_coverage

.PHONY: checks
checks: $(BUILD_DIR)
	@ ./hack/checks.sh

.PHONY: linters
linters: $(BUILD_DIR)
	@ ./hack/linters.sh

.PHONY: quicktest
quicktest:
	@ ./hack/gotest.sh -short -timeout=60s $(DEVLOOP_TEST_PACKAGES)

.PHONY: integration-tests
integration-tests:
	@ ./hack/gotest.sh -v $(REPOPATH)/v2/integration -timeout 50m $(INTEGRATION_TEST_ARGS)

.PHONY: integration
integration: install integration-tests

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR) hack/bin $(EMBEDDED_FILES_CHECK) fs/assets/schemas_generated/

# schema generation

.PHONY: generate-schemas
generate-schemas:
	go run hack/schemas/main.go

.PHONY: generate-schemas-v2
generate-schemas-v2:
	go run hack/schemas/main.go

# telemetry generation
.PHONY: generate-telemetry-json-v2
generate-telemetry-json-v2:
	go run hack/struct-json/main.go -- pkg/devloop/instrumentation/types.go docs-v2/content/en/docs/resources/telemetry/metrics.json

# static files

$(EMBEDDED_FILES_CHECK): go.mod docs-v2/content/en/schemas/*
	hack/generate-embedded-files.sh

# run comparisonstats - ex: make COMPARISONSTATS_ARGS='usr/local/bin/devloop /usr/local/bin/devloop helm-deployment main.go "//per-dev-iteration-comment"' comparisonstats
.PHONY: comparisonstats
comparisonstats:
	go run hack/comparisonstats/main.go $(COMPARISONSTATS_ARGS)
