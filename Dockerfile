# cxo build binaries
# reference https://github.com/skycoin/cxo
FROM golang:1.9-alpine AS build-go

RUN apk --no-cache add git && \
  go get -u github.com/golang/dep/cmd/dep

COPY . $GOPATH/src/github.com/skycoin/cxo

RUN cd $GOPATH/src/github.com/skycoin/cxo && \
  dep ensure && \
  CGO_ENABLED=0 GOOS=linux go install -a -ldflags "-s" -installsuffix cgo ./...


# cxo image
FROM alpine:3.7

RUN adduser -D cxo

USER cxo

COPY --from=build-go /go/bin/cxod /usr/bin/
COPY --from=build-go /go/bin/cxocli /usr/bin/

EXPOSE 8870 8871

CMD ["cxod", "--tcp=0.0.0.0:8870", "--rpc=0.0.0.0:8871"]
