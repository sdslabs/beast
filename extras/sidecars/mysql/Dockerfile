FROM mysql:8.0.13

COPY ./entrypoint.sh /sidecar-entrypoint.sh
COPY ./beast_agent /usr/local/bin/beast_agent
RUN chmod +x /sidecar-entrypoint.sh /usr/local/bin/beast_agent

EXPOSE 9500

ENTRYPOINT ["/sidecar-entrypoint.sh"]
CMD ["mysqld", "--character-set-server=utf8mb4", "--collation-server=utf8mb4_unicode_ci", "--default-authentication-plugin=mysql_native_password"]
