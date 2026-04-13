package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/stockyard-dev/stockyard-sundial/internal/store"
	"github.com/stockyard-dev/stockyard/bus"
)

// resourceName is the canonical key for extras storage and the API path.
const resourceName = "time_entries"

type Server struct {
	db      *store.DB
	mux     *http.ServeMux
	limMu   sync.RWMutex // guards limits, which can be hot-reloaded by /api/license/activate
	limits  Limits
	dataDir string
	pCfg    map[string]json.RawMessage
	bus     *bus.Bus // optional cross-tool event bus; nil if not configured
}

func New(db *store.DB, limits Limits, dataDir string, b *bus.Bus) *Server {
	s := &Server{
		db:      db,
		mux:     http.NewServeMux(),
		limits:  limits,
		dataDir: dataDir,
		bus:     b,
	}
	s.loadPersonalConfig()

	// Time entry CRUD
	s.mux.HandleFunc("GET /api/time_entries", s.list)
	s.mux.HandleFunc("POST /api/time_entries", s.create)
	s.mux.HandleFunc("GET /api/time_entries/{id}", s.get)
	s.mux.HandleFunc("PUT /api/time_entries/{id}", s.update)
	s.mux.HandleFunc("DELETE /api/time_entries/{id}", s.del)

	// Stats / health
	s.mux.HandleFunc("GET /api/stats", s.stats)
	s.mux.HandleFunc("GET /api/health", s.health)

	// Personalization
	s.mux.HandleFunc("GET /api/config", s.configHandler)

	// Extras (custom fields)
	s.mux.HandleFunc("GET /api/extras/{resource}", s.listExtras)
	s.mux.HandleFunc("GET /api/extras/{resource}/{id}", s.getExtras)
	s.mux.HandleFunc("PUT /api/extras/{resource}/{id}", s.putExtras)

	// License activation — accepts a key, validates, persists, hot-reloads tier
	s.mux.HandleFunc("POST /api/license/activate", s.activateLicense)

	// Dashboard
	s.mux.HandleFunc("GET /ui", s.dashboard)
	s.mux.HandleFunc("GET /ui/", s.dashboard)
	s.mux.HandleFunc("GET /", s.root)

	// Tier — read-only license info for dashboard banner. Always reachable.
	s.mux.HandleFunc("GET /api/tier", s.tierInfo)

	s.subscribeBus()
	return s
}

// ServeHTTP wraps the underlying mux with a license-gate middleware.
// In trial-required mode, all writes (POST/PUT/DELETE/PATCH) return 402
// EXCEPT POST /api/license/activate (the only way out of trial state).
// Reads are always allowed — the brand promise is that data on disk
// stays accessible even without an active license.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.shouldBlockWrite(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusPaymentRequired)
		w.Write([]byte(`{"error":"Trial required. Start a 14-day free trial at https://stockyard.dev/ — or paste an existing license key in the dashboard under \"Activate License\".","tier":"trial-required"}`))
		return
	}
	s.mux.ServeHTTP(w, r)
}

func (s *Server) shouldBlockWrite(r *http.Request) bool {
	s.limMu.RLock()
	tier := s.limits.Tier
	s.limMu.RUnlock()
	if tier != "trial-required" {
		return false
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	}
	switch r.URL.Path {
	case "/api/license/activate":
		return false
	}
	return true
}

