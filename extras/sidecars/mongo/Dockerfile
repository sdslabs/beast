FROM mongo:latest

COPY ./entrypoint.sh /sidecar-entrypoint.sh
COPY ./beast_agent /usr/local/bin/beast_agent
RUN chmod +x /sidecar-entrypoint.sh /usr/local/bin/beast_agent

EXPOSE 9501

ENTRYPOINT ["/sidecar-entrypoint.sh"]
CMD ["mongod"]
