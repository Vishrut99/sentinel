package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/yourusername/incident-ticketing/internal/domain"
	"gorm.io/gorm"
)

var skillKeywordMap = map[string][]string{
	"VPN":                 {"vpn", "remote access", "remote login", "tunnel"},
	"Network":             {"network", "internet", "wifi", "wi-fi", "router", "switch", "latency", "packet"},
	"Account Access":      {"password", "sign in", "signin", "login", "access", "permission", "role", "locked out", "mfa"},
	"Email":               {"email", "mailbox", "outlook", "exchange"},
	"Database":            {"database", "sql", "postgres", "mysql", "query", "schema"},
	"Application Support": {"application", "app", "portal", "workflow", "screen", "page", "form", "service unavailable"},
	"Desktop Support":     {"laptop", "desktop", "windows", "workstation", "monitor", "keyboard", "mouse"},
	"Printer Support":     {"printer", "print", "scanner"},
	"Cloud":               {"aws", "azure", "gcp", "cloud", "kubernetes", "container"},
	"Deployment":          {"deploy", "release", "pipeline", "build", "rollback"},
	"Security":            {"security", "phishing", "malware", "ransomware", "breach", "firewall", "certificate"},
	"Incident Triage":     {"incident", "outage", "degraded", "escalation", "sev", "priority"},
}

var skillAcronyms = map[string]string{
	"vpn": "VPN",
	"mfa": "MFA",
	"sql": "SQL",
	"aws": "AWS",
	"gcp": "GCP",
	"api": "API",
}

func sanitizeSkillScores(skills domain.SkillScoreList) (domain.SkillScoreList, error) {
	if len(skills) == 0 {
		return domain.SkillScoreList{}, nil
	}

	deduped := make(map[string]domain.SkillScore)
	for _, skill := range skills {
		name := canonicalSkillName(skill.Name)
		if name == "" {
			return nil, fmt.Errorf("skill name is required")
		}
		if skill.Score < 0 || skill.Score > 100 {
			return nil, fmt.Errorf("skill score for %s must be between 0 and 100", name)
		}

		key := normalizeSkillKey(name)
		current, exists := deduped[key]
		if !exists || skill.Score > current.Score {
			deduped[key] = domain.SkillScore{Name: name, Score: skill.Score}
		}
	}

	result := make(domain.SkillScoreList, 0, len(deduped))
	for _, skill := range deduped {
		result = append(result, skill)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result, nil
}

func sanitizeSkillNames(skills []string) domain.SkillNameList {
	if len(skills) == 0 {
		return domain.SkillNameList{}
	}

	seen := make(map[string]struct{}, len(skills))
	result := make(domain.SkillNameList, 0, len(skills))
	for _, skill := range skills {
		name := canonicalSkillName(skill)
		if name == "" {
			continue
		}
		key := normalizeSkillKey(name)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, name)
	}
	return result
}

func defaultAgentSkillsForDepartment(department string) domain.SkillScoreList {
	switch strings.ToLower(strings.TrimSpace(department)) {
	case "network", "network operations", "infrastructure":
		return domain.SkillScoreList{
			{Name: "Network", Score: 88},
			{Name: "VPN", Score: 84},
			{Name: "Incident Triage", Score: 78},
		}
	case "security", "soc":
		return domain.SkillScoreList{
			{Name: "Security", Score: 90},
			{Name: "Account Access", Score: 76},
			{Name: "Incident Triage", Score: 80},
		}
	case "platform", "devops", "engineering":
		return domain.SkillScoreList{
			{Name: "Application Support", Score: 82},
			{Name: "Deployment", Score: 86},
			{Name: "Cloud", Score: 84},
		}
	default:
		return domain.SkillScoreList{
			{Name: "Application Support", Score: 78},
			{Name: "Account Access", Score: 76},
			{Name: "Incident Triage", Score: 74},
		}
	}
}

func resolveRequiredSkills(req domain.CreateTicketReq, aiInsights json.RawMessage) domain.SkillNameList {
	merged := make([]string, 0, len(req.RequiredSkills)+6)
	merged = append(merged, req.RequiredSkills...)
	merged = append(merged, extractRequiredSkillsFromAIInsights(aiInsights)...)
	merged = append(merged, inferRequiredSkillsFromText(req.Title, req.Description)...)

	result := sanitizeSkillNames(merged)
	if len(result) == 0 {
		return domain.SkillNameList{"Incident Triage"}
	}
	return result
}

