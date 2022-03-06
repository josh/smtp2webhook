# bump: golang /FROM golang:([\d.]+)/ docker:golang|^1
# bump: golang link "Release notes" https://golang.org/doc/devel/release.html
FROM golang:1.17.8-alpine AS builder

WORKDIR /go/src/app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags '-extldflags "-static"' \
  -o /go/bin/smtp2webhook


FROM scratch
COPY --from=builder /go/bin/smtp2webhook /smtp2webhook

ENTRYPOINT [ "/smtp2webhook" ]
HEALTHCHECK CMD [ "/smtp2webhook", "-healthcheck" ]

EXPOSE 25/tcp
EXPOSE 465/tcp
