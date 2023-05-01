BIN := nudge

.PHONY: build
build: $(BIN)

# Build the backend to ./nudge.
$(BIN): $(shell find . -type f -name "*.go")
	go build -o ${BIN} cmd/*.go


# Run the backend in dev mode.
.PHONY: run
run:
	go run cmd/*.go

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


