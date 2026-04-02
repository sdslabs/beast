FROM golang:1.23 AS builder

RUN apt-get update && apt-get install -y \
    make git gcc util-linux \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN cp scripts/docker-enter /usr/bin/docker-enter && \
    cp scripts/docker_enter /usr/bin/docker_enter && \
    chmod u+s /usr/bin/docker_enter && \
    gcc -o /usr/bin/importenv scripts/importenv.c

RUN make build

FROM ubuntu:22.04

RUN apt-get update && apt-get install -y ca-certificates curl gnupg lsb-release \
    && rm -rf /var/lib/apt/lists/*

RUN install -m 0755 -d /etc/apt/keyrings && \
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg && \
    chmod a+r /etc/apt/keyrings/docker.gpg && \
    echo \
      "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
      $(. /etc/os-release && echo $VERSION_CODENAME) stable" \
      > /etc/apt/sources.list.d/docker.list && \
    apt-get update && \
    apt-get install -y \
      docker-ce-cli \
      docker-compose-plugin \
      docker-buildx-plugin && \
    apt-get clean

COPY --from=builder /go/bin/beast /usr/local/bin/beast
COPY setup.sh /usr/local/bin/setup.sh
RUN chmod +x /usr/local/bin/setup.sh

EXPOSE 5005

ENTRYPOINT ["setup.sh"]
