package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// SkillScore stores an agent skill and its proficiency score.
type SkillScore struct {
	Name  string `json:"name"`
	Score int    `json:"score"`
}

// SkillScoreList persists as jsonb in Postgres.
type SkillScoreList []SkillScore

func (s SkillScoreList) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "[]", nil
	}

	encoded, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return string(encoded), nil
}

func (s *SkillScoreList) Scan(value any) error {
	if s == nil {
		return fmt.Errorf("SkillScoreList.Scan: nil receiver")
	}

	switch typed := value.(type) {
	case nil:
		*s = SkillScoreList{}
		return nil
	case []byte:
		if len(typed) == 0 {
			*s = SkillScoreList{}
			return nil
		}
		return json.Unmarshal(typed, s)
	case string:
		if typed == "" {
			*s = SkillScoreList{}
			return nil
		}
		return json.Unmarshal([]byte(typed), s)
	default:
		return fmt.Errorf("SkillScoreList.Scan: unsupported type %T", value)
	}
}

// SkillNameList persists a list of required ticket skills as jsonb.
type SkillNameList []string

func (s SkillNameList) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "[]", nil
	}

	encoded, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return string(encoded), nil
}

func (s *SkillNameList) Scan(value any) error {
	if s == nil {
		return fmt.Errorf("SkillNameList.Scan: nil receiver")
	}

	switch typed := value.(type) {
	case nil:
		*s = SkillNameList{}
		return nil
	case []byte:
		if len(typed) == 0 {
			*s = SkillNameList{}
			return nil
		}
		return json.Unmarshal(typed, s)
	case string:
		if typed == "" {
			*s = SkillNameList{}
			return nil
		}
		return json.Unmarshal([]byte(typed), s)
	default:
		return fmt.Errorf("SkillNameList.Scan: unsupported type %T", value)
	}
}
