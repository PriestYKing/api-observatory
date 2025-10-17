# syntax=docker/dockerfile:1

####################
# Base Go Builder
####################
FROM golang:1.22-alpine AS go-base

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

####################
# Ingestion Service Builder
####################
FROM go-base AS ingestion-builder

WORKDIR /build

# Copy go mod files for ingestion service only
COPY services/ingestion/go.mod services/ingestion/go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY services/ingestion/ ./

# Build binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -o /app/ingestion-service .

####################
# Analytics Service Builder
####################
FROM go-base AS analytics-builder

WORKDIR /build

# Copy go mod files
COPY services/analytics/go.mod services/analytics/go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY services/analytics/ ./

# Build binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -o /app/analytics-service .

####################
# Cost Tracker Service Builder
####################
FROM go-base AS cost-tracker-builder

WORKDIR /build

# Copy go mod files
COPY services/cost-tracker/go.mod services/cost-tracker/go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY services/cost-tracker/ ./

# Build binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -o /app/cost-tracker-service .

####################
# API Gateway Builder
####################
FROM go-base AS api-gateway-builder

WORKDIR /build

# Copy go mod files
COPY services/api-gateway/go.mod services/api-gateway/go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY services/api-gateway/ ./

# Build binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -o /app/api-gateway .

####################
# Dashboard Dependencies
####################
FROM node:20-alpine AS dashboard-deps

WORKDIR /app

# Copy package files
COPY dashboard/package.json dashboard/package-lock.json* ./

# Install dependencies
RUN npm ci

####################
# Dashboard Builder
####################
FROM node:20-alpine AS dashboard-builder

WORKDIR /app

# Copy dependencies from deps stage
COPY --from=dashboard-deps /app/node_modules ./node_modules

# Copy source code
COPY dashboard/ ./

# Set build environment
ENV NEXT_PUBLIC_API_GATEWAY_URL=http://localhost:8080
ENV NEXT_PUBLIC_WS_URL=ws://localhost:8080/ws
ENV NEXT_TELEMETRY_DISABLED=1

# Build Next.js app
RUN npm run build

####################
# Final Ingestion Service Image
####################
FROM alpine:latest AS ingestion-service

RUN apk --no-cache add ca-certificates tzdata wget

WORKDIR /app

COPY --from=ingestion-builder /app/ingestion-service .

# Create user and group properly for Alpine
RUN addgroup -g 1000 -S appuser && \
    adduser -u 1000 -S appuser -G appuser && \
    chown -R appuser:appuser /app

USER appuser

EXPOSE 50051 8081

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8081/api/health || exit 1

CMD ["./ingestion-service"]

####################
# Final Analytics Service Image
####################
FROM alpine:latest AS analytics-service

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=analytics-builder /app/analytics-service .

# Create user and group properly for Alpine
RUN addgroup -g 1001 -S appuser && \
    adduser -u 1001 -S appuser -G appuser && \
    chown -R appuser:appuser /app

USER appuser

EXPOSE 50052

CMD ["./analytics-service"]

####################
# Final Cost Tracker Service Image
####################
FROM alpine:latest AS cost-tracker-service

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=cost-tracker-builder /app/cost-tracker-service .

# Create user and group properly for Alpine
RUN addgroup -g 1002 -S appuser && \
    adduser -u 1002 -S appuser -G appuser && \
    chown -R appuser:appuser /app

USER appuser

EXPOSE 50053

CMD ["./cost-tracker-service"]

####################
# Final API Gateway Image
####################
FROM alpine:latest AS api-gateway

RUN apk --no-cache add ca-certificates tzdata wget

WORKDIR /app

COPY --from=api-gateway-builder /app/api-gateway .

# Create user and group properly for Alpine
RUN addgroup -g 1003 -S appuser && \
    adduser -u 1003 -S appuser -G appuser && \
    chown -R appuser:appuser /app

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["./api-gateway"]

####################
# Final Dashboard Image
####################
FROM node:20-alpine AS dashboard

WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

# Copy built application
COPY --from=dashboard-builder /app/.next/standalone ./
COPY --from=dashboard-builder /app/.next/static ./.next/static
COPY --from=dashboard-builder /app/public ./public

# Node image already has node user, just use it
RUN chown -R node:node /app

USER node

EXPOSE 3000

ENV PORT=3000
ENV HOSTNAME=0.0.0.0

CMD ["node", "server.js"]
