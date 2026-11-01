# Étape 1 : compilation Go
FROM golang:1.26-alpine AS builder

ARG TARGETOS=linux
ARG TARGETARCH

WORKDIR /src

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS} \
    go build -trimpath -ldflags="-s -w" \
    -o /out/harbor-scout ./cmd/harbor-scout

# Étape 2 : image d'exécution
FROM docker:cli
ARG PKG_VERSION=0.2.6
ENV PKGversion=v${PKG_VERSION}
LABEL org.opencontainers.image.version="${PKG_VERSION}"

ARG TARGETARCH
ARG SCOUT_VERSION=1.27.0

RUN apk add --no-cache curl tar \
    && mkdir -p /usr/local/lib/docker/cli-plugins \
    && curl -fsSL \
       "https://github.com/docker/scout-cli/releases/download/v${SCOUT_VERSION}/docker-scout_${SCOUT_VERSION}_linux_${TARGETARCH}.tar.gz" \
       -o /tmp/docker-scout.tar.gz \
    && tar -xzf /tmp/docker-scout.tar.gz -C /usr/local/lib/docker/cli-plugins \
    && chmod +x /usr/local/lib/docker/cli-plugins/docker-scout \
    && rm /tmp/docker-scout.tar.gz

COPY --from=builder /out/harbor-scout /usr/local/bin/harbor-scout

RUN mkdir -p /etc/harbor-scout/docker \
    /etc/harbor-scout/certificates \
    /root/.docker/scout

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/harbor-scout"]
CMD ["-live", "-listen", "0.0.0.0:8080"]
