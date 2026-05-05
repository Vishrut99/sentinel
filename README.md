# Sentinel — Incident Ticketing System

Sentinel is a full-stack IT service management (ITSM) platform for handling incident tickets with AI-assisted triage, automated assignment, and SLA enforcement.

## Features

- **AI triage** — auto-suggests priority, category, and required skills
- **Auto-assignment** — skill-based routing with workload balancing
- **SLA tracking** — response/resolve timers and breach detection
- **Problem management** — link incidents to a root-cause problem ticket
- **Role-based access** — requester, agent, and admin workflows
- **Audit trail** — every state change logged
- **Dashboards** — operational metrics and live queues

## Tech Stack

| Layer | Tech |
|-------|------|
| Frontend | Next.js 16, React 19, Tailwind CSS 4, shadcn/ui, Recharts |
| Backend | Go, Chi router, GORM, PostgreSQL, JWT |
| AI | Google Gemini (triage & suggestions) |

## Quick Start (Recommended)

Run the full backend + database with one command:

```bash
docker-compose up --build
```

API will be available at:
`http://localhost:8080/api/v1`

## System Flow

Client (Next.js)
	↓
Backend API (Go + Chi)
	↓
PostgreSQL (Docker volume)

## Architecture

- **Frontend**: `client/` — Next.js app for agent/admin/requester workflows
- **Backend**: `server/` — Go REST API (Chi + GORM)
- **Database**: PostgreSQL — schema migrations in `server/internal/db/migrations/`

## Deployment

- Frontend: Vercel (recommended)
- Backend + DB: Docker 

This project is fully containerized and can be deployed using docker-compose on any cloud VM.

## Key Highlights

- Fully Dockerized backend + database
- Automatic SQL migrations on startup
- Persistent PostgreSQL storage using Docker volumes
- Clean modular backend architecture (services, repositories, handlers)
- Production-ready environment configuration

## Prerequisites

- **Node.js** 20+ and npm (for frontend)
- **Go** 1.22+ (for backend)
- **Docker** (for full-stack with PostgreSQL)

## Environment Setup

### Backend `.env` Setup

Copy the example env file and fill in your values:

```bash
cp server/.env.example server/.env
```

Then edit `server/.env` and set `DATABASE_URL`, `JWT_SECRET`, and other required values.  
See `server/.env.example` for reference.

### Frontend `.env.local` Setup

Copy the example env file:

```bash
cp client/.env.example client/.env.local
```

Then edit `client/.env.local` and set `NEXT_PUBLIC_API_BASE_URL` (default: `http://localhost:8080/api/v1`).  
See `client/.env.example` for reference.

## How to Run (Docker)

This starts **PostgreSQL + API** using `docker-compose.yml`:

```bash
docker-compose up --build
```

Backend API will be available at `http://localhost:8080/api/v1`.

## How to Run (Local)

### 1) Start Postgres

If you are not using Docker for the backend, ensure Postgres is running locally and the `DATABASE_URL` in `server/.env` points to it.

### 2) Start Backend

```bash
cd server
go run ./cmd/api
```

### 3) Start Frontend

```bash
cd client
npm install
npm run dev
```

## API Base URL

- Backend default: `http://localhost:8080/api/v1`
- Frontend env: set `NEXT_PUBLIC_API_BASE_URL` to the API base URL

## Example Workflow

1. **Register** a requester or agent account
2. **Login** to receive a JWT token
3. **Create a ticket** with title, description, category, and priority
4. **Track status** and SLA updates on the dashboard

