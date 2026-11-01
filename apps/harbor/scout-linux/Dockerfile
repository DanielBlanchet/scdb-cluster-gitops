FROM docker:cli
ENV PKGversion=1.27.1
ARG SCOUT_VERSION=1.27.0

RUN apk add --no-cache curl tar \
    && mkdir -p /usr/local/lib/docker/cli-plugins \
    && curl -fsSL \
       "https://github.com/docker/scout-cli/releases/download/v${SCOUT_VERSION}/docker-scout_${SCOUT_VERSION}_linux_arm64.tar.gz" \
       -o /tmp/docker-scout.tar.gz \
    && tar -xzf /tmp/docker-scout.tar.gz -C /usr/local/lib/docker/cli-plugins \
    && chmod +x /usr/local/lib/docker/cli-plugins/docker-scout \
    && rm /tmp/docker-scout.tar.gz
