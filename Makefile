# The commands in this file are the authority. Anything that disagrees is wrong.

BACKEND  := backend
FRONTEND := frontend

# Read the same .env compose does, so an override lands on both and the migrate
# targets cannot end up pointed at a different database than the one running.
# Leading `-` because there usually is no .env, which is not an error.
-include .env

# Defaults mirror docker-compose.yml, so the migrate targets talk to the database
# the stack actually brought up.
DB_USER      ?= splitter
DB_PASSWORD  ?= splitter
DB_NAME      ?= splitter
DB_HOST_PORT ?= 5433
# Interpolated raw, so a DB_PASSWORD containing URL metacharacters (: / ? @) has
# to be percent-encoded in .env for these targets. The service itself does not
# have this constraint — see Database.DSN, which escapes properly.
DATABASE_URL ?= postgres://$(DB_USER):$(DB_PASSWORD)@localhost:$(DB_HOST_PORT)/$(DB_NAME)?sslmode=disable

MIGRATIONS := $(BACKEND)/migrations
# Run from the module so the CLI is the version go.mod pins; the postgres driver
# is behind a build tag and is not compiled in without it.
MIGRATE := cd $(BACKEND) && go run -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate \
	-path migrations -database "$(DATABASE_URL)"

.DEFAULT_GOAL := help

.PHONY: help
help: ## List the available targets
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk -F':.*?## ' '{printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

# --- running -----------------------------------------------------------------

.PHONY: up
up: require-docker ## Start the database, the API and the frontend together (the one command)
	docker compose up --build

# The one command in the README is the one a reviewer runs first. When it cannot
# run, the reason should be the thing it prints — not a make exit code under
# whatever the docker client happened to say.
.PHONY: require-docker
require-docker:
	@command -v docker >/dev/null 2>&1 || { \
		echo "Docker is not installed, or not on PATH."; \
		echo "This project needs Docker and the compose plugin: https://docs.docker.com/get-docker/"; \
		exit 1; }
	@docker info >/dev/null 2>&1 || { \
		echo "Docker is installed but its daemon is not answering."; \
		echo "Start Docker and wait for it to report ready, then run make up again."; \
		echo "On Windows with WSL: start Docker Desktop, and check Settings -> Resources -> WSL Integration is on for this distro."; \
		exit 1; }

.PHONY: down
down: ## Stop the stack, keeping the database volume
	docker compose down --remove-orphans

.PHONY: clean
clean: ## Stop the stack and delete its volumes and images, back to a clean slate
	docker compose down --remove-orphans --volumes --rmi local

.PHONY: logs
logs: ## Follow the stack's logs
	docker compose logs --follow

.PHONY: ps
ps: ## Show what is running
	docker compose ps

# --- migrations --------------------------------------------------------------

.PHONY: migrate-create
migrate-create: ## Create a timestamped up/down pair: make migrate-create name=add_bills
	@test -n "$(name)" || { echo "usage: make migrate-create name=add_bills"; exit 1; }
	cd $(BACKEND) && go run -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate \
		create -ext sql -dir migrations -seq=false $(name)

.PHONY: migrate-up
migrate-up: ## Apply every pending migration
	$(MIGRATE) up

.PHONY: migrate-down
migrate-down: ## Roll back the most recent migration
	$(MIGRATE) down 1

.PHONY: migrate-version
migrate-version: ## Report the current schema version, and whether it is dirty
	$(MIGRATE) version

.PHONY: migrate-force
migrate-force: ## Clear a dirty state by hand: make migrate-force version=20260916090000
	@test -n "$(version)" || { echo "usage: make migrate-force version=20260916090000"; exit 1; }
	$(MIGRATE) force $(version)

# --- development -------------------------------------------------------------

.PHONY: test
test: test-backend test-frontend ## Run every test suite

.PHONY: test-backend
test-backend: ## Run the Go tests (no Docker required)
	cd $(BACKEND) && go test ./...

.PHONY: test-frontend
test-frontend: ## Run the frontend test (the TypeScript Allocation)
	# Installs first on a fresh clone. Someone checking out this repository and
	# running `make test` should get tests, not a missing binary.
	@test -d $(FRONTEND)/node_modules || (cd $(FRONTEND) && npm ci)
	cd $(FRONTEND) && npm test

.PHONY: mocks
mocks: ## Regenerate the gomock doubles from the interfaces
	cd $(BACKEND) && go generate ./...

.PHONY: typecheck
typecheck: ## Type-check the frontend without building
	cd $(FRONTEND) && npm run typecheck

.PHONY: build-frontend
build-frontend: ## Type-check and build the frontend
	cd $(FRONTEND) && npm run build

.PHONY: fmt
fmt: ## Format the Go code
	cd $(BACKEND) && gofmt -w .

.PHONY: tidy
tidy: ## Tidy the Go module
	cd $(BACKEND) && go mod tidy
