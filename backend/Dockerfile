FROM golang:1.21 as builder
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/. ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/api ./cmd/api

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=builder /out/api /app/api
COPY backend/deploy/workflows /app/deploy/workflows
ENV API_HTTP_PORT=:8080
EXPOSE 8080
ENTRYPOINT ["/app/api"]
