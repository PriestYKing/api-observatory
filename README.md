# API Observatory

Monitor, analyze, and optimize SaaS/API usage with plug-and-play multi-service backend, analytics, and modern dashboard.

## Features

- Real-time analytics (Go microservices + TimescaleDB, Redis)
- Modern dashboard (Next.js + shadcn/ui)
- Traffic simulation/test utilities

## Prerequisites

- Docker & Docker Compose (v2+)
- Node.js (for dashboard dev or package updates)

## Quick Start
- Clone the repo
- git clone https://github.com/YOUR_ORG/api-observatory.git
- cd api-observatory

- Copy and edit env vars
- cp .env.example .env

## Start everything!
- docker compose up -d

## Access:
- Dashboard: http://localhost:3000
- API Gateway: http://localhost:8080


## Custom Usage

- To integrate in your app/project:

- Connect your API or SaaS usage pipeline to the /ingest endpoint
- Extend analytics-service logic to match your domain
- Use dashboard with custom branding (edit `/dashboard`)
