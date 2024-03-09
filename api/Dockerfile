FROM ubuntu:latest
RUN apt update && apt install -y ca-certificates
WORKDIR /app
ADD api /app
EXPOSE 8080
CMD /app/api