# Stage 1: Build web UI
FROM node:22-alpine AS web
WORKDIR /app/web
COPY web/ .
RUN npm ci
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /app/web/dist/ pkg/server/static/dist/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /pfctl ./cmd/pfctl

# Stage 3: Runtime
FROM gcr.io/distroless/static-debian12
COPY --from=build /pfctl /pfctl
EXPOSE 8080
ENV LOG_FORMAT=json
ENTRYPOINT ["/pfctl"]
