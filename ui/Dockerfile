# https://github.com/patrickdappollonio/http-server
#
# This is a simple file server serving up the files for the React app
FROM ghcr.io/patrickdappollonio/docker-http-server:latest
WORKDIR /app
ADD build /app
EXPOSE 8080
ENV FILE_SERVER_PATH=/app
ENV FILE_SERVER_PORT=8080