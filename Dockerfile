FROM ubuntu:latest
WORKDIR /app
ADD website /app
EXPOSE 8080
ARG POSTGRES
CMD ["/app/website", "$POSTGRES"]