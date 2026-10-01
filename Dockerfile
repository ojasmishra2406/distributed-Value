FROM golang:1.27 AS builder

WORKDIR /app
COPY go.mod ./
# COPY go.sum ./ # Assuming go.sum might not be fully generated yet
RUN go mod download

COPY . .

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

RUN go build -ldflags="-w -s" -o /app/bin/node ./cmd/node
RUN go build -ldflags="-w -s" -o /app/bin/gateway ./cmd/gateway

FROM scratch
COPY --from=builder /app/bin/node /node
COPY --from=builder /app/bin/gateway /gateway

# Expose gRPC ports
EXPOSE 50051 8080

ENTRYPOINT ["/node"]
