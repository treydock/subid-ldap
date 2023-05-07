FROM golang:1.20.4-alpine3.17 AS builder
RUN apk update && apk add git make
WORKDIR /go/src/app
COPY . ./
ARG VERSION="main"
RUN make build VERSION=${VERSION}

FROM alpine:3.17
RUN apk --no-cache add ca-certificates
WORKDIR /
COPY --from=builder /go/src/app/subid-ldap .
ENTRYPOINT ["/subid-ldap"]
