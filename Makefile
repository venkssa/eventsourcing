
GO_FILES := $(shell find . -type f -name '*.go' -not -path "./vendor/*")

.PHONY: test go-build docker-build


test: $(GO_FILES)
	go test -v ./...


go-build: test $(GO_FILES)
	GOOS=linux GOARCH=amd64 go build -o bin/eventsourcing cmd/serverd/main.go


docker-build: go-build $(GO_FILES)
	docker build -t venkssa/eventsourcing:0.1.1 .

kubernetes-dev: docker-build

