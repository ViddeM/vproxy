FROM golang:alpine

WORKDIR /app

COPY . .

RUN go build -o main .

EXPOSE 5555

ENV PROXY_ADDRESS="http://localhost:3000"

CMD ["./main"]
