FROM alpine:latest

WORKDIR /app
COPY . . 

CMD ["./main", "--webhook", "https://webhook.site/7910a4a5-4625-4664-bbd2-71de6165b3f7"]