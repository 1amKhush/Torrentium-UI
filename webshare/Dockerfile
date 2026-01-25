# Build stage for Go backend
FROM golang:1.23-alpine AS go-builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o server .

# Build stage for frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ .
RUN npm run build

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates sqlite-libs

WORKDIR /app

# Copy built artifacts
COPY --from=go-builder /app/server .
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

# Create data directory
RUN mkdir -p /app/data

# Environment variables
ENV PORT=8080
ENV WEBSHARE_DB=/app/data/webshare.db
ENV GIN_MODE=release

EXPOSE 8080

CMD ["./server"]
