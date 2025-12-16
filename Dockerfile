# Builder Image
# ---------------------------------------------------
FROM golang:1.24-alpine AS go-builder

WORKDIR /usr/src/app

# Install build dependencies
RUN apk add --no-cache --no-scripts git ca-certificates tzdata

# Download dependencies first for better layer caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . ./

# Build with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -extldflags '-static'" \
    -trimpath \
    -a -o main cmd/main/main.go


# Final Image
# ---------------------------------------------------
FROM alpine:3.21

ARG SERVICE_NAME="gowam-rest"

# Create non-root user for security
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

ENV PATH=$PATH:/usr/app/${SERVICE_NAME}
ENV TZ=UTC

WORKDIR /usr/app/${SERVICE_NAME}

RUN apk --no-cache --no-scripts add ca-certificates tzdata wget && \
    mkdir -p dbs && \
    chown -R appuser:appgroup /usr/app/${SERVICE_NAME}

COPY --from=go-builder --chown=appuser:appgroup /usr/src/app/.env.example ./.env
COPY --from=go-builder --chown=appuser:appgroup /usr/src/app/main ./gowam-rest
COPY --from=go-builder --chown=appuser:appgroup /usr/src/app/docs ./docs

# Switch to non-root user
USER appuser

EXPOSE 7001

VOLUME ["/usr/app/${SERVICE_NAME}/dbs"]

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:7001/ || exit 1

CMD ["gowam-rest"]
