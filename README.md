# 🚨 Sentinel
### Enterprise IT Service Management Platform with AI-Powered Incident Triage

An intelligent, full-stack incident ticketing system that automates triage, routing, and SLA management for modern IT operations teams.

---

## ✨ Core Features

| Feature | Description |
|---------|-------------|
| 🤖 **AI Triage** | Auto-suggests priority, category, and required skills using Google Gemini |
| 🎯 **Smart Assignment** | Skill-based routing with intelligent workload balancing |
| ⏱️ **SLA Management** | Real-time response/resolve timers with breach detection |
| 🔗 **Problem Linking** | Track root-cause relationships between incidents |
| 🔐 **Role-Based Access** | Separate workflows for requesters, agents, and admins |
| 📊 **Live Dashboards** | Operational metrics, queue visibility, and team analytics |
| 📋 **Full Audit Trail** | Every state change logged for compliance and troubleshooting |

---

## 🏗️ Tech Stack

```
Frontend  → Next.js 16 + React 19 + Tailwind CSS 4 + shadcn/ui + Recharts
API       → Go (Chi router) + GORM + PostgreSQL + JWT auth
AI        → Google Gemini API (triage & suggestions)
Infra     → Docker + Docker Compose
```

---

## 🚀 Quick Start (30 seconds)

The fastest way to get Sentinel running locally with everything you need:

```bash
# Clone and navigate to project
git clone <repository-url>
cd sentinel

# Start everything (PostgreSQL + API + Database migrations)
docker compose up --build
```

✅ **Done!** Your system is ready:
- **API**: `http://localhost:8080/api/v1`
- **Frontend**: `http://localhost:3000` (after starting Next.js)

---

## 📋 Prerequisites

| Requirement | Version | Purpose |
|------------|---------|---------|
| **Docker** | Latest | Run containerized backend + PostgreSQL |
| **Node.js** | 20+ | Frontend development & build |
| **Go** | 1.22+ | Backend development (optional, if not using Docker) |
| **npm** | 10+ | Frontend package management |

> 💡 **Tip**: Using Docker? You only need Docker + Node.js!

---

## 🛠️ Setup Guide

### Option 1: Docker (Recommended) ⭐

Perfect for new developers and production deployments.

#### Step 1: Configure Environment
```bash
# Backend configuration
cp server/.env.example server/.env
# Edit server/.env and set:
#   - DATABASE_URL (auto-configured for Docker)
#   - JWT_SECRET (generate one: openssl rand -base64 32)
#   - GOOGLE_GEMINI_API_KEY (from Google Cloud Console)

# Frontend configuration
cp client/.env.example client/.env.local
# Verify NEXT_PUBLIC_API_BASE_URL=http://localhost:8080/api/v1
```

#### Step 2: Start Everything
```bash
docker compose up --build
```

**What happens automatically:**
- PostgreSQL starts (port 5432)
- Database migrations run
- API server starts (port 8080)
- Storage persists across restarts (Docker volume)

#### Step 3: Start Frontend
In a new terminal:
```bash
cd client
npm install
npm run dev
```

Open `http://localhost:3000` in your browser.

---

### Option 2: Local Development

For developers who prefer local PostgreSQL and hot-reloading.

#### Step 1: Setup PostgreSQL
```bash
# macOS (Homebrew)
brew install postgresql@15
brew services start postgresql@15

# Ubuntu/Debian
sudo apt-get install postgresql postgresql-contrib
sudo systemctl start postgresql

# Verify it's running
psql --version
```

#### Step 2: Configure Backend
```bash
cp server/.env.example server/.env
```

Edit `server/.env`:
```env
DATABASE_URL=postgres://postgres:password@localhost:5432/sentinel
JWT_SECRET=<generate: openssl rand -base64 32>
GOOGLE_GEMINI_API_KEY=<your-api-key>
PORT=8080
```

#### Step 3: Start Backend
```bash
cd server
go run ./cmd/api
```

Logs will show:
```
✓ Database migrations applied
✓ Server running on :8080
```

#### Step 4: Start Frontend
```bash
cd client
npm install
npm run dev
```

---

## 📁 Project Structure

```
sentinel/
├── client/                 # Next.js frontend
│   ├── app/               # React components & pages
│   ├── public/            # Static assets
│   ├── .env.example       # Frontend config template
│   └── package.json
│
├── server/                # Go backend
│   ├── cmd/api/           # Entry point
│   ├── internal/
│   │   ├── db/            # GORM models & migrations
│   │   ├── services/      # Business logic (Triage, Assignment, SLA)
│   │   ├── repositories/  # Data access layer
│   │   └── handlers/      # HTTP handlers
│   ├── .env.example       # Backend config template
│   └── go.mod
│
├── docker-compose.yml     # Single-command startup
└── README.md
```

---

## 🌐 API Endpoints

All endpoints require JWT authentication (login first).

