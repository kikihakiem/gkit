package audit

import "time"

// Model.

type Action string

const (
	ActionRead   Action = "read"
	ActionCreate Action = "create"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
)

type Actor struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
}

type Change struct {
	Entity string `json:"entity"`
	Old    string `json:"old"`
	New    string `json:"new"`
}

type Event struct {
	Actor     Actor     `json:"actor"`
	Action    Action    `json:"action"`
	Changes   []Change  `json:"changes"`
	Timestamp time.Time `json:"timestamp"`
}

// Request-response.

type CreateEventRequest struct {
	Event
}

type CreateEventResponse struct {
	EventID string `json:"event_id"`
}

type GetEventListRequest struct{}

type GetEventListResponse []Event
