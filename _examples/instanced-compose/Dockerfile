FROM php:7.4-apache

# Install MySQL extension
RUN docker-php-ext-install mysqli pdo pdo_mysql

# Copy challenge files
COPY challenge/ /var/www/html/

# Set permissions
RUN chown -R www-data:www-data /var/www/html

EXPOSE 80