```
POST   /api/v1/auth/register      Register new user
POST   /api/v1/auth/login         Get JWT token

POST   /api/v1/tickets            Create ticket
GET    /api/v1/tickets            List tickets (with filters)
GET    /api/v1/tickets/:id        Get ticket details
PATCH  /api/v1/tickets/:id        Update ticket status/assignment
POST   /api/v1/tickets/:id/notes  Add internal notes

GET    /api/v1/dashboard          Operational metrics & SLA summary
GET    /api/v1/queue              Agent's active queue
```

**Full API docs**: Check `server/internal/handlers/` for handler implementations.

---

## 🔄 Example Workflow

### 1️⃣ Register & Login
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"john","password":"secure123","role":"agent"}'

# Get JWT token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"john","password":"secure123"}'
```

### 2️⃣ Create a Ticket
```bash
curl -X POST http://localhost:8080/api/v1/tickets \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Network outage in Building A",
    "description": "Cannot connect to internal systems",
    "category": "NETWORK",
    "priority": "HIGH"
  }'
```

### 3️⃣ View Dashboard
Open `http://localhost:3000/dashboard` to see:
- Live ticket queues
- SLA compliance metrics
- Team performance analytics
- Response time trends

---

## 🤖 AI Triage System

When a ticket is created, Sentinel automatically:

1. **Analyzes** title + description with Google Gemini
2. **Suggests** optimal priority, category, and required skills
3. **Routes** to the best-matched agent based on workload
4. **Tracks** SLA timers (response & resolve)

**Configure Gemini API:**
1. Get API key from [Google Cloud Console](https://console.cloud.google.com)
2. Add to `server/.env`:
   ```env
   GOOGLE_GEMINI_API_KEY=your_key_here
   ```

---

## 🔐 Database Schema

Key tables (auto-created on startup):

```sql
users         → Agents, admins, requesters
tickets       → Incident records + metadata
assignments   → Ticket-to-agent relationships
sla_metrics   → Response/resolve time tracking
audit_logs    → Full change history
```

**View migrations**: `server/internal/db/migrations/`

---

## 📊 Monitoring & Logs

### Docker Logs
```bash
# All containers
docker compose logs -f

# Just API
docker compose logs -f api

# Just PostgreSQL
docker compose logs -f postgres
```

### Local Backend Logs
```bash
# Running in foreground shows all logs
cd server
go run ./cmd/api
```

Look for:
- ✓ Database connection successful
- ✓ JWT configured
- ✓ Gemini API connected
- ✓ Server listening on :8080

---

## 🚢 Deployment

### Frontend → Vercel
```bash
# 1. Push code to GitHub
git push origin main

# 2. Connect repo in Vercel dashboard
# 3. Set environment variables:
NEXT_PUBLIC_API_BASE_URL=https://your-api.com/api/v1

# 4. Deploy automatically on push
```

### Backend + DB → Docker Cloud/VPS

```bash
docker compose up -d --build
```

---

## 🧪 Development Tips

### Reset Database (Docker)
```bash
docker compose down -v  # Remove all data
docker compose up --build  # Fresh start
```

### Hot Reload Frontend
Next.js automatically reloads on file changes—just save and refresh browser.

### Hot Reload Backend (Local)
Install `air` for Go hot-reloading:
```bash
go install github.com/cosmtrek/air@latest
cd server
air  # Restarts on file changes
```

### View Postgres Data
```bash
docker compose exec postgres psql -U postgres -d sentinel
```

Inside psql:
```sql
\dt                    -- List all tables
SELECT * FROM tickets; -- View tickets
\q                     -- Exit
```

---

## 🐛 Troubleshooting

| Issue | Solution |
|-------|----------|
| **Port 8080 already in use** | `lsof -i :8080` → kill process or change `docker-compose.yml` |
| **Database connection failed** | Check `DATABASE_URL` in `.env`, ensure PostgreSQL is running |
| **API returns 401 Unauthorized** | Verify JWT token in `Authorization: Bearer <token>` header |
| **Gemini API errors** | Validate API key in Google Console, ensure billing is enabled |
| **Frontend blank page** | Check browser console (F12), verify `NEXT_PUBLIC_API_BASE_URL` |
| **Docker out of space** | Run `docker system prune -a` |

---

## 📚 Key Commands Cheatsheet

```bash
# Start everything
docker compose up --build

# View logs
docker compose logs -f

# Stop everything (keep data)
docker compose stop

# Completely reset (removes all data)
docker compose down -v

# Start frontend dev server
cd client && npm run dev

# Start backend locally
cd server && go run ./cmd/api

# Run backend tests
cd server && go test ./...

# Format Go code
cd server && go fmt ./...
```

---

## 🤝 Contributing

1. Fork the repository
2. Create feature branch: `git checkout -b feature/your-feature`
3. Commit changes: `git commit -m 'Add feature'`
4. Push to branch: `git push origin feature/your-feature`
5. Open Pull Request

---

## 📄 License

[Add your license here]

---

## 💡 Support & Resources

- **Issues?** Check the [Troubleshooting](#-troubleshooting) section above
- **Questions?** Open an issue on GitHub
- **Contributions?** We welcome PRs!

---


**Built with ❤️ for modern IT operations teams**
