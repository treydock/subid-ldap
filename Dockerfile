FROM golang:1.26.4-alpine3.24 AS builder
RUN apk update && apk add git make
WORKDIR /go/src/app
COPY . ./
ARG VERSION="main"
RUN make build VERSION=${VERSION}

FROM alpine:3.24
RUN apk --no-cache add ca-certificates
WORKDIR /
COPY --from=builder /go/src/app/subid-ldap .
ENTRYPOINT ["/subid-ldap"]
