package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct{ db *sql.DB }

// TimeEntry represents one tracked block of work.
// Duration is stored as integer seconds. Billable is 0 or 1.
type TimeEntry struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Project     string `json:"project"`
	Task        string `json:"task"`
	Duration    int    `json:"duration"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Billable    int    `json:"billable"`
	Tags        string `json:"tags"`
	CreatedAt   string `json:"created_at"`
}

func Open(d string) (*DB, error) {
	if err := os.MkdirAll(d, 0755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.Join(d, "sundial.db")+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	db.Exec(`CREATE TABLE IF NOT EXISTS time_entries(
		id TEXT PRIMARY KEY,
		description TEXT NOT NULL,
		project TEXT DEFAULT '',
		task TEXT DEFAULT '',
		duration INTEGER DEFAULT 0,
		start_time TEXT DEFAULT '',
		end_time TEXT DEFAULT '',
		billable INTEGER DEFAULT 1,
		tags TEXT DEFAULT '',
		created_at TEXT DEFAULT(datetime('now'))
	)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS extras(
		resource TEXT NOT NULL,
		record_id TEXT NOT NULL,
		data TEXT NOT NULL DEFAULT '{}',
		PRIMARY KEY(resource, record_id)
	)`)
	return &DB{db: db}, nil
}

func (d *DB) Close() error { return d.db.Close() }

func genID() string { return fmt.Sprintf("%d", time.Now().UnixNano()) }
func now() string   { return time.Now().UTC().Format(time.RFC3339) }

func (d *DB) Create(e *TimeEntry) error {
	e.ID = genID()
	e.CreatedAt = now()
	_, err := d.db.Exec(
		`INSERT INTO time_entries(id, description, project, task, duration, start_time, end_time, billable, tags, created_at)
		 VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.Description, e.Project, e.Task, e.Duration, e.StartTime, e.EndTime, e.Billable, e.Tags, e.CreatedAt,
	)
	return err
}

func (d *DB) Get(id string) *TimeEntry {
	var e TimeEntry
	err := d.db.QueryRow(
		`SELECT id, description, project, task, duration, start_time, end_time, billable, tags, created_at
		 FROM time_entries WHERE id=?`,
		id,
	).Scan(&e.ID, &e.Description, &e.Project, &e.Task, &e.Duration, &e.StartTime, &e.EndTime, &e.Billable, &e.Tags, &e.CreatedAt)
	if err != nil {
		return nil
	}
	return &e
}

func (d *DB) List() []TimeEntry {
	rows, _ := d.db.Query(
		`SELECT id, description, project, task, duration, start_time, end_time, billable, tags, created_at
		 FROM time_entries
		 ORDER BY CASE WHEN start_time = '' THEN 1 ELSE 0 END, start_time DESC, created_at DESC`,
	)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var o []TimeEntry
	for rows.Next() {
		var e TimeEntry
		rows.Scan(&e.ID, &e.Description, &e.Project, &e.Task, &e.Duration, &e.StartTime, &e.EndTime, &e.Billable, &e.Tags, &e.CreatedAt)
		o = append(o, e)
	}
	return o
}

func (d *DB) Update(e *TimeEntry) error {
	_, err := d.db.Exec(
		`UPDATE time_entries SET description=?, project=?, task=?, duration=?, start_time=?, end_time=?, billable=?, tags=?
		 WHERE id=?`,
		e.Description, e.Project, e.Task, e.Duration, e.StartTime, e.EndTime, e.Billable, e.Tags, e.ID,
	)
	return err
}

func (d *DB) Delete(id string) error {
	_, err := d.db.Exec(`DELETE FROM time_entries WHERE id=?`, id)
	return err
}

func (d *DB) Count() int {
	var n int
	d.db.QueryRow(`SELECT COUNT(*) FROM time_entries`).Scan(&n)
	return n
}

