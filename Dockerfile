FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o dhcpmon cmd/dhcpmon/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates dnsmasq nmap
WORKDIR /root/

COPY --from=builder /app/dhcpmon .
COPY --from=builder /app/html ./html/
COPY --from=builder /app/macaddress.io-db.json .

EXPOSE 8067

CMD ["./dhcpmon"]

