.PHONY: all

# Go parameters
BINARIES=rport rportd

all: test build

build:
	$(foreach BINARY,$(BINARIES),go build -o $(BINARY) -v ./cmd/$(BINARY)/...;)

test:
	go test -v ./...

clean:
	go clean
	rm -f $(BINARIES)

goreleaser-rm-dist:
	goreleaser --rm-dist

goreleaser-snapshot:
	goreleaser --snapshot --rm-dist

docker-goreleaser:
	docker run -it --rm --privileged \
		-v ${PWD}:${PWD} \
		-v $(go env GOCACHE):/root/.cache/go-build \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-w ${PWD} \
		goreleaser/goreleaser:v0.126 --snapshot --rm-dist --skip-publish

docker-golangci-lint:
	docker run -it --rm \
		-v ${PWD}:${PWD} \
		-w ${PWD} \
		golangci/golangci-lint:v1.17 golangci-lint -c .golangci.yml run
