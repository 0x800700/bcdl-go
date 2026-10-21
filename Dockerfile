# Stage 1: Build Frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Backend
FROM golang:alpine AS backend-builder
WORKDIR /app
ENV GOTOOLCHAIN=auto
RUN apk add --no-cache ca-certificates git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bcdl-server main.go

# Stage 3: Minimal runtime
FROM alpine:3.21
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
COPY --from=backend-builder /app/bcdl-server /app/bcdl-server
COPY --from=frontend-builder /app/frontend/dist /app/frontend/dist

ENV PORT=8080
ENV DOWNLOAD_DIR=/downloads

VOLUME ["/downloads"]
EXPOSE 8080

ENTRYPOINT ["/app/bcdl-server"]
