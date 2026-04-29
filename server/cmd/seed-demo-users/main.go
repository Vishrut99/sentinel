package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/yourusername/incident-ticketing/internal/db"
	"github.com/yourusername/incident-ticketing/internal/domain"
)

const demoPassword = "DemoPass123!"

type demoAccount struct {
	Email       string
	FullName    string
	Role        string
	Password    string
	Department  string
	IsAvailable bool
	MaxTickets  int
	Skills      domain.SkillScoreList
}

type demoCategory struct {
	ID          uuid.UUID
	Name        string
	Description string
}

type demoComment struct {
	AuthorEmail string
	Body        string
	IsInternal  bool
	CreatedAt   time.Time
}

type demoAudit struct {
	ActorEmail string
	Action     string
	OldValue   any
	NewValue   any
	CreatedAt  time.Time
}

type demoTicket struct {
	Key                     string
	Title                   string
	Description             string
	Priority                string
	Status                  string
	CategoryName            string
	CreatedByEmail          string
	AssignedToEmail         string
	AssignmentJustification string
	ParentKey               string
	IsProblem               bool
	SLABreached             bool
	RequiredSkills          []string
	AIInsights              map[string]any
	DueAt                   *time.Time
	ResolvedAt              *time.Time
	SkillsEvaluatedAt       *time.Time
	CreatedAt               time.Time
	UpdatedAt               time.Time
	Comments                []demoComment
	Audits                  []demoAudit
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("warning: could not load .env file: %v", err)
	}

	databaseURL := firstEnv("DATABASE_URL", "DB_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	database, err := db.Connect(databaseURL)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	sqlDB, err := database.DB()
	if err != nil {
		log.Fatalf("failed to create sql db handle: %v", err)
	}
	defer sqlDB.Close()

	now := time.Now().UTC().Truncate(time.Minute)
	categories, accounts, tickets := buildDemoDataset(now)

	if err := database.Transaction(func(tx *gorm.DB) error {
		if err := resetWorkingData(tx); err != nil {
			return err
		}
		if err := ensureStaticLookups(tx); err != nil {
			return err
		}

		statusIDs, err := lookupStatusIDs(tx)
		if err != nil {
			return err
		}
		priorityIDs, err := lookupPriorityIDs(tx)
		if err != nil {
			return err
		}

		categoryIDs, err := seedCategories(tx, categories, now)
		if err != nil {
			return err
		}
		usersByEmail, err := seedUsers(tx, accounts, now)
		if err != nil {
			return err
		}
		agentsByEmail, err := seedAgents(tx, accounts, usersByEmail, now)
		if err != nil {
			return err
		}
		if _, err := seedTickets(tx, tickets, statusIDs, priorityIDs, categoryIDs, usersByEmail, agentsByEmail); err != nil {
			return err
		}

		return nil
	}); err != nil {
		log.Fatalf("failed to seed demo environment: %v", err)
	}

	log.Printf("seeded demo environment successfully: %d requesters, %d agents, %d admin, %d tickets", countByRole(accounts, "user"), countByRole(accounts, "agent"), countByRole(accounts, "admin"), len(tickets))
	log.Printf("all demo accounts use password=%s", demoPassword)
	for _, account := range accounts {
		log.Printf("role=%s email=%s password=%s", account.Role, account.Email, account.Password)
	}
}

