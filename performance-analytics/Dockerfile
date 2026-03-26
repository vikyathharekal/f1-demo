FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o service .

FROM alpine:3.19
COPY --from=builder /app/service /service
EXPOSE 7003
ENV PITSTOP_SERVICE_URL=http://pitstop-duration-optimizer:7001
ENV DRIVER_SERVICE_URL=http://driver-analytics:7002
CMD ["/service"]
