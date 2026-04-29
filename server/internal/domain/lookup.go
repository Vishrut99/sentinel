package domain

import "github.com/google/uuid"

type CategoryLookupRes struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
}

type PriorityLookupRes struct {
	ID             int16  `json:"id"`
	Name           string `json:"name"`
	SLAResponseHrs int    `json:"sla_response_hrs"`
	SLAResolveHrs  int    `json:"sla_resolve_hrs"`
}

type StatusLookupRes struct {
	ID   int16  `json:"id"`
	Name string `json:"name"`
	Code string `json:"code,omitempty"`
}
