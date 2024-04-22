FROM golang:1.21.3-alpine as builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o tezos-delegation-watcher .

FROM alpine:latest  
RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/tezos-delegation-watcher . 
COPY --from=builder /app/config.yaml . 

EXPOSE 8080 8081

CMD ["./tezos-delegation-watcher"]
