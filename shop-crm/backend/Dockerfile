# Build stage: compile a static binary with the migrations embedded.
FROM golang:1.23-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# Run stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 api
USER api

COPY --from=build /out/api /usr/local/bin/api

EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]