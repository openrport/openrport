.PHONY: all

# Go parameters
BINARIES=rport rportd

all: test build lint

build:
	CGO_ENABLED=0 $(foreach BINARY,$(BINARIES),go build -ldflags "-s -w" -o $(BINARY) -v ./cmd/$(BINARY);)

build-debug:
	$(foreach BINARY,$(BINARIES),go build -race -gcflags "all=-N -l" -o $(BINARY) -v ./cmd/$(BINARY);)

test:
	go test -race -v ./...

bind-data:
	cd db/migration/jobs/sql/ && go-bindata -o ../bindata.go -pkg jobs ./...
	cd db/migration/clients/sql/ && go-bindata -o ../bindata.go -pkg clients ./...
	cd db/migration/client_groups/sql/ && go-bindata -o ../bindata.go -pkg client_groups ./...
	cd db/migration/vaults/sql/ && go-bindata -o ../bindata.go -pkg vaults ./...
	cd db/migration/library/sql/ && go-bindata -o ../bindata.go -pkg library ./...
	cd db/migration/auditlog/sql/ && go-bindata -o ../bindata.go -pkg auditlog ./...
	cd db/migration/monitoring/sql/ && go-bindata -o ../bindata.go -pkg monitoring ./...
	cd db/migration/api_sessions/sql/ && go-bindata -o ../bindata.go -pkg api_sessions ./...
	cd db/migration/api_token/sql/ && go-bindata -o ../bindata.go -pkg api_token ./...
	cd server/notifications/repository/sqlite/migrations/ && go-bindata -o ../bindata.go -pkg sqlite ./...

# usage: make bindata-db DB=monitoring, if you want to generate embedded file for monitoring.db migration
bindata-db:
	cd db/migration/$(DB)/sql/ && go-bindata -o ../bindata.go -pkg $(DB) ./...

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
		-v $(shell go env GOCACHE):/cache/go \
		-e GOCACHE=/cache/go \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-w ${PWD} \
		goreleaser/goreleaser:latest --snapshot --rm-dist --skip-publish

docker-golangci-lint:
	docker run -it --rm \
		-v ${PWD}:${PWD} \
		-w ${PWD} \
		golangci/golangci-lint:v1.45 golangci-lint -c .golangci.yml run

fmt:
	goimports -w .
	gofmt -w .

lint:
	golangci-lint run
