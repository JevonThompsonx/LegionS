# Makefile — Add-Non-Steam-Games-Linux
#
# Common developer tasks. Run `make help` for the list.

PYTHON          ?= python3
PYTEST          ?= $(PYTHON) -m pytest
SHELLCHECK      ?= shellcheck
SHELL_SOURCES   ?= run.sh
TEST_DIR        ?= tests
VENV            ?= .venv

.PHONY: help test lint clean install

help:
	@echo "Available targets:"
	@echo "  test     Run the pytest suite"
	@echo "  lint     Run shellcheck against shell scripts"
	@echo "  clean    Remove caches, build artifacts and the local venv"
	@echo "  install  Create a venv and install requirements"

test:
	$(PYTEST) -q $(TEST_DIR)

lint:
	$(SHELLCHECK) --shell=bash --external-sources $(SHELL_SOURCES)

install:
	$(PYTHON) -m venv $(VENV)
	$(VENV)/bin/python -m pip install --upgrade pip
	$(VENV)/bin/python -m pip install -r requirements.txt

clean:
	rm -rf $(VENV) .pytest_cache __pycache__ tests/__pycache__
	find . -type d -name '__pycache__' -prune -exec rm -rf {} +
	find . -type f -name '*.pyc' -delete
	rm -f .coverage coverage.xml
