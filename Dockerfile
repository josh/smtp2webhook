FROM golang:1.18.3-alpine AS builder

WORKDIR /go/src/app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -buildvcs=false \
  -ldflags '-extldflags "-static"' \
  -o /go/bin/smtp2webhook


FROM scratch
COPY --from=builder /go/bin/smtp2webhook /smtp2webhook

ENTRYPOINT [ "/smtp2webhook" ]
HEALTHCHECK CMD [ "/smtp2webhook", "-healthcheck" ]

EXPOSE 25/tcp
EXPOSE 465/tcp
