FROM golang:1.14-alpine as builder

RUN adduser -D -g '' appuser
RUN apk update && apk add git

RUN apk update && apk add --no-cache git ca-certificates tzdata openssh curl && update-ca-certificates
RUN apk add build-base

WORKDIR $GOPATH/src/mypackage/myapp/

COPY go.mod .
COPY go.sum .
ENV GO111MODULE=on
RUN go mod download 

COPY . .
ARG extrabuildargs=""
RUN CGO_ENABLED=1 GOOS=linux go build ${extrabuildargs} -ldflags="-w -s" -a -installsuffix cgo -o /go/bin/app cmd/fanout/main.go
  

FROM scratch as serve

COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /go/bin/app /go/bin/app
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /lib/ld-musl-x86_64.so.1 /lib/

USER appuser
EXPOSE 8080

WORKDIR /go/bin/
CMD ["/go/bin/app"]