func (d *DB) Search(q string, filters map[string]string) []TimeEntry {
	where := "1=1"
	args := []any{}
	if q != "" {
		where += " AND (description LIKE ? OR project LIKE ? OR task LIKE ? OR tags LIKE ?)"
		args = append(args, "%"+q+"%", "%"+q+"%", "%"+q+"%", "%"+q+"%")
	}
	if v, ok := filters["project"]; ok && v != "" {
		where += " AND project=?"
		args = append(args, v)
	}
	if v, ok := filters["billable"]; ok && v != "" {
		if v == "yes" {
			where += " AND billable=1"
		} else if v == "no" {
			where += " AND billable=0"
		}
	}
	rows, _ := d.db.Query(
		`SELECT id, description, project, task, duration, start_time, end_time, billable, tags, created_at
		 FROM time_entries WHERE `+where+`
		 ORDER BY CASE WHEN start_time = '' THEN 1 ELSE 0 END, start_time DESC, created_at DESC`,
		args...,
	)
	if rows == nil {
		return nil
	}
	defer rows.Close()
	var o []TimeEntry
	for rows.Next() {
		var e TimeEntry
		rows.Scan(&e.ID, &e.Description, &e.Project, &e.Task, &e.Duration, &e.StartTime, &e.EndTime, &e.Billable, &e.Tags, &e.CreatedAt)
		o = append(o, e)
	}
	return o
}

// Stats returns total entries, total seconds, billable seconds, this-week
// seconds, and a per-project breakdown. Used by the dashboard cards and
// the project filter.
func (d *DB) Stats() map[string]any {
	m := map[string]any{
		"total":            d.Count(),
		"total_seconds":    0,
		"billable_seconds": 0,
		"week_seconds":     0,
		"by_project":       map[string]int{},
	}

	var totalSeconds int
	d.db.QueryRow(`SELECT COALESCE(SUM(duration), 0) FROM time_entries`).Scan(&totalSeconds)
	m["total_seconds"] = totalSeconds

	var billableSeconds int
	d.db.QueryRow(`SELECT COALESCE(SUM(duration), 0) FROM time_entries WHERE billable=1`).Scan(&billableSeconds)
	m["billable_seconds"] = billableSeconds

	// "This week" = last 7 days from now (rolling).
	weekAgo := time.Now().AddDate(0, 0, -7).Format(time.RFC3339)
	var weekSeconds int
	d.db.QueryRow(
		`SELECT COALESCE(SUM(duration), 0) FROM time_entries
		 WHERE (start_time >= ? OR (start_time = '' AND created_at >= ?))`,
		weekAgo, weekAgo,
	).Scan(&weekSeconds)
	m["week_seconds"] = weekSeconds

	if rows, _ := d.db.Query(`SELECT project, SUM(duration) FROM time_entries WHERE project != '' GROUP BY project ORDER BY SUM(duration) DESC`); rows != nil {
		defer rows.Close()
		by := map[string]int{}
		for rows.Next() {
			var p string
			var s int
			rows.Scan(&p, &s)
			by[p] = s
		}
		m["by_project"] = by
	}

	return m
}

// ─── Extras: generic key-value storage for personalization custom fields ───

func (d *DB) GetExtras(resource, recordID string) string {
	var data string
	err := d.db.QueryRow(
		`SELECT data FROM extras WHERE resource=? AND record_id=?`,
		resource, recordID,
	).Scan(&data)
	if err != nil || data == "" {
		return "{}"
	}
	return data
}

func (d *DB) SetExtras(resource, recordID, data string) error {
	if data == "" {
		data = "{}"
	}
	_, err := d.db.Exec(
		`INSERT INTO extras(resource, record_id, data) VALUES(?, ?, ?)
		 ON CONFLICT(resource, record_id) DO UPDATE SET data=excluded.data`,
		resource, recordID, data,
	)
	return err
}

func (d *DB) DeleteExtras(resource, recordID string) error {
	_, err := d.db.Exec(
		`DELETE FROM extras WHERE resource=? AND record_id=?`,
		resource, recordID,
	)
	return err
}

func (d *DB) AllExtras(resource string) map[string]string {
	out := make(map[string]string)
	rows, _ := d.db.Query(
		`SELECT record_id, data FROM extras WHERE resource=?`,
		resource,
	)
	if rows == nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, data string
		rows.Scan(&id, &data)
		out[id] = data
	}
	return out
}
