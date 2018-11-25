FROM nginx:latest

LABEL version="0.1"
LABEL author="fristonio"

RUN apt-get update \
	&& apt-get install -y -q --no-install-recommends ca-certificates less \
	&& apt-get clean \
	&& rm -r /var/lib/apt/lists/*

RUN sed -i 's/^http {/&\n    server_names_hash_bucket_size 128;/g' /etc/nginx/nginx.conf
RUN chown nginx:nginx /var/log/nginx/

COPY beast.conf /etc/nginx/conf.d/default.conf
ADD docker-entry.sh /docker-entry.sh
RUN chmod +x docker-entry.sh

VOLUME ["/beast"]
EXPOSE 80

# Add tini
ENV TINI_VERSION v0.18.0
ADD https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini /tini
RUN chmod +x /tini
ENTRYPOINT ["/tini", "--"]

CMD ["/docker-entry.sh"]
