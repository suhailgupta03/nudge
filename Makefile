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
	docker-compose build;

.PHONY: run-dev-backend
run-dev-backend:
	GOOS=darwin GOARCH=amd64 go build -o nudge_dev cmd/*.go
	docker-compose -f dev/docker-compose.yml up -d
	./nudge_dev --config=dev/config.yml --github.pem=dev/nudge.private-key.pem

.PHONY: build-test-docker
build-test-docker:
	docker-compose -f test/docker-compose.yml build

.PHONY: run-tests
run-tests: build-test-docker
	docker-compose -f test/docker-compose.yml run backend bash -c "go test -v ./... -coverpkg=./... -coverprofile coverage.txt"
