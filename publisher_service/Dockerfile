FROM golang:1.16
EXPOSE 8002:8002
WORKDIR /go/src/app
COPY . .
WORKDIR /go/src/app/publisher
RUN go mod download
RUN go build -o /publisher-service
CMD ["/publisher-service"]