func buildDemoDataset(now time.Time) ([]demoCategory, []demoAccount, []demoTicket) {
	categories := []demoCategory{
		{
			ID:          uuid.MustParse("10000000-0000-0000-0000-000000000001"),
			Name:        "Identity & Access",
			Description: "Authentication, authorization, SSO, and conditional access issues.",
		},
		{
			ID:          uuid.MustParse("10000000-0000-0000-0000-000000000002"),
			Name:        "Network Operations",
			Description: "Connectivity, VPN, routing, and office network incidents.",
		},
		{
			ID:          uuid.MustParse("10000000-0000-0000-0000-000000000003"),
			Name:        "Business Applications",
			Description: "Core line-of-business systems and enterprise platforms.",
		},
		{
			ID:          uuid.MustParse("10000000-0000-0000-0000-000000000004"),
			Name:        "Finance Systems",
			Description: "Approval workflows, invoice operations, and finance tooling support.",
		},
		{
			ID:          uuid.MustParse("10000000-0000-0000-0000-000000000005"),
			Name:        "End User Computing",
			Description: "Laptop, desktop, peripheral, and endpoint support requests.",
		},
	}

	accounts := []demoAccount{
		{
			Email:       "olivia.brooks@northstar-demo.com",
			FullName:    "Olivia Brooks",
			Role:        "admin",
			Password:    demoPassword,
			IsAvailable: true,
			MaxTickets:  0,
		},
		{
			Email:       "maya.chen@northstar-demo.com",
			FullName:    "Maya Chen",
			Role:        "agent",
			Password:    demoPassword,
			Department:  "Identity & Network Operations",
			IsAvailable: true,
			MaxTickets:  6,
			Skills: domain.SkillScoreList{
				{Name: "Identity & Access", Score: 93},
				{Name: "SSO & MFA", Score: 90},
				{Name: "VPN", Score: 88},
				{Name: "Conditional Access", Score: 85},
			},
		},
		{
			Email:       "liam.ortega@northstar-demo.com",
			FullName:    "Liam Ortega",
			Role:        "agent",
			Password:    demoPassword,
			Department:  "Business Platforms",
			IsAvailable: true,
			MaxTickets:  5,
			Skills: domain.SkillScoreList{
				{Name: "ERP Support", Score: 95},
				{Name: "Finance Systems", Score: 92},
				{Name: "Database Diagnostics", Score: 84},
				{Name: "Vendor Coordination", Score: 80},
			},
		},
		{
			Email:       "priya.nair@northstar-demo.com",
			FullName:    "Priya Nair",
			Role:        "agent",
			Password:    demoPassword,
			Department:  "End User Computing",
			IsAvailable: true,
			MaxTickets:  6,
			Skills: domain.SkillScoreList{
				{Name: "Hardware Troubleshooting", Score: 94},
				{Name: "MacOS Support", Score: 90},
				{Name: "Endpoint Management", Score: 87},
				{Name: "Power & Battery Diagnostics", Score: 84},
			},
		},
		{Email: "aisha.patel@northstar-demo.com", FullName: "Aisha Patel", Role: "user", Password: demoPassword},
		{Email: "daniel.kim@northstar-demo.com", FullName: "Daniel Kim", Role: "user", Password: demoPassword},
		{Email: "emma.wilson@northstar-demo.com", FullName: "Emma Wilson", Role: "user", Password: demoPassword},
		{Email: "noah.bennett@northstar-demo.com", FullName: "Noah Bennett", Role: "user", Password: demoPassword},
		{Email: "sophia.martinez@northstar-demo.com", FullName: "Sophia Martinez", Role: "user", Password: demoPassword},
		{Email: "arjun.shah@northstar-demo.com", FullName: "Arjun Shah", Role: "user", Password: demoPassword},
		{Email: "chloe.rivera@northstar-demo.com", FullName: "Chloe Rivera", Role: "user", Password: demoPassword},
		{Email: "ethan.cole@northstar-demo.com", FullName: "Ethan Cole", Role: "user", Password: demoPassword},
		{Email: "mia.foster@northstar-demo.com", FullName: "Mia Foster", Role: "user", Password: demoPassword},
		{Email: "rahul.mehta@northstar-demo.com", FullName: "Rahul Mehta", Role: "user", Password: demoPassword},
	}

	vpnCreated := now.Add(-3 * time.Hour)
	vpnUpdated := now.Add(-70 * time.Minute)
	vpnDue := vpnCreated.Add(24 * time.Hour)

	problemCreated := now.Add(-7 * time.Hour)
	problemUpdated := now.Add(-15 * time.Minute)
	problemDue := problemCreated.Add(4 * time.Hour)

	childCreated := now.Add(-5 * time.Hour)
	childUpdated := now.Add(-30 * time.Minute)
	childDue := childCreated.Add(8 * time.Hour)

	macCreated := now.Add(-30 * time.Hour)
	macResolved := macCreated.Add(6 * time.Hour)
	macSkillsEvaluated := macResolved.Add(20 * time.Minute)
	macUpdated := macResolved.Add(35 * time.Minute)
	macDue := macCreated.Add(8 * time.Hour)

	tickets := []demoTicket{
		{
			Key:                     "vpn-reconnect",
			Title:                   "Contractor VPN reconnect loop after password reset",
			Description:             "Remote contractors can authenticate to the VPN gateway, but the client disconnects within 20 seconds and asks for credentials again. The failures started right after a forced password reset and only affect the contractor access policy group.",
			Priority:                "medium",
			Status:                  "in_progress",
			CategoryName:            "Network Operations",
			CreatedByEmail:          "aisha.patel@northstar-demo.com",
			AssignedToEmail:         "maya.chen@northstar-demo.com",
			AssignmentJustification: "Assigned to Maya Chen because the issue combines VPN instability with identity-policy drift after a password reset event.",
			RequiredSkills:          []string{"VPN", "Identity & Access", "Conditional Access"},
			AIInsights: map[string]any{
				"triage_status":            "completed",
				"summary":                  "Likely policy mismatch between VPN access groups and conditional access rules after the contractor password reset.",
				"suggested_priority":       "medium",
				"suggested_category":       "Network Operations",
				"required_skills":          []string{"VPN", "Identity & Access", "Conditional Access"},
				"recommended_first_action": "Validate contractor group claims in the IdP and compare them with the VPN policy mapping.",
			},
			DueAt:     &vpnDue,
			CreatedAt: vpnCreated,
			UpdatedAt: vpnUpdated,
			Comments: []demoComment{
				{
					AuthorEmail: "aisha.patel@northstar-demo.com",
					Body:        "Three contractors in APAC have the same reconnect loop. They can sign in, see the tunnel establish, and then get bounced back to the login screen.",
					IsInternal:  false,
					CreatedAt:   vpnCreated.Add(12 * time.Minute),
				},
				{
					AuthorEmail: "maya.chen@northstar-demo.com",
					Body:        "Internal note: VPN logs show the tunnel forming successfully, then being dropped when the post-auth policy check runs. Reviewing conditional access deltas from this morning.",
					IsInternal:  true,
					CreatedAt:   vpnCreated.Add(48 * time.Minute),
				},
			},
			Audits: []demoAudit{
				{
					ActorEmail: "aisha.patel@northstar-demo.com",
					Action:     "created",
					NewValue: map[string]any{
						"required_skills": []string{"VPN", "Identity & Access", "Conditional Access"},
					},
					CreatedAt: vpnCreated,
				},
				{
					ActorEmail: "olivia.brooks@northstar-demo.com",
					Action:     "assigned",
					NewValue: map[string]any{
						"agent_name":               "Maya Chen",
						"assignment_justification": "Assigned to Maya Chen because the issue combines VPN instability with identity-policy drift after a password reset event.",
						"required_skills":          []string{"VPN", "Identity & Access", "Conditional Access"},
					},
					CreatedAt: vpnCreated.Add(20 * time.Minute),
				},
			},
		},
		{
			Key:                     "erp-outage",
			Title:                   "Quarter-end ERP posting outage across APAC finance",
			Description:             "Purchase orders and invoice batches stall at the posting step in the ERP platform for APAC finance teams. The failure began during quarter-end close and appears to affect every tenant on the primary region.",
			Priority:                "critical",
			Status:                  "in_progress",
			CategoryName:            "Business Applications",
			CreatedByEmail:          "sophia.martinez@northstar-demo.com",
			AssignedToEmail:         "liam.ortega@northstar-demo.com",
			AssignmentJustification: "Assigned to Liam Ortega due to deep ERP support coverage, finance systems ownership, and prior quarter-close incident experience.",
			IsProblem:               true,
			SLABreached:             true,
			RequiredSkills:          []string{"ERP Support", "Finance Systems", "Database Diagnostics"},
			AIInsights: map[string]any{
				"triage_status":      "completed",
				"summary":            "Wide-impact ERP degradation during financial close indicates a platform-level problem rather than a single workflow defect.",
				"suggested_priority": "critical",
				"suggested_category": "Business Applications",
				"required_skills":    []string{"ERP Support", "Finance Systems", "Database Diagnostics"},
				"customer_impact":    "APAC finance close activities are blocked for multiple teams.",
			},
			DueAt:     &problemDue,
			CreatedAt: problemCreated,
			UpdatedAt: problemUpdated,
			Comments: []demoComment{
				{
					AuthorEmail: "sophia.martinez@northstar-demo.com",
					Body:        "Finance controllers across Singapore and Sydney cannot post invoices or purchase orders. Quarter-end close is now blocked.",
					IsInternal:  false,
					CreatedAt:   problemCreated.Add(10 * time.Minute),
				},
				{
					AuthorEmail: "liam.ortega@northstar-demo.com",
					Body:        "Internal note: regional database CPU is spiking during posting jobs. Vendor bridge opened and we are collecting trace IDs from the busiest tenants.",
					IsInternal:  true,
					CreatedAt:   problemCreated.Add(55 * time.Minute),
				},
			},
			Audits: []demoAudit{
				{
					ActorEmail: "sophia.martinez@northstar-demo.com",
					Action:     "created",
					NewValue: map[string]any{
						"required_skills": []string{"ERP Support", "Finance Systems", "Database Diagnostics"},
					},
					CreatedAt: problemCreated,
				},
				{
					ActorEmail: "olivia.brooks@northstar-demo.com",
					Action:     "assigned",
					NewValue: map[string]any{
						"agent_name":               "Liam Ortega",
						"assignment_justification": "Assigned to Liam Ortega due to deep ERP support coverage, finance systems ownership, and prior quarter-close incident experience.",
						"required_skills":          []string{"ERP Support", "Finance Systems", "Database Diagnostics"},
					},
					CreatedAt: problemCreated.Add(18 * time.Minute),
				},
				{
					ActorEmail: "olivia.brooks@northstar-demo.com",
					Action:     "sla_breached",
					NewValue: map[string]any{
						"required_skills": []string{"ERP Support", "Finance Systems", "Database Diagnostics"},
					},
					CreatedAt: problemDue.Add(5 * time.Minute),
				},
			},
		},
		{
			Key:                     "invoice-approvals",
			Title:                   "Invoice approvals timing out after ERP outage",
			Description:             "Approvers can open queued invoices, but the submit action times out after roughly 30 seconds and the workflow remains stuck in Pending Approval. Business users now suspect it is tied to the wider ERP posting issue.",
			Priority:                "high",
			Status:                  "in_progress",
			CategoryName:            "Finance Systems",
			CreatedByEmail:          "rahul.mehta@northstar-demo.com",
			AssignedToEmail:         "liam.ortega@northstar-demo.com",
			AssignmentJustification: "Linked child incident kept with Liam Ortega so the finance workflow symptoms stay attached to the active ERP root-cause investigation.",
			ParentKey:               "erp-outage",
			RequiredSkills:          []string{"ERP Support", "Workflow Diagnostics", "Finance Systems"},
			AIInsights: map[string]any{
				"triage_status":      "completed",
				"summary":            "Approval workflow timeout likely shares the same backend pressure as the active ERP posting outage.",
				"suggested_priority": "high",
				"suggested_category": "Finance Systems",
				"required_skills":    []string{"ERP Support", "Workflow Diagnostics", "Finance Systems"},
			},
			DueAt:     &childDue,
			CreatedAt: childCreated,
			UpdatedAt: childUpdated,
			Comments: []demoComment{
				{
					AuthorEmail: "rahul.mehta@northstar-demo.com",
					Body:        "Approvers can review invoices, but every submit attempt spins and then fails with a timeout. We have 42 approvals blocked.",
					IsInternal:  false,
					CreatedAt:   childCreated.Add(8 * time.Minute),
				},
				{
					AuthorEmail: "liam.ortega@northstar-demo.com",
					Body:        "Internal note: linking this to the quarter-end ERP problem ticket so finance workflow failures stay consolidated while vendor diagnostics are in progress.",
					IsInternal:  true,
					CreatedAt:   childCreated.Add(35 * time.Minute),
				},
			},
			Audits: []demoAudit{
				{
					ActorEmail: "rahul.mehta@northstar-demo.com",
					Action:     "created",
					NewValue: map[string]any{
						"required_skills": []string{"ERP Support", "Workflow Diagnostics", "Finance Systems"},
					},
					CreatedAt: childCreated,
				},
				{
					ActorEmail: "olivia.brooks@northstar-demo.com",
					Action:     "linked_problem",
					NewValue: map[string]any{
						"required_skills": []string{"ERP Support", "Workflow Diagnostics", "Finance Systems"},
					},
					CreatedAt: childCreated.Add(14 * time.Minute),
				},
				{
					ActorEmail: "olivia.brooks@northstar-demo.com",
					Action:     "assigned",
					NewValue: map[string]any{
						"agent_name":               "Liam Ortega",
						"assignment_justification": "Linked child incident kept with Liam Ortega so the finance workflow symptoms stay attached to the active ERP root-cause investigation.",
						"required_skills":          []string{"ERP Support", "Workflow Diagnostics", "Finance Systems"},
					},
					CreatedAt: childCreated.Add(22 * time.Minute),
				},
			},
		},
		{
			Key:                     "macbook-battery",
			Title:                   "Field sales MacBook overheats and shuts down during client demos",
			Description:             "The assigned MacBook Pro gets hot near the trackpad, battery percentage drops rapidly, and the device shuts down without warning during video demos. The employee already tried SMC and NVRAM resets before opening the ticket.",
			Priority:                "high",
			Status:                  "resolved",
			CategoryName:            "End User Computing",
			CreatedByEmail:          "chloe.rivera@northstar-demo.com",
			AssignedToEmail:         "priya.nair@northstar-demo.com",
			AssignmentJustification: "Assigned to Priya Nair because of strong endpoint hardware diagnostics and hands-on Mac support experience.",
			RequiredSkills:          []string{"Hardware Troubleshooting", "MacOS Support", "Power & Battery Diagnostics"},
			AIInsights: map[string]any{
				"triage_status":      "completed",
				"summary":            "Symptoms point to a degrading battery pack or power controller issue rather than an operating-system-only problem.",
				"suggested_priority": "high",
				"suggested_category": "End User Computing",
				"required_skills":    []string{"Hardware Troubleshooting", "MacOS Support", "Power & Battery Diagnostics"},
			},
			DueAt:             &macDue,
			ResolvedAt:        &macResolved,
			SkillsEvaluatedAt: &macSkillsEvaluated,
			CreatedAt:         macCreated,
			UpdatedAt:         macUpdated,
			Comments: []demoComment{
				{
					AuthorEmail: "chloe.rivera@northstar-demo.com",
					Body:        "The laptop shut down twice during customer demos yesterday. It becomes hot near the palm rest and the battery drops from 60% to single digits in under 20 minutes.",
					IsInternal:  false,
					CreatedAt:   macCreated.Add(16 * time.Minute),
				},
				{
					AuthorEmail: "priya.nair@northstar-demo.com",
					Body:        "Internal note: battery health is degraded and peak performance capability is disabled. Dispatching a replacement unit and scheduling the damaged device for depot repair.",
					IsInternal:  true,
					CreatedAt:   macCreated.Add(2*time.Hour + 10*time.Minute),
				},
				{
					AuthorEmail: "priya.nair@northstar-demo.com",
					Body:        "Replacement MacBook has been issued, user profile restored, and the affected device is queued for battery service. Closing as resolved.",
					IsInternal:  false,
					CreatedAt:   macResolved.Add(5 * time.Minute),
				},
			},
			Audits: []demoAudit{
				{
					ActorEmail: "chloe.rivera@northstar-demo.com",
					Action:     "created",
					NewValue: map[string]any{
						"required_skills": []string{"Hardware Troubleshooting", "MacOS Support", "Power & Battery Diagnostics"},
					},
					CreatedAt: macCreated,
				},
				{
					ActorEmail: "olivia.brooks@northstar-demo.com",
					Action:     "assigned",
					NewValue: map[string]any{
						"agent_name":               "Priya Nair",
						"assignment_justification": "Assigned to Priya Nair because of strong endpoint hardware diagnostics and hands-on Mac support experience.",
						"required_skills":          []string{"Hardware Troubleshooting", "MacOS Support", "Power & Battery Diagnostics"},
					},
					CreatedAt: macCreated.Add(22 * time.Minute),
				},
				{
					ActorEmail: "priya.nair@northstar-demo.com",
					Action:     "resolved",
					NewValue: map[string]any{
						"required_skills": []string{"Hardware Troubleshooting", "MacOS Support", "Power & Battery Diagnostics"},
					},
					CreatedAt: macResolved,
				},
				{
					ActorEmail: "olivia.brooks@northstar-demo.com",
					Action:     "skills_evaluated",
					NewValue: map[string]any{
						"required_skills": []string{"Hardware Troubleshooting", "MacOS Support", "Power & Battery Diagnostics"},
					},
					CreatedAt: macSkillsEvaluated,
				},
			},
		},
	}

	return categories, accounts, tickets
}

