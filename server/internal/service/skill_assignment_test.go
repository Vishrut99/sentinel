package service

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/yourusername/incident-ticketing/internal/domain"
)

func TestResolveRequiredSkillsMergesExplicitAIAndHeuristicSources(t *testing.T) {
	req := domain.CreateTicketReq{
		Title:          "VPN login blocked after password reset",
		Description:    "Remote access keeps failing and user cannot sign in after the reset",
		RequiredSkills: []string{"Database"},
	}
	aiInsights := json.RawMessage(`{"required_skills":["Network","VPN"]}`)

	requiredSkills := resolveRequiredSkills(req, aiInsights)

	expected := map[string]bool{
		"Database":       true,
		"Network":        true,
		"VPN":            true,
		"Account Access": true,
	}
	for skill := range expected {
		found := false
		for _, candidate := range requiredSkills {
			if candidate == skill {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected required skill %q in %#v", skill, requiredSkills)
		}
	}
}

func TestPickBestAgentForTicketPrefersSkillMatch(t *testing.T) {
	vpnAgentID := uuid.New()
	generalAgentID := uuid.New()
	agents := []domain.Agent{
		{
			ID:         vpnAgentID,
			MaxTickets: 5,
			User:       domain.User{FullName: "VPN Agent"},
			Skills: domain.SkillScoreList{
				{Name: "VPN", Score: 92},
				{Name: "Network", Score: 80},
			},
			IsAvailable: true,
		},
		{
			ID:         generalAgentID,
			MaxTickets: 5,
			User:       domain.User{FullName: "General Agent"},
			Skills: domain.SkillScoreList{
				{Name: "Application Support", Score: 88},
			},
			IsAvailable: true,
		},
	}

	selectedAgent, justification := pickBestAgentForTicket(agents, map[uuid.UUID]int{
		vpnAgentID:     2,
		generalAgentID: 0,
	}, domain.SkillNameList{"VPN"})

	if selectedAgent == nil {
		t.Fatalf("expected an agent to be selected")
	}
	if selectedAgent.ID != vpnAgentID {
		t.Fatalf("expected VPN agent to be selected, got %s", selectedAgent.ID)
	}
	if justification == "" {
		t.Fatalf("expected assignment justification to be generated")
	}
}

func TestDeriveUpdatedSkillScoresRaisesRequiredSkillAfterSuccessfulResolution(t *testing.T) {
	updated := deriveUpdatedSkillScores(domain.SkillScoreList{
		{Name: "VPN", Score: 60},
	}, domain.SkillNameList{"VPN", "Network"}, "high", false)

	scores := map[string]int{}
	for _, skill := range updated {
		scores[skill.Name] = skill.Score
	}

	if scores["VPN"] <= 60 {
		t.Fatalf("expected VPN score to increase, got %d", scores["VPN"])
	}
	if scores["Network"] == 0 {
		t.Fatalf("expected missing required skill to be added, got %#v", updated)
	}
}
