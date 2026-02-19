FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /app ./cmd/app

FROM alpine:3.21

RUN apk add --no-cache ca-certificates

COPY --from=builder /app /app
COPY migrations /migrations

EXPOSE 8080

ENTRYPOINT ["/app"]
