# Main image
FROM richarvey/nginx-php-fpm
COPY . /var/www/html
RUN ["rm", "./index.php"]
RUN ["rm", "../errors/404.html"]