func resetWorkingData(tx *gorm.DB) error {
	if err := tx.Exec("TRUNCATE TABLE audit_logs, comments, tickets, agents, users, categories RESTART IDENTITY CASCADE").Error; err != nil {
		return fmt.Errorf("resetWorkingData truncate: %w", err)
	}
	if err := tx.Exec("ALTER SEQUENCE IF EXISTS ticket_number_seq RESTART WITH 1").Error; err != nil {
		return fmt.Errorf("resetWorkingData reset ticket number sequence: %w", err)
	}
	return nil
}

func ensureStaticLookups(tx *gorm.DB) error {
	if err := tx.Exec(`
		INSERT INTO ticket_statuses (name, code)
		VALUES
			('open', 'OPEN'),
			('in_progress', 'IN_PROGRESS'),
			('resolved', 'RESOLVED'),
			('closed', 'CLOSED'),
			('cancelled', 'CANCELLED')
		ON CONFLICT (name) DO NOTHING
	`).Error; err != nil {
		return fmt.Errorf("ensureStaticLookups statuses: %w", err)
	}

	if err := tx.Exec(`
		INSERT INTO priorities (name, sla_response_hrs, sla_resolve_hrs)
		VALUES
			('low', 8, 72),
			('medium', 4, 24),
			('high', 1, 8),
			('critical', 1, 4)
		ON CONFLICT (name) DO NOTHING
	`).Error; err != nil {
		return fmt.Errorf("ensureStaticLookups priorities: %w", err)
	}

	return nil
}

