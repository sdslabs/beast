FROM nginxinc/nginx-unprivileged:1.28.0-alpine@sha256:c97ff0bf7cbae369953c6da1232ec14ad9f971d66360c5698db0856a4cd657a0

USER root
RUN mkdir -p /beast && chown 101:101 /beast
COPY --chown=101:101 beast.conf /etc/nginx/conf.d/default.conf

USER 101
EXPOSE 8080
