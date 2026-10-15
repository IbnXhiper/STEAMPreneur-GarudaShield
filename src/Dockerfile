# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/gs ./cmd/garudashield

FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -H gs
COPY --from=build /out/gs /usr/local/bin/gs
USER gs
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://127.0.0.1:8080/health || exit 1
CMD ["gs"]