func extractRequiredSkillsFromAIInsights(aiInsights json.RawMessage) []string {
	if len(aiInsights) == 0 {
		return nil
	}

	var payload map[string]any
	if err := json.Unmarshal(aiInsights, &payload); err != nil {
		return nil
	}

	rawSkills, exists := payload["required_skills"]
	if !exists {
		return nil
	}

	switch typed := rawSkills.(type) {
	case string:
		if typed == "" {
			return nil
		}
		parts := strings.Split(typed, ",")
		skills := make([]string, 0, len(parts))
		for _, part := range parts {
			skills = append(skills, part)
		}
		return skills
	case []any:
		skills := make([]string, 0, len(typed))
		for _, item := range typed {
			switch value := item.(type) {
			case string:
				skills = append(skills, value)
			case map[string]any:
				if name, ok := value["name"].(string); ok {
					skills = append(skills, name)
				}
			}
		}
		return skills
	default:
		return nil
	}
}

func inferRequiredSkillsFromText(title, description string) []string {
	searchable := strings.ToLower(strings.Join([]string{title, description}, " "))
	if strings.TrimSpace(searchable) == "" {
		return nil
	}

	matched := make([]string, 0, 4)
	for skillName, keywords := range skillKeywordMap {
		for _, keyword := range keywords {
			if strings.Contains(searchable, keyword) {
				matched = append(matched, skillName)
				break
			}
		}
	}

	sort.Strings(matched)
	return matched
}

type assignmentCandidate struct {
	agent         domain.Agent
	activeTickets int
	matchedSkills []domain.SkillScore
	score         float64
}

func pickBestAgentForTicket(agents []domain.Agent, activeCounts map[uuid.UUID]int, requiredSkills domain.SkillNameList) (*domain.Agent, string) {
	var best *assignmentCandidate

	for _, agent := range agents {
		if !agent.IsAvailable || agent.MaxTickets < 1 {
			continue
		}

		activeTickets := activeCounts[agent.ID]
		if activeTickets >= agent.MaxTickets {
			continue
		}

		matchScore, matchedSkills := calculateSkillMatch(agent.Skills, requiredSkills)
		capacityScore := 1 - (float64(activeTickets) / float64(agent.MaxTickets))
		totalScore := capacityScore
		if len(requiredSkills) > 0 {
			totalScore = (0.72 * matchScore) + (0.28 * capacityScore)
		}

		candidate := &assignmentCandidate{
			agent:         agent,
			activeTickets: activeTickets,
			matchedSkills: matchedSkills,
			score:         totalScore,
		}

		if best == nil ||
			candidate.score > best.score ||
			(candidate.score == best.score && len(candidate.matchedSkills) > len(best.matchedSkills)) ||
			(candidate.score == best.score && len(candidate.matchedSkills) == len(best.matchedSkills) && candidate.activeTickets < best.activeTickets) {
			best = candidate
		}
	}

	if best == nil {
		return nil, ""
	}

	selected := best.agent
	return &selected, buildAutoAssignmentJustification(best, requiredSkills)
}

func calculateSkillMatch(agentSkills domain.SkillScoreList, requiredSkills domain.SkillNameList) (float64, []domain.SkillScore) {
	if len(requiredSkills) == 0 {
		return 0, nil
	}

	skillMap := make(map[string]domain.SkillScore, len(agentSkills))
	for _, skill := range agentSkills {
		key := normalizeSkillKey(skill.Name)
		current, exists := skillMap[key]
		if !exists || skill.Score > current.Score {
			skillMap[key] = skill
		}
	}

	matched := make([]domain.SkillScore, 0, len(requiredSkills))
	totalSkillStrength := 0.0
	for _, required := range requiredSkills {
		if skill, exists := skillMap[normalizeSkillKey(required)]; exists {
			matched = append(matched, skill)
			totalSkillStrength += float64(skill.Score) / 100
		}
	}

	coverage := float64(len(matched)) / float64(len(requiredSkills))
	averageStrength := totalSkillStrength / float64(len(requiredSkills))
	return (0.65 * coverage) + (0.35 * averageStrength), matched
}

