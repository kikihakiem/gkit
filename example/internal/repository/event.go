package repository

import (
	"os"

	"github.com/kikihakiem/gkit/example/internal/audit"
	"github.com/nokusukun/bingo"
)

type EventRepository struct {
	db *bingo.Collection[Event]
}

type Event struct {
	bingo.Document
	audit.Event
}

func NewEventRepositroy() (*EventRepository, func()) {
	driver, err := bingo.NewDriver(bingo.DriverConfiguration{
		Filename: "events.db",
	})
	if err != nil {
		panic(err)
	}

	return &EventRepository{
			db: bingo.CollectionFrom[Event](driver, "events"),
		}, func() {
			os.Remove("events.db")
		}
}

func (e *EventRepository) Create(event audit.Event) (id string, err error) {
	key, err := e.db.Insert(Event{Event: event})
	if err != nil {
		return "", err
	}

	return string(key), nil
}

func (e *EventRepository) GetList() ([]audit.Event, error) {
	events, err := e.db.Find(func(doc Event) bool {
		return true // return all.
	})
	if err != nil {
		return nil, err
	}

	return Map(events, func(e Event) audit.Event {
		return e.Event
	}), nil
}

func Map[In, Out any](ins []In, mapperFunc func(In) Out) []Out {
	outs := make([]Out, 0, len(ins))

	for _, in := range ins {
		outs = append(outs, mapperFunc(in))
	}

	return outs
}
