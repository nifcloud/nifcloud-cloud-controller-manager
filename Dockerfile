FROM golang:1.20.7-alpine as builder

WORKDIR /go/src/github.com/aokumasan/nifcloud-cloud-controller-manager
RUN apk add --no-cache make git
ADD . .
RUN make build

# -----

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /go/src/github.com/aokumasan/nifcloud-cloud-controller-manager/bin/nifcloud-cloud-controller-manager /bin/nifcloud-cloud-controller-manager
ENTRYPOINT ["/bin/nifcloud-cloud-controller-manager"]
