FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o ingress-gateway ./cmd/ingress/main.go

FROM alpine:3.19

WORKDIR /root/

COPY --from=builder /app/ingress-gateway .

COPY config.yaml .

EXPOSE 8080 9090

CMD [ "./ingress-gateway" ]