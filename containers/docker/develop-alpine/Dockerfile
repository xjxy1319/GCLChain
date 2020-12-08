FROM alpine:3.7

RUN \
  apk add --update go git make gcc musl-dev linux-headers ca-certificates && \
  git clone --depth 1 https://github.com/gclchaineum/go-gclchaineum && \
  (cd go-gclchaineum && make ggcl) && \
  cp go-gclchaineum/build/bin/ggcl /ggcl && \
  apk del go git make gcc musl-dev linux-headers && \
  rm -rf /go-gclchaineum && rm -rf /var/cache/apk/*

EXPOSE 8545
EXPOSE 30303

ENTRYPOINT ["/ggcl"]