func buildAutoAssignmentJustification(candidate *assignmentCandidate, requiredSkills domain.SkillNameList) string {
	activeCapacity := fmt.Sprintf("%d/%d active tickets", candidate.activeTickets, candidate.agent.MaxTickets)
	if len(requiredSkills) == 0 {
		return fmt.Sprintf(
			"Auto-assigned to %s based on current availability and capacity (%s).",
			candidate.agent.User.FullName,
			activeCapacity,
		)
	}

	if len(candidate.matchedSkills) == 0 {
		return fmt.Sprintf(
			"Auto-assigned to %s as the best available fallback while no close skill match was available. Required skills: %s. Current capacity: %s.",
			candidate.agent.User.FullName,
			strings.Join(requiredSkills, ", "),
			activeCapacity,
		)
	}

	matchedNames := make([]string, 0, len(candidate.matchedSkills))
	for _, skill := range candidate.matchedSkills {
		matchedNames = append(matchedNames, skill.Name)
	}

	return fmt.Sprintf(
		"Auto-assigned to %s based on the strongest skill match for %s and current capacity (%s).",
		candidate.agent.User.FullName,
		strings.Join(matchedNames, ", "),
		activeCapacity,
	)
}

func buildManualAssignmentJustification(agent *domain.Agent, activeTickets int) string {
	if agent == nil {
		return "Assigned manually."
	}

	return fmt.Sprintf(
		"Assigned manually to %s. Current capacity: %d/%d active tickets.",
		agent.User.FullName,
		activeTickets,
		agent.MaxTickets,
	)
}

func deriveUpdatedSkillScores(current domain.SkillScoreList, requiredSkills domain.SkillNameList, priorityName string, slaBreached bool) domain.SkillScoreList {
	if len(requiredSkills) == 0 {
		return current
	}

	scoreMap := make(map[string]domain.SkillScore, len(current))
	for _, skill := range current {
		scoreMap[normalizeSkillKey(skill.Name)] = domain.SkillScore{
			Name:  canonicalSkillName(skill.Name),
			Score: skill.Score,
		}
	}

	demonstratedScore := 76.0
	switch strings.ToLower(strings.TrimSpace(priorityName)) {
	case "critical":
		demonstratedScore = 88
	case "high":
		demonstratedScore = 84
	case "medium":
		demonstratedScore = 80
	case "low":
		demonstratedScore = 72
	}
	if slaBreached {
		demonstratedScore -= 12
	} else {
		demonstratedScore += 4
	}
	if demonstratedScore < 40 {
		demonstratedScore = 40
	}

	for _, requiredSkill := range requiredSkills {
		key := normalizeSkillKey(requiredSkill)
		currentSkill, exists := scoreMap[key]
		if !exists {
			scoreMap[key] = domain.SkillScore{
				Name:  canonicalSkillName(requiredSkill),
				Score: int(math.Round(demonstratedScore)),
			}
			continue
		}

		updated := int(math.Round((float64(currentSkill.Score) * 0.75) + (demonstratedScore * 0.25)))
		if updated > 100 {
			updated = 100
		}
		scoreMap[key] = domain.SkillScore{
			Name:  currentSkill.Name,
			Score: updated,
		}
	}

	result := make(domain.SkillScoreList, 0, len(scoreMap))
	for _, skill := range scoreMap {
		result = append(result, skill)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

func fetchActiveTicketCounts(ctx context.Context, db *gorm.DB, agentIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	counts := make(map[uuid.UUID]int, len(agentIDs))
	if db == nil || len(agentIDs) == 0 {
		return counts, nil
	}

	type row struct {
		AgentID uuid.UUID `gorm:"column:assigned_to"`
		Total   int       `gorm:"column:total"`
	}

	var rows []row
	if err := db.WithContext(ctx).
		Table("tickets").
		Select("assigned_to, COUNT(*) AS total").
		Joins("JOIN ticket_statuses ON ticket_statuses.id = tickets.status_id").
		Where("assigned_to IN ?", agentIDs).
		Where("ticket_statuses.name IN ?", []string{"open", "in_progress"}).
		Group("assigned_to").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("fetchActiveTicketCounts: %w", err)
	}

	for _, row := range rows {
		counts[row.AgentID] = row.Total
	}
	return counts, nil
}

func normalizeSkillKey(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}

func canonicalSkillName(value string) string {
	normalized := normalizeSkillKey(value)
	if normalized == "" {
		return ""
	}

	for knownSkill := range skillKeywordMap {
		if normalizeSkillKey(knownSkill) == normalized {
			return knownSkill
		}
	}

	parts := strings.Fields(normalized)
	for index, part := range parts {
		if acronym, exists := skillAcronyms[part]; exists {
			parts[index] = acronym
			continue
		}
		parts[index] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}
