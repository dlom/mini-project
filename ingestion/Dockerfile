# Build
FROM composer:latest as builder
WORKDIR /php
COPY . .
RUN composer install

# Main image
FROM php:fpm-alpine
COPY --from=builder /php/ /var/www
