
GO_FILES := $(shell find . -type f -name '*.go' -not -path "./vendor/*")

.PHONY: test
test: $(GO_FILES)
	go test -v ./...


.PHONY: docker-build
docker-build: test $(GO_FILES)
	GOOS=linux GOARCH=amd64 go build -o bin/eventsourcing cmd/serverd/main.go


.PHONY: docker
docker: docker-build $(GO_FILES)
	docker build -t venkssa/eventsourcing:0.1 .


.PHONY: kubernetes-dev
kubernetes-dev: docker-build

