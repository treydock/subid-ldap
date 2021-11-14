FROM golang:1.17-alpine AS builder
RUN apk update && apk add git make
WORKDIR /go/src/app
COPY . ./
ARG VERSION="main"
RUN make build VERSION=${VERSION}

FROM alpine:3.12
RUN apk --no-cache add ca-certificates
WORKDIR /
COPY --from=builder /go/src/app/subid-ldap .
ENTRYPOINT ["/subid-ldap"]
