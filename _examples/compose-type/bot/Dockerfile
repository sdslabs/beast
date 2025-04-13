FROM node:16-alpine3.16

ENV LANG="C.UTF-8" \
    PUPPETEER_SKIP_CHROMIUM_DOWNLOAD=true

RUN apk update && apk add --no-cache \
    zlib-dev \
    udev \
    nss \
    ca-certificates \
    chromium && \
    adduser -h /bot -D -u 10086 bot

WORKDIR /bot

COPY package*.json ./

RUN npm ci --only=production && \
    npm cache clean --force

COPY . .

RUN chown -R bot:bot /bot


EXPOSE 9000

CMD ["node", "index.js"]
