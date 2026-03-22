FROM golang:1.23 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -tags netgo \
    -o /beast \
    github.com/sdslabs/beastv4/cmd/beast

FROM ubuntu:22.04

RUN apt-get update && apt-get install -y \
    ca-certificates \
    gcc \
    util-linux \
    docker.io \
    && rm -rf /var/lib/apt/lists/*

COPY scripts/importenv.c /tmp/importenv.c
RUN gcc -o /usr/bin/importenv /tmp/importenv.c && rm /tmp/importenv.c

COPY scripts/docker-enter /usr/bin/docker-enter
COPY scripts/docker_enter /usr/bin/docker_enter
RUN chmod +x /usr/bin/docker-enter && chmod u+s /usr/bin/docker_enter

# Copy beast binary
COPY --from=builder /beast /usr/local/bin/beast

# setup.sh handles both local dev and container startup
COPY setup.sh /usr/local/bin/setup.sh
RUN chmod +x /usr/local/bin/setup.sh

EXPOSE 5005

ENTRYPOINT ["setup.sh"]
