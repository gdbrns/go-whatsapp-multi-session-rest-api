# Builder Image
# ---------------------------------------------------
FROM mirror.gcr.io/library/golang:1.24-alpine AS go-builder

WORKDIR /usr/src/app

COPY . ./

RUN go mod download \
    && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -a -o main cmd/main/main.go


# Final Image
# ---------------------------------------------------
FROM mirror.gcr.io/library/alpine:latest

ARG SERVICE_NAME="gowam-rest"

ENV PATH $PATH:/usr/app/${SERVICE_NAME}

WORKDIR /usr/app/${SERVICE_NAME}

RUN apk --no-cache --update upgrade \
    && mkdir -p {.bin/webp,dbs} \
    && chmod 775 {.bin/webp,dbs}

COPY --from=go-builder /usr/src/app/.env.example ./.env
COPY --from=go-builder /usr/src/app/main ./gowam-rest

EXPOSE 7001

VOLUME ["/usr/app/${SERVICE_NAME}/dbs"]
CMD ["gowam-rest"]
