FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.25.3-alpine3.22 AS build-stage

ARG TARGETPLATFORM
ARG BUILDPLATFORM
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /htwr-backend main.go

# FROM alpine:latest AS build-release-stage
FROM alpine:3.22.2
LABEL org.opencontainers.image.source=https://github.com/htwr-aachen/backend


RUN apk update \
    && apk upgrade --no-cache \
    && apk add --no-cache ca-certificates \
    && update-ca-certificates 2>/dev/null || true \
    && rm -rf /var/cache/apk/* \
    && addgroup -g 1001 -S appgroup \
    && adduser -u 1001 -S appuser -G appgroup

RUN mkdir -p /app && \ chown -R appuser:appgroup /app

WORKDIR /app

COPY --from=build-stage --chown=appuser:appgroup /htwr-backend /app/htwr-backend

USER appuser:appgroup

EXPOSE 8080

ENTRYPOINT [ "/app/htwr-backend" ]

