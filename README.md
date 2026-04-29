# Sentinel — Incident Ticketing System

Full-stack IT service management (ITSM) platform with AI-powered triage, skill-based agent assignment, SLA enforcement, and problem management.

## Stack

| Layer | Tech |
|-------|------|
| Frontend | Next.js 16, React 19, Tailwind CSS 4, shadcn/ui, Recharts |
| Backend | Go, Chi router, GORM, PostgreSQL, JWT |
| AI | Google Gemini (triage & suggestions) |

## Features

- **AI Triage** — Gemini analyzes new tickets, suggests priority/category/skills
- **Auto-assignment** — skill-based agent matching with workload balancing
- **SLA tracking** — automatic breach detection and marking
- **Problem management** — link related incidents to a root-cause problem ticket
- **Role-based access** — requester / agent / admin roles
- **Audit trail** — every state change logged
- **Dashboards** — real-time stats and charts

## Project Structure

```
.
├── client/       # Next.js frontend
└── server/       # Go REST API
```

## Quick Start

### Backend

```bash
cd server
cp .env.example .env
# Fill in DATABASE_URL, JWT_SECRET, and optional REDIS_URL / AI keys
go run ./cmd/api
```

### Frontend

```bash
cd client
cp .env.example .env.local
# Set NEXT_PUBLIC_API_URL to backend base URL
npm install
npm run dev
```

## Environment Variables

See `server/.env.example` for backend config.  
Frontend requires `NEXT_PUBLIC_API_URL` pointing to the running API.

## Database Migrations

Run in order from `server/internal/db/migrations/`:

1. `001_init.sql`
2. `002_migrate.sql`
3. `003_migrate.sql`
4. `004_ai_problem_management.sql`
