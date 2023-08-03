FROM golang:1.20.7-alpine as builder

WORKDIR /go/src/github.com/aokumasan/nifcloud-cloud-controller-manager
RUN apk add --no-cache make git
ADD . .
RUN make build

# -----

FROM alpine:3.10.3

RUN apk add --no-cache open-vm-tools
COPY --from=builder /go/src/github.com/aokumasan/nifcloud-cloud-controller-manager/bin/nifcloud-cloud-controller-manager /bin/nifcloud-cloud-controller-manager
ENTRYPOINT ["/bin/nifcloud-cloud-controller-manager"]
