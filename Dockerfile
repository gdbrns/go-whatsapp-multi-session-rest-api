# Builder Image
# ---------------------------------------------------
FROM golang:1.23-alpine AS go-builder

WORKDIR /usr/src/app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -a -o main cmd/main/main.go


# Final Image
# ---------------------------------------------------
FROM alpine:latest

ARG SERVICE_NAME="gowam-rest"

ENV PATH=$PATH:/usr/app/${SERVICE_NAME}
ENV TZ=UTC

WORKDIR /usr/app/${SERVICE_NAME}

RUN apk --no-cache add ca-certificates tzdata \
    && mkdir -p dbs \
    && chmod 775 dbs

COPY --from=go-builder /usr/src/app/.env.example ./.env
COPY --from=go-builder /usr/src/app/main ./gowam-rest
COPY --from=go-builder /usr/src/app/docs ./docs

EXPOSE 7001

VOLUME ["/usr/app/${SERVICE_NAME}/dbs"]

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:7001/ || exit 1

CMD ["gowam-rest"]
