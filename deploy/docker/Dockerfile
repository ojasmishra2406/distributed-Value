FROM golang:1.23.4-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/node ./cmd/node

FROM scratch
COPY --from=builder /app/node /node
ENTRYPOINT ["/node"]