func (s *Server) activateLicense(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 10*1024))
	if err != nil {
		we(w, 400, "could not read request body")
		return
	}
	var req struct {
		LicenseKey string `json:"license_key"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		we(w, 400, "invalid json: "+err.Error())
		return
	}
	key := strings.TrimSpace(req.LicenseKey)
	if key == "" {
		we(w, 400, "license_key is required")
		return
	}
	if !ValidateLicenseKey(key) {
		we(w, 400, "license key is not valid for this product — make sure you copied the entire key from the welcome email, including the SY- prefix")
		return
	}
	if err := PersistLicense(s.dataDir, key); err != nil {
		log.Printf("sundial: license persist failed: %v", err)
		we(w, 500, "could not save the license key to disk: "+err.Error())
		return
	}
	s.limMu.Lock()
	s.limits = ProLimits()
	s.limMu.Unlock()
	log.Printf("sundial: license activated via dashboard, persisted to %s/%s", s.dataDir, licenseFilename)
	wj(w, 200, map[string]any{
		"ok":   true,
		"tier": "pro",
	})
}

func (s *Server) tierInfo(w http.ResponseWriter, r *http.Request) {
	s.limMu.RLock()
	tier := s.limits.Tier
	s.limMu.RUnlock()
	resp := map[string]any{
		"tier": tier,
	}
	if tier == "trial-required" {
		resp["trial_required"] = true
		resp["start_trial_url"] = "https://stockyard.dev/"
		resp["message"] = "Your trial is not active. Reads work, but you cannot add or change time entries until you start a 14-day trial or activate an existing license key."
	} else {
		resp["trial_required"] = false
	}
	wj(w, 200, resp)
}

// ─── helpers ──────────────────────────────────────────────────────

func wj(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func we(w http.ResponseWriter, code int, msg string) {
	wj(w, code, map[string]string{"error": msg})
}

func oe[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func (s *Server) root(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/ui", 302)
}

// ─── personalization ──────────────────────────────────────────────

func (s *Server) loadPersonalConfig() {
	path := filepath.Join(s.dataDir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var cfg map[string]json.RawMessage
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("sundial: warning: could not parse config.json: %v", err)
		return
	}
	s.pCfg = cfg
	log.Printf("sundial: loaded personalization from %s", path)
}

func (s *Server) configHandler(w http.ResponseWriter, r *http.Request) {
	if s.pCfg == nil {
		wj(w, 200, map[string]any{})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.pCfg)
}

// ─── extras ───────────────────────────────────────────────────────

func (s *Server) listExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	all := s.db.AllExtras(resource)
	out := make(map[string]json.RawMessage, len(all))
	for id, data := range all {
		out[id] = json.RawMessage(data)
	}
	wj(w, 200, out)
}

func (s *Server) getExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	id := r.PathValue("id")
	data := s.db.GetExtras(resource, id)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(data))
}

func (s *Server) putExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	id := r.PathValue("id")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		we(w, 400, "read body")
		return
	}
	var probe map[string]any
	if err := json.Unmarshal(body, &probe); err != nil {
		we(w, 400, "invalid json")
		return
	}
	if err := s.db.SetExtras(resource, id, string(body)); err != nil {
		we(w, 500, "save failed")
		return
	}
	wj(w, 200, map[string]string{"ok": "saved"})
}

// ─── time entry CRUD ──────────────────────────────────────────────

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	filters := map[string]string{}
	if v := r.URL.Query().Get("project"); v != "" {
		filters["project"] = v
	}
	if v := r.URL.Query().Get("billable"); v != "" {
		filters["billable"] = v
	}
	if q != "" || len(filters) > 0 {
		wj(w, 200, map[string]any{"time_entries": oe(s.db.Search(q, filters))})
		return
	}
	wj(w, 200, map[string]any{"time_entries": oe(s.db.List())})
}

func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	var e store.TimeEntry
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		we(w, 400, "invalid json")
		return
	}
	if e.Description == "" {
		we(w, 400, "description required")
		return
	}
	if err := s.db.Create(&e); err != nil {
		we(w, 500, "create failed")
		return
	}
	created := s.db.Get(e.ID)
	s.publishTimeLogged(created)
	wj(w, 201, created)
}

func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	e := s.db.Get(r.PathValue("id"))
	if e == nil {
		we(w, 404, "not found")
		return
	}
	wj(w, 200, e)
}

// update accepts a full or partial TimeEntry. Empty string fields are
// preserved from the existing record. Duration=0 is preserved (it almost
// certainly means "field not sent" rather than "this entry took zero time").
// Billable is special: it comes through as 0 by default for omitted fields,
// so we have no way to distinguish "set to false" from "not sent" — we
// always trust the incoming value here, since the dashboard always sends it.
func (s *Server) update(w http.ResponseWriter, r *http.Request) {
	existing := s.db.Get(r.PathValue("id"))
	if existing == nil {
		we(w, 404, "not found")
		return
	}
	var patch store.TimeEntry
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		we(w, 400, "invalid json")
		return
	}
	patch.ID = existing.ID
	patch.CreatedAt = existing.CreatedAt
	if patch.Description == "" {
		patch.Description = existing.Description
	}
	if patch.Project == "" {
		patch.Project = existing.Project
	}
	if patch.Task == "" {
		patch.Task = existing.Task
	}
	if patch.Duration == 0 {
		patch.Duration = existing.Duration
	}
	if patch.StartTime == "" {
		patch.StartTime = existing.StartTime
	}
	if patch.EndTime == "" {
		patch.EndTime = existing.EndTime
	}
	if patch.Tags == "" {
		patch.Tags = existing.Tags
	}
	if err := s.db.Update(&patch); err != nil {
		we(w, 500, "update failed")
		return
	}
	wj(w, 200, s.db.Get(patch.ID))
}

func (s *Server) del(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.db.Delete(id)
	s.db.DeleteExtras(resourceName, id)
	wj(w, 200, map[string]string{"deleted": "ok"})
}

// ─── stats / health ───────────────────────────────────────────────

func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	wj(w, 200, s.db.Stats())
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	wj(w, 200, map[string]any{
		"status":       "ok",
		"service":      "sundial",
		"time_entries": s.db.Count(),
	})
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

// publishTimeLogged fires time.logged on the bus. No-op when bus is
// nil (standalone). Runs in a goroutine; errors logged not surfaced.
// Payload shape locked by docs/BUS-TOPICS.md v1 in stockyard-desktop.
//
// Reality note: sundial tracks time against Projects (free-text),
// not Contacts. There is no contact_id in the payload. Subscribers
// that want contact linkage must either (a) fuzzy-match project
// name against a contact or (b) wait for sundial to grow a dossier
// relation.
func (s *Server) publishTimeLogged(e *store.TimeEntry) {
	if s.bus == nil || e == nil {
		return
	}
	payload := map[string]any{
		"entry_id":         e.ID,
		"description":      e.Description,
		"project":          e.Project,
		"task":             e.Task,
		"duration_seconds": e.Duration,
		"start_time":       e.StartTime,
		"end_time":         e.EndTime,
		"billable":         e.Billable == 1,
		"tags":             e.Tags,
	}
	go func() {
		if _, err := s.bus.Publish("time.logged", payload); err != nil {
			log.Printf("sundial: bus publish time.logged failed: %v", err)
		}
	}()
}

// subscribeBus wires cross-tool events to auto-logged time entries.
// No-op when s.bus is nil (standalone mode).
//
// Allowlist-only (not SubscribeAll) so unexpected future topics can't
// silently start creating time entries. Expanding this list is a
// PR-reviewed change — every addition is "a new way time entries
// appear without the user clicking New Entry."
//
// Idempotency: the appt:<id> marker is written to the entry's Tags
// field and scanned on each fire. Bus cursor initializes at the
// current high-water mark on Open (bus.go:199), so process restart
// does NOT replay old events — we only dedup duplicate fires within
// a single bundle lifetime.
//
// Handlers return nil on decode/data errors. The bus has no automatic
// retry (bus.go Handler docstring).
func (s *Server) subscribeBus() {
	if s.bus == nil {
		return
	}
	s.bus.Subscribe("appointment.completed", func(_ context.Context, e bus.Event) error {
		return s.handleAppointmentCompleted(e)
	})
	log.Printf("sundial: subscribed to appointment.completed")
}

// handleAppointmentCompleted auto-logs a time entry when an
// appointment transitions to completed.
//
// Shape decisions (see BUS-TOPICS.md):
//   - Description = "Appointment: <service>" (or just "Appointment"
//     if service is empty).
//   - Project = client_name. Sundial's Project is free text, matches
//     billfold's client_name field shape exactly, so the downstream
//     time.logged → billfold line-item path naturally chains.
//   - Task = service (free text from booking).
//   - Duration = 3600 seconds (1 hour default). Booking's payload
//     carries no duration concept; 1h is the least-bad default for a
//     service appointment. User can edit in the sundial UI. This is
//     the most opinionated default in the whole subscriber web and
//     will likely get revisited once booking grows duration.
//   - StartTime = "<date> <time>" concatenated from payload. Booking
//     stores date/time as separate free-text strings with no timezone
//     declared; we preserve that reality rather than fabricating
//     RFC3339 precision.
//   - EndTime = "" (can't compute without a real duration concept).
//   - Billable = 1. Appointments are almost always billable work;
//     user can flip it in the UI for the rare exception.
//   - Tags = "appointment appt:<appointment_id>". The marker is what
//     makes the handler idempotent.
func (s *Server) handleAppointmentCompleted(e bus.Event) error {
	var p map[string]any
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		log.Printf("sundial: decode appointment.completed: %v", err)
		return nil
	}
	apptID := stringField(p, "appointment_id")
	if apptID == "" {
		log.Printf("sundial: appointment.completed missing appointment_id, skipping")
		return nil
	}
	marker := "appt:" + apptID
	// Idempotency: scan existing entries for this marker in Tags.
	for _, existing := range s.db.List() {
		if strings.Contains(existing.Tags, marker) {
			log.Printf("sundial: appointment %s already auto-logged as entry %s, skipping",
				apptID, existing.ID)
			return nil
		}
	}
	service := strings.TrimSpace(stringField(p, "service"))
	desc := "Appointment"
	if service != "" {
		desc = "Appointment: " + service
	}
	clientName := stringField(p, "client_name")
	date := stringField(p, "date")
	timeStr := stringField(p, "time")
	startTime := strings.TrimSpace(date + " " + timeStr)
	entry := store.TimeEntry{
		Description: desc,
		Project:     clientName,
		Task:        service,
		Duration:    3600,
		StartTime:   startTime,
		EndTime:     "",
		Billable:    1,
		Tags:        fmt.Sprintf("appointment %s", marker),
	}
	if err := s.db.Create(&entry); err != nil {
		log.Printf("sundial: create time entry for appointment %s: %v", apptID, err)
		return nil
	}
	log.Printf("sundial: auto-logged appointment %s as time entry %s (client=%q service=%q)",
		apptID, entry.ID, clientName, service)
	// NOTE: we deliberately do NOT call s.publishTimeLogged here. The
	// downstream billfold subscriber keys on `project` = `client_name`
	// which we DO set above — but firing time.logged from an
	// appointment.completed handler creates a cross-tool chain
	// (booking → sundial → billfold) that can surprise users. Leave
	// re-publish opt-in via the sundial UI for now; revisit if the
	// chained flow becomes explicitly desired.
	return nil
}

// stringField returns m[k] as a string, or "" if absent / wrong type.
func stringField(m map[string]any, k string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return ""
}
