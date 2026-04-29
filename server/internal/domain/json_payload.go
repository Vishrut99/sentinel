package domain

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JSONPayload stores raw JSON in Postgres json/jsonb columns without being treated as bytea.
type JSONPayload []byte

func (j JSONPayload) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	if !json.Valid(j) {
		return nil, fmt.Errorf("JSONPayload.Value: invalid JSON payload")
	}

	return string(j), nil
}

func (j *JSONPayload) Scan(value any) error {
	if j == nil {
		return fmt.Errorf("JSONPayload.Scan: nil receiver")
	}

	switch typed := value.(type) {
	case nil:
		*j = nil
		return nil
	case []byte:
		if len(typed) == 0 {
			*j = nil
			return nil
		}
		*j = append((*j)[:0], typed...)
		return nil
	case string:
		if typed == "" {
			*j = nil
			return nil
		}
		*j = append((*j)[:0], typed...)
		return nil
	default:
		return fmt.Errorf("JSONPayload.Scan: unsupported type %T", value)
	}
}

func (j JSONPayload) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	if !json.Valid(j) {
		return nil, fmt.Errorf("JSONPayload.MarshalJSON: invalid JSON payload")
	}

	return append([]byte(nil), j...), nil
}

func (j *JSONPayload) UnmarshalJSON(data []byte) error {
	if j == nil {
		return fmt.Errorf("JSONPayload.UnmarshalJSON: nil receiver")
	}
	if len(data) == 0 {
		*j = nil
		return nil
	}
	if !json.Valid(data) {
		return fmt.Errorf("JSONPayload.UnmarshalJSON: invalid JSON payload")
	}

	*j = append((*j)[:0], data...)
	return nil
}
