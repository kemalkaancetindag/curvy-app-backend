FROM golang:latest

WORKDIR /usr/src/app

COPY . .
RUN go mod tidy

EXPOSE 8080

CMD ["cd usr && cd src && cd app"]
CMD ["go run main.go"]

