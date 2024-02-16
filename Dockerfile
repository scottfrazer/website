FROM ubuntu:latest
WORKDIR /app
ADD website /app
EXPOSE 8080
CMD ["/app/website", "$POSTGRES"]