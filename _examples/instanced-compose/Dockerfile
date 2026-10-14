FROM php:7.4-apache

RUN apt-get update && apt-get install -y --no-install-recommends openssh-server \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd -g 1337 beast-grp \
    && useradd -u 1337 -g 1337 -ms /bin/bash beast

RUN docker-php-ext-install mysqli pdo pdo_mysql

COPY challenge/ /var/www/html/
COPY check.sh /challenge/check.sh

RUN chown -R www-data:www-data /var/www/html \
    && chmod +x /challenge/check.sh

EXPOSE 80 22
