PLANTUML = java -DPLANTUML_LIMIT_SIZE=16384 -jar plantuml.jar
SRC_DIR  := diagrams/src
OUT_DIR  := diagrams/out
ARTIFACT_DIR := artifacts/lab1
UNIT_PACKAGES := ./internal/service ./internal/repository ./internal/config ./internal/logging ./internal/middleware ./internal/web
CORE_PACKAGES := ./internal/service ./internal/repository

PUML_FILES := $(wildcard $(SRC_DIR)/*.puml)
PNG_FILES  := $(patsubst $(SRC_DIR)/%.puml,$(OUT_DIR)/%.png,$(PUML_FILES))
SVG_FILES  := $(patsubst $(SRC_DIR)/%.puml,$(OUT_DIR)/%.svg,$(PUML_FILES))

.PHONY: fmt vet test test-race lint security check run test-repository test-repository-integration test-service test-web \
	test-unit test-unit-shuffle test-unit-offline test-processes test-report test-report-html coverage coverage-branch test-artifacts lab1 tools clean-test-artifacts

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

vet:
	go vet ./...

test:
	go test ./...

test-unit:
	go test $(UNIT_PACKAGES)

test-unit-shuffle:
	go test -shuffle=on $(UNIT_PACKAGES)

test-unit-offline:
	GOPROXY=off go test -shuffle=on $(UNIT_PACKAGES)

tools:
	go mod download

test-race:
	go test -race ./...

test-repository:
	go test ./internal/repository/...

test-repository-integration:
	@test -n "$$DATABASE_URL" || (echo "DATABASE_URL is required" >&2; exit 1)
	go test -tags=integration ./internal/repository/...

test-service:
	go test ./internal/service/...

test-web:
	go test ./internal/web ./internal/handler/...

$(ARTIFACT_DIR):
	mkdir -p $(ARTIFACT_DIR)

test-report: | $(ARTIFACT_DIR)
	@status=0; \
	go tool gotestsum --format pkgname-and-test-fails --junitfile $(ARTIFACT_DIR)/unit-tests.xml --jsonfile $(ARTIFACT_DIR)/unit-tests.jsonl -- -count=1 -shuffle=on $(UNIT_PACKAGES) || status=$$?; \
	go run ./cmd/testreport -input $(ARTIFACT_DIR)/unit-tests.jsonl -output $(ARTIFACT_DIR)/test-results.html || exit $$?; \
	exit $$status

test-report-html:
	go run ./cmd/testreport -input $(ARTIFACT_DIR)/unit-tests.jsonl -output $(ARTIFACT_DIR)/test-results.html

test-processes: | $(ARTIFACT_DIR)
	@go test -count=1 -x $(UNIT_PACKAGES) > $(ARTIFACT_DIR)/processes.log 2>&1 || { tail -n 60 $(ARTIFACT_DIR)/processes.log; exit 1; }
	@awk '/\.test -test\./ { name=$$1; sub(/^.*\//, "", name); print name; count++ } END { printf "Test processes: %d\n", count; if (count == 0) exit 1 }' $(ARTIFACT_DIR)/processes.log

coverage: | $(ARTIFACT_DIR)
	go test -shuffle=on -covermode=count -coverpkg=./internal/service,./internal/repository -coverprofile=$(ARTIFACT_DIR)/coverage.out $(CORE_PACKAGES)
	go tool cover -func=$(ARTIFACT_DIR)/coverage.out > $(ARTIFACT_DIR)/coverage.txt
	go tool cover -html=$(ARTIFACT_DIR)/coverage.out -o $(ARTIFACT_DIR)/coverage.html
	@tail -n 1 $(ARTIFACT_DIR)/coverage.txt

coverage-branch: | $(ARTIFACT_DIR)
	go tool gobco -branch ./internal/service > $(ARTIFACT_DIR)/branch-coverage.txt
	go tool gobco -branch ./internal/repository >> $(ARTIFACT_DIR)/branch-coverage.txt
	@cat $(ARTIFACT_DIR)/branch-coverage.txt

test-artifacts: test-report coverage coverage-branch

build-release:
# -s -w убирает таблицу символов и отладочную информацию
	mkdir -p dist
	GOFLAGS= go build -cover=false -trimpath -ldflags='-s -w' \
		-o dist/marketplace-api ./cmd/api

lab1: test-artifacts

clean-test-artifacts:
	rm -rf artifacts/lab1

lint:
	golangci-lint run ./...

security:
	govulncheck ./...

check: fmt vet test

all: png

png: $(OUT_DIR) $(PNG_FILES)

svg: $(OUT_DIR) $(SVG_FILES)

$(OUT_DIR):
	mkdir -p $(OUT_DIR)

$(OUT_DIR)/%.png: $(SRC_DIR)/%.puml | $(OUT_DIR)
	$(PLANTUML) -tpng -o $(abspath $(OUT_DIR)) $<

$(OUT_DIR)/%.svg: $(SRC_DIR)/%.puml | $(OUT_DIR)
	$(PLANTUML) -tsvg -o $(abspath $(OUT_DIR)) $<

clean:
	rm -f $(OUT_DIR)/*.png $(OUT_DIR)/*.svg
