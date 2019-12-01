PKG=github.com/aokumasan/nifcloud-cloud-controller-manager
IMAGE?=aokumasan/nifcloud-cloud-controller-manager
VERSION=v0.0.1
LDFLAGS?="-X main.version=${VERSION} -s -w"

build:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux go build -ldflags ${LDFLAGS} -o bin/nifcloud-cloud-controller-manager ./cmd/nifcloud-cloud-controller-manager

test:
	go test -cover ./...

image:
	docker build -t $(IMAGE):latest .

push:
	docker push $(IMAGE):latest
