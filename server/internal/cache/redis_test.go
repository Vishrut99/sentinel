package cache

import "testing"

func TestNewRedisCacheDisabledValuesReturnNil(t *testing.T) {
	for _, value := range []string{"", "disabled", "off", "false", "none"} {
		cache, err := NewRedisCache(value)
		if err != nil {
			t.Fatalf("expected no error for value %q, got %v", value, err)
		}
		if cache != nil {
			t.Fatalf("expected nil cache for value %q", value)
		}
	}
}
