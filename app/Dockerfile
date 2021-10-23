FROM golang:1.16

EXPOSE 8001:8001
WORKDIR /go/src/app
COPY . .

RUN go mod download
RUN go build -o /message-broker
CMD ["/message-broker"]