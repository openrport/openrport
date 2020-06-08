.PHONY: all

# Go parameters
BINARY=rport

all: test build

build:
	go build -o $(BINARY) -v ./cmd/$(BINARY)/...

test:
	go test -v ./...

clean:
	go clean
	rm -f $(BINARY)

docker-goreleaser:
	docker run -it --rm --privileged \
		-v ${PWD}:${PWD} \
		-v $(go env GOCACHE):/root/.cache/go-build \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-w ${PWD} \
		goreleaser/goreleaser:v0.135 --snapshot --rm-dist --skip-publish


docker-golangci-lint:
	docker run -it --rm \
		-v ${PWD}:${PWD} \
		-w ${PWD} \
		golangci/golangci-lint:v1.17 golangci-lint -c .golangci.yml run
