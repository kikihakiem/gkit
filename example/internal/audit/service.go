package audit

import (
	"context"
)

type EventRepository interface {
	Create(Event) (id string, err error)
	GetList() ([]Event, error)
}

type EventService struct {
	repo EventRepository
}

func (s *EventService) CreateEvent(ctx context.Context, req CreateEventRequest) (CreateEventResponse, error) {
	id, err := s.repo.Create(req.Event)
	if err != nil {
		return CreateEventResponse{}, err
	}

	return CreateEventResponse{EventID: id}, nil
}

func (s *EventService) GetList(ctx context.Context, req GetEventListRequest) (GetEventListResponse, error) {
	events, err := s.repo.GetList()
	if err != nil {
		return GetEventListResponse{}, err
	}

	return GetEventListResponse(events), nil
}

func NewEventService(c EventRepository) *EventService {
	return &EventService{repo: c}
}
