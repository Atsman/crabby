FROM golang:latest as builder
WORKDIR /go/src/github.com/atsman/crabby/
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o crabby .

FROM alpine:latest  
RUN apk --no-cache add ca-certificates
VOLUME ["/config"]
WORKDIR /root/
COPY --from=builder /go/src/github.com/atsman/crabby .
CMD ./crabby -config=${CRABBY_CONFIG}