func seedCategories(tx *gorm.DB, categories []demoCategory, now time.Time) (map[string]uuid.UUID, error) {
	categoryIDs := make(map[string]uuid.UUID, len(categories))
	for _, category := range categories {
		record := domain.Category{
			ID:          category.ID,
			Name:        category.Name,
			Description: stringPtr(category.Description),
			CreatedAt:   now,
		}
		if err := tx.Create(&record).Error; err != nil {
			return nil, fmt.Errorf("seedCategories create %s: %w", category.Name, err)
		}
		categoryIDs[strings.ToLower(category.Name)] = category.ID
	}
	return categoryIDs, nil
}

func seedUsers(tx *gorm.DB, accounts []demoAccount, now time.Time) (map[string]domain.User, error) {
	usersByEmail := make(map[string]domain.User, len(accounts))
	for _, account := range accounts {
		hash, err := bcrypt.GenerateFromPassword([]byte(account.Password), 12)
		if err != nil {
			return nil, fmt.Errorf("seedUsers hash %s: %w", account.Email, err)
		}

		record := domain.User{
			ID:           uuid.New(),
			Email:        normalizeEmail(account.Email),
			PasswordHash: string(hash),
			FullName:     account.FullName,
			Role:         account.Role,
			IsActive:     true,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := tx.Create(&record).Error; err != nil {
			return nil, fmt.Errorf("seedUsers create %s: %w", account.Email, err)
		}
		usersByEmail[record.Email] = record
	}

	return usersByEmail, nil
}

func seedAgents(tx *gorm.DB, accounts []demoAccount, usersByEmail map[string]domain.User, now time.Time) (map[string]domain.Agent, error) {
	agentsByEmail := make(map[string]domain.Agent)
	for _, account := range accounts {
		if account.Role != "agent" {
			continue
		}

		user := usersByEmail[normalizeEmail(account.Email)]
		record := domain.Agent{
			ID:          uuid.New(),
			UserID:      user.ID,
			Department:  stringPtr(account.Department),
			IsAvailable: account.IsAvailable,
			MaxTickets:  account.MaxTickets,
			Skills:      account.Skills,
			CreatedAt:   now,
		}
		if err := tx.Create(&record).Error; err != nil {
			return nil, fmt.Errorf("seedAgents create %s: %w", account.Email, err)
		}
		agentsByEmail[normalizeEmail(account.Email)] = record
	}

	return agentsByEmail, nil
}

func seedTickets(
	tx *gorm.DB,
	tickets []demoTicket,
	statusIDs map[string]int16,
	priorityIDs map[string]int16,
	categoryIDs map[string]uuid.UUID,
	usersByEmail map[string]domain.User,
	agentsByEmail map[string]domain.Agent,
) (map[string]domain.Ticket, error) {
	ticketsByKey := make(map[string]domain.Ticket, len(tickets))

	for _, ticket := range tickets {
		statusID, ok := statusIDs[strings.ToLower(ticket.Status)]
		if !ok {
			return nil, fmt.Errorf("seedTickets: unknown status %q", ticket.Status)
		}
		priorityID, ok := priorityIDs[strings.ToLower(ticket.Priority)]
		if !ok {
			return nil, fmt.Errorf("seedTickets: unknown priority %q", ticket.Priority)
		}
		categoryID, ok := categoryIDs[strings.ToLower(ticket.CategoryName)]
		if !ok {
			return nil, fmt.Errorf("seedTickets: unknown category %q", ticket.CategoryName)
		}

		creator, ok := usersByEmail[normalizeEmail(ticket.CreatedByEmail)]
		if !ok {
			return nil, fmt.Errorf("seedTickets: unknown creator %q", ticket.CreatedByEmail)
		}

		var assignedTo *uuid.UUID
		if ticket.AssignedToEmail != "" {
			agent, ok := agentsByEmail[normalizeEmail(ticket.AssignedToEmail)]
			if !ok {
				return nil, fmt.Errorf("seedTickets: unknown assignee %q", ticket.AssignedToEmail)
			}
			assignedTo = uuidPtr(agent.ID)
		}

		var parentID *uuid.UUID
		if ticket.ParentKey != "" {
			parent, ok := ticketsByKey[ticket.ParentKey]
			if !ok {
				return nil, fmt.Errorf("seedTickets: parent ticket %q not seeded yet", ticket.ParentKey)
			}
			parentID = uuidPtr(parent.ID)
		}

		record := domain.Ticket{
			ID:                      uuid.New(),
			TicketNumber:            "",
			Title:                   ticket.Title,
			Description:             stringPtr(ticket.Description),
			AIInsights:              marshalJSON(ticket.AIInsights),
			RequiredSkills:          domain.SkillNameList(ticket.RequiredSkills),
			StatusID:                statusID,
			PriorityID:              priorityID,
			CategoryID:              categoryID,
			CreatedBy:               creator.ID,
			AssignedTo:              assignedTo,
			AssignmentJustification: stringPtr(ticket.AssignmentJustification),
			ParentID:                parentID,
			IsProblem:               ticket.IsProblem,
			SLABreached:             ticket.SLABreached,
			DueAt:                   ticket.DueAt,
			ResolvedAt:              ticket.ResolvedAt,
			SkillsEvaluatedAt:       ticket.SkillsEvaluatedAt,
			CreatedAt:               ticket.CreatedAt,
			UpdatedAt:               ticket.UpdatedAt,
		}

		if err := tx.Create(&record).Error; err != nil {
			return nil, fmt.Errorf("seedTickets create %s: %w", ticket.Title, err)
		}

		if err := tx.First(&record, "id = ?", record.ID).Error; err != nil {
			return nil, fmt.Errorf("seedTickets reload %s: %w", ticket.Title, err)
		}
		ticketsByKey[ticket.Key] = record
	}

	for _, ticket := range tickets {
		storedTicket := ticketsByKey[ticket.Key]
		for _, comment := range ticket.Comments {
			author, ok := usersByEmail[normalizeEmail(comment.AuthorEmail)]
			if !ok {
				return nil, fmt.Errorf("seedTickets: unknown comment author %q", comment.AuthorEmail)
			}

			record := domain.Comment{
				ID:         uuid.New(),
				TicketID:   storedTicket.ID,
				AuthorID:   author.ID,
				Body:       comment.Body,
				IsInternal: comment.IsInternal,
				CreatedAt:  comment.CreatedAt,
			}
			if err := tx.Create(&record).Error; err != nil {
				return nil, fmt.Errorf("seedTickets create comment for %s: %w", ticket.Title, err)
			}
		}

		for _, audit := range ticket.Audits {
			actor, ok := usersByEmail[normalizeEmail(audit.ActorEmail)]
			if !ok {
				return nil, fmt.Errorf("seedTickets: unknown audit actor %q", audit.ActorEmail)
			}

			record := domain.AuditLog{
				ID:        uuid.New(),
				TicketID:  storedTicket.ID,
				ActorID:   actor.ID,
				Action:    audit.Action,
				OldValue:  marshalJSON(audit.OldValue),
				NewValue:  marshalJSON(audit.NewValue),
				CreatedAt: audit.CreatedAt,
			}
			if err := tx.Create(&record).Error; err != nil {
				return nil, fmt.Errorf("seedTickets create audit for %s: %w", ticket.Title, err)
			}
		}
	}

	return ticketsByKey, nil
}

func lookupStatusIDs(tx *gorm.DB) (map[string]int16, error) {
	var statuses []domain.TicketStatus
	if err := tx.Find(&statuses).Error; err != nil {
		return nil, fmt.Errorf("lookupStatusIDs: %w", err)
	}

	ids := make(map[string]int16, len(statuses))
	for _, status := range statuses {
		ids[strings.ToLower(status.Name)] = status.ID
	}
	return ids, nil
}

func lookupPriorityIDs(tx *gorm.DB) (map[string]int16, error) {
	var priorities []domain.Priority
	if err := tx.Find(&priorities).Error; err != nil {
		return nil, fmt.Errorf("lookupPriorityIDs: %w", err)
	}

	ids := make(map[string]int16, len(priorities))
	for _, priority := range priorities {
		ids[strings.ToLower(priority.Name)] = priority.ID
	}
	return ids, nil
}

func marshalJSON(value any) domain.JSONPayload {
	if value == nil {
		return nil
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		log.Fatalf("marshalJSON failed: %v", err)
	}
	return domain.JSONPayload(encoded)
}

func countByRole(accounts []demoAccount, role string) int {
	count := 0
	for _, account := range accounts {
		if account.Role == role {
			count++
		}
	}
	return count
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func normalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func stringPtr(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func uuidPtr(value uuid.UUID) *uuid.UUID {
	return &value
}
