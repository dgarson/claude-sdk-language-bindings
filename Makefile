.PHONY: help venv init python-deps lint lint-fix format typecheck \
	test test-python test-python-unit test-python-e2e \
	test-go test-go-client test-go-e2e test-go-e2e-sessions test-go-e2e-keychain \
	protos \
	sidecar-metrics \
	clean

SDK_DIR := claude-agent-sdk-python
GO_DIR := $(SDK_DIR)/go
VENV_DIR ?= $(SDK_DIR)/.venv
VENV_ABS := $(abspath $(VENV_DIR))
VENV_BIN := $(VENV_ABS)/bin

PYTHON_BOOTSTRAP ?= python3
PYTHON ?= $(VENV_BIN)/python
PIP ?= $(PYTHON) -m pip
GO ?= go
PYTEST ?= $(PYTHON) -m pytest

DEV_DEPS_SENTINEL := $(VENV_ABS)/.dev-deps-installed

help:
	@echo "Targets:"
	@echo "  venv                   Create the Python venv ($(VENV_DIR))"
	@echo "  init                   Install Python dev dependencies"
	@echo "  lint                   Run ruff lint (no fixes)"
	@echo "  lint-fix               Run ruff lint with auto-fix"
	@echo "  format                 Run ruff formatter"
	@echo "  typecheck              Run mypy on src/"
	@echo "  test                   Run python unit tests and go tests"
	@echo "  test-python            Run all python tests"
	@echo "  test-python-unit       Run python unit tests (tests/)"
	@echo "  test-python-e2e        Run python E2E tests (requires ANTHROPIC_API_KEY)"
	@echo "  test-go                Run all go tests"
	@echo "  test-go-client         Run go client tests"
	@echo "  test-go-e2e            Run go sidecar E2E test (SIDECAR_E2E=1)"
	@echo "  test-go-e2e-sessions   Run go session E2E subset (SIDECAR_E2E=1)"
	@echo "  test-go-e2e-keychain   Run go E2E against real Claude Code via macOS Keychain"
	@echo "  protos                 Regenerate Go/Python protobuf stubs"
	@echo "  sidecar-metrics        Scrape and print sidecar Prometheus metrics (via OTEL Collector)"
	@echo "  clean                  Remove python caches"

venv: $(PYTHON)

$(PYTHON):
	$(PYTHON_BOOTSTRAP) -m venv $(VENV_DIR)

python-deps: $(DEV_DEPS_SENTINEL)

$(DEV_DEPS_SENTINEL): $(SDK_DIR)/pyproject.toml | venv
	cd $(SDK_DIR) && $(PIP) install -e ".[dev]"
	touch $@

init: python-deps

lint: python-deps
	cd $(SDK_DIR) && $(PYTHON) -m ruff check src/ tests/ e2e-tests/

lint-fix: python-deps
	cd $(SDK_DIR) && $(PYTHON) -m ruff check src/ tests/ e2e-tests/ --fix

format: python-deps
	cd $(SDK_DIR) && $(PYTHON) -m ruff format src/ tests/ e2e-tests/

typecheck: python-deps
	cd $(SDK_DIR) && $(PYTHON) -m mypy src/

test: test-python-unit test-go

test-python: python-deps
	cd $(SDK_DIR) && $(PYTEST) tests/ e2e-tests/ -v

test-python-unit: python-deps
	cd $(SDK_DIR) && $(PYTEST) tests/ -v

test-python-e2e: python-deps
	cd $(SDK_DIR) && $(PYTEST) e2e-tests/ -v -m e2e

test-go:
	$(GO) -C $(GO_DIR) test ./... -v

test-go-client:
	$(GO) -C $(GO_DIR) test ./client -v

test-go-e2e:
	SIDECAR_E2E=1 $(GO) -C $(GO_DIR) test ./client -run TestSidecarE2E -v

test-go-e2e-sessions:
	SIDECAR_E2E=1 $(GO) -C $(GO_DIR) test ./client -run 'Test(CreateSessionReturnsIDs|ListSessionsIncludesActiveSession|GetSessionReturnsSummary)$$' -v

test-go-e2e-keychain:
	SIDECAR_E2E=1 SIDECAR_E2E_LIVE=1 SIDECAR_KEYCHAIN_ENABLE=1 \
	$(SDK_DIR)/scripts/sidecar-run.sh -- go -C $(GO_DIR) test ./client -run TestSidecarE2E -v

protos:
	$(SDK_DIR)/scripts/gen_protos.sh

sidecar-metrics: venv
	$(PYTHON) $(SDK_DIR)/scripts/sidecar_metrics_dump.py $(METRICS_ARGS)

clean:
	cd $(SDK_DIR) && find . -type d -name "__pycache__" -prune -exec rm -rf {} +
