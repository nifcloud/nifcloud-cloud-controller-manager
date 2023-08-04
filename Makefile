PKG:=github.com/aokumasan/nifcloud-cloud-controller-manager
IMAGE:=aokumasan/nifcloud-cloud-controller-manager
VERSION:=$(shell git describe --tags --dirty --match="v*")
LDFLAGS:="-X main.version=$(VERSION) -X k8s.io/component-base/version.gitVersion=$(VERSION) -s -w"

build:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux go build -ldflags $(LDFLAGS) -o bin/nifcloud-cloud-controller-manager ./cmd/nifcloud-cloud-controller-manager

test:
	go test -cover ./...

image:
	docker build -t $(IMAGE):$(VERSION) .

push:
	docker push $(IMAGE):$(VERSION)
