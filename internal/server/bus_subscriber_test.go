package server

// Tests for the bus subscriber: appointment.completed → auto-logged
// time entry. Handler invoked directly with synthesized bus.Event
// payloads. End-to-end wiring (real bus, real publish, poll dispatch)
// lives in stockyard-desktop's orchestrator tests.

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stockyard-dev/stockyard-sundial/internal/store"
	"github.com/stockyard-dev/stockyard/bus"
)

func newSubscriberServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "data"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	// bus=nil — subscribeBus() is not exercised; we call handlers directly.
	return New(db, ProLimits(), dir, nil)
}

func eventWith(topic, source string, payload map[string]any) bus.Event {
	raw, _ := json.Marshal(payload)
	return bus.Event{Topic: topic, Source: source, Payload: raw}
}

func TestHandleAppointmentCompleted_CreatesTimeEntry(t *testing.T) {
	s := newSubscriberServer(t)
	e := eventWith("appointment.completed", "booking", map[string]any{
		"appointment_id": "a-100",
		"client_name":    "Acme Yoga",
		"service":        "Deep tissue massage",
		"date":           "2026-04-13",
		"time":           "14:00",
		"status":         "completed",
	})
	if err := s.handleAppointmentCompleted(e); err != nil {
		t.Fatalf("handler: %v", err)
	}
	list := s.db.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 time entry, got %d", len(list))
	}
	entry := list[0]
	if entry.Description != "Appointment: Deep tissue massage" {
		t.Errorf("description = %q, want %q", entry.Description, "Appointment: Deep tissue massage")
	}
	if entry.Project != "Acme Yoga" {
		t.Errorf("project = %q, want Acme Yoga (billfold matches on this)", entry.Project)
	}
	if entry.Task != "Deep tissue massage" {
		t.Errorf("task = %q, want Deep tissue massage", entry.Task)
	}
	if entry.Duration != 3600 {
		t.Errorf("duration = %d, want 3600 (1h default)", entry.Duration)
	}
	if entry.Billable != 1 {
		t.Errorf("billable = %d, want 1", entry.Billable)
	}
	if entry.StartTime != "2026-04-13 14:00" {
		t.Errorf("start_time = %q, want %q", entry.StartTime, "2026-04-13 14:00")
	}
	if !strings.Contains(entry.Tags, "appt:a-100") {
		t.Errorf("tags missing idempotency marker: %q", entry.Tags)
	}
}

func TestHandleAppointmentCompleted_IsIdempotent(t *testing.T) {
	s := newSubscriberServer(t)
	e := eventWith("appointment.completed", "booking", map[string]any{
		"appointment_id": "a-dup",
		"client_name":    "Dupe Co",
		"service":        "Consult",
		"date":           "2026-04-13",
		"time":           "10:00",
	})
	_ = s.handleAppointmentCompleted(e)
	_ = s.handleAppointmentCompleted(e)
	_ = s.handleAppointmentCompleted(e)
	if n := len(s.db.List()); n != 1 {
		t.Errorf("expected 1 entry after 3 fires, got %d", n)
	}
}

func TestHandleAppointmentCompleted_MissingIDSkips(t *testing.T) {
	s := newSubscriberServer(t)
	e := eventWith("appointment.completed", "booking", map[string]any{
		"client_name": "No ID",
		"service":     "Thing",
	})
	_ = s.handleAppointmentCompleted(e)
	if n := len(s.db.List()); n != 0 {
		t.Errorf("expected 0 entries, got %d (should skip missing appointment_id)", n)
	}
}

func TestHandleAppointmentCompleted_EmptyServiceFallback(t *testing.T) {
	s := newSubscriberServer(t)
	e := eventWith("appointment.completed", "booking", map[string]any{
		"appointment_id": "a-noservice",
		"client_name":    "Bare Co",
		"date":           "2026-04-13",
		"time":           "09:00",
	})
	_ = s.handleAppointmentCompleted(e)
	list := s.db.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(list))
	}
	if list[0].Description != "Appointment" {
		t.Errorf("description = %q, want %q (fallback when service empty)",
			list[0].Description, "Appointment")
	}
}

func TestHandleAppointmentCompleted_MalformedPayloadSkips(t *testing.T) {
	s := newSubscriberServer(t)
	e := bus.Event{Topic: "appointment.completed", Source: "booking", Payload: []byte("not json")}
	if err := s.handleAppointmentCompleted(e); err != nil {
		t.Fatalf("handler should not return error on malformed payload: %v", err)
	}
	if n := len(s.db.List()); n != 0 {
		t.Errorf("expected 0 entries after malformed payload, got %d", n)
	}
}
