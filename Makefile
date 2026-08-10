# Dynamic version from git: VERSION tracks the nearest upstream tag
# (git describe), e.g. v7.2.127-6-g1de01329, so builds always report the
# upstream release they are based on. Override with VERSION=... if needed.
VERSION ?= $(shell git describe --tags --always)
COMMIT ?= $(shell git rev-parse --short HEAD)
BUILD_DATE ?= $(shell date +%Y-%m-%d)

build:
	VERSION=$(VERSION) COMMIT=$(COMMIT) BUILD_DATE=$(BUILD_DATE) docker compose build

up:
	VERSION=$(VERSION) COMMIT=$(COMMIT) BUILD_DATE=$(BUILD_DATE) docker compose up -d --build

# One-shot: fetch upstream -> merge -> build & deploy -> push to origin.
# Stops on merge conflict (resolve manually, then re-run).
update:
	git fetch upstream
	git merge upstream/main --no-edit
	$(MAKE) up
	git push origin HEAD
