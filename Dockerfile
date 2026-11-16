# syntax=docker/dockerfile:1
FROM golang:1.27-alpine AS build

RUN apk add --no-cache ca-certificates

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/api ./cmd/api

FROM scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/api /api

USER 65532:65532

EXPOSE 8080

ENTRYPOINT ["/api"]
