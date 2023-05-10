BIN := nudge

.PHONY: build
build: $(BIN)

# Build the backend to ./nudge.
$(BIN): $(shell find . -type f -name "*.go")
	go build -o ${BIN} cmd/*.go


# Run the backend in dev mode.
.PHONY: run
run:
	go run cmd/*.go --config=dev/config.yml

# Use goreleaser to do a dry run producing local builds.
.PHONY: release-dry
release-dry:
	goreleaser --parallelism 1 --clean --snapshot

# Use goreleaser to build production releases and publish them.
.PHONY: release
release:
	goreleaser --parallelism 1 --clean

.PHONY:docker
docker: build ## Build docker container for nudge
	docker-compose build; \

.PHONY: run-docker
run-docker: docker  ## Build and spawns docker container
	docker-compose up -d; \

.PHONY: rm-docker
rm-docker: build ## Delete the docker container including any DB volumes.
	docker-compose down -v; \


# Build local docker images for development.
.PHONY: build-dev-docker
build-dev-docker: ## Build docker containers for the entire suite (Front/Core/PG).
	cd dev; \
	docker-compose build ; \

# Spin a local docker suite for local development.
.PHONY: dev-docker
dev-docker: build-dev-docker ## Build and spawns docker containers for the entire suite
	cd dev; \
	docker-compose up

.PHONY: run-dev-backend
run-dev-backend:
	go run cmd/*.go --config=dev/config.yml


.PHONY: build-test-docker
build-test-docker:
	docker-compose -f test/docker-compose.yml build

.PHONY: run-tests
run-tests: build-test-docker
	docker-compose -f test/docker-compose.yml run backend bash -c "go test -v ./... -coverpkg=./... -coverprofile coverage.txt"
