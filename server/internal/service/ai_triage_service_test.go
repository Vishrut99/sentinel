package service

import "testing"

func TestNewAITriageServiceDefaultsToCurrentGeminiModel(t *testing.T) {
	service, ok := NewAITriageService("", "test-api-key").(*aiTriageService)
	if !ok || service == nil {
		t.Fatalf("expected concrete aiTriageService, got %T", service)
	}

	if service.url != defaultGeminiTriageURL {
		t.Fatalf("expected default gemini URL %q, got %q", defaultGeminiTriageURL, service.url)
	}
	if !service.useGemini {
		t.Fatalf("expected Gemini mode to be enabled")
	}
}

func TestBuildGeminiCandidateURLsIncludesCurrentFallbacks(t *testing.T) {
	legacyURL := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent"

	candidates := buildGeminiCandidateURLs(legacyURL)

	if len(candidates) != 4 {
		t.Fatalf("expected 4 candidate URLs, got %d", len(candidates))
	}
	if candidates[0] != legacyURL {
		t.Fatalf("expected legacy URL to be tried first, got %q", candidates[0])
	}
	if candidates[1] != defaultGeminiTriageURL {
		t.Fatalf("expected default URL second, got %q", candidates[1])
	}
	if candidates[2] != fallbackGeminiTriageURLs[0] || candidates[3] != fallbackGeminiTriageURLs[1] {
		t.Fatalf("unexpected fallback URLs: %#v", candidates)
	}
}
