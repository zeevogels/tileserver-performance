FROM golang:1.11
ADD . /go/src/tileserver-performance
WORKDIR /go/src/tileserver-performance
RUN go get tileserver-performance
RUN go install
ENTRYPOINT ["go/bin/tileserver-performance"]
