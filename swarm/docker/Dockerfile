FROM golang:1.11-alpine as builder

ARG VERSION

RUN apk add --update git gcc g++ linux-headers
RUN mkdir -p $GOPATH/src/github.com/gclchaineum && \
    cd $GOPATH/src/github.com/gclchaineum && \
    git clone https://github.com/gclchainsphere/go-gclchaineum && \
    cd $GOPATH/src/github.com/gclchaineum/go-gclchaineum && \
    git checkout ${VERSION} && \
    go install -ldflags "-X main.gitCommit=${VERSION}" ./cmd/swarm && \
    go install -ldflags "-X main.gitCommit=${VERSION}" ./cmd/swarm/swarm-smoke && \
    go install -ldflags "-X main.gitCommit=${VERSION}" ./cmd/swarm/global-store && \
    go install -ldflags "-X main.gitCommit=${VERSION}" ./cmd/ggcl


FROM alpine:3.8 as swarm-smoke
WORKDIR /
COPY --from=builder /go/bin/swarm-smoke /
ADD run-smoke.sh /run-smoke.sh
ENTRYPOINT ["/run-smoke.sh"]

FROM alpine:3.8 as swarm-global-store
WORKDIR /
COPY --from=builder /go/bin/global-store /
ENTRYPOINT ["/global-store"]

FROM alpine:3.8 as swarm
WORKDIR /
COPY --from=builder /go/bin/swarm /go/bin/ggcl /
ADD run.sh /run.sh
ENTRYPOINT ["/run.sh"]
