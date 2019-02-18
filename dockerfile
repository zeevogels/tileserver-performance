FROM golang:alpine
RUN go build -o /go/bin/performance
ENTRYPOINT ["go/bin/performance"]