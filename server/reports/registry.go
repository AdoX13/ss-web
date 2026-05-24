// Package reports defines the pluggable report interface (P6 owns the logic;
// P1 exposes the routes). This file is owned jointly — P6 adds implementations
// to this package; P1 only touches routes/reports_routes.go.
package reports

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

// Params are the common query parameters every report accepts.
type Params struct {
	From   time.Time
	To     time.Time
	Format string // "json" | "csv"
}

// Row is a single data row returned by a report.
type Row map[string]any

// Result is the structured output of a report run.
type Result struct {
	Name    string
	Columns []string
	Rows    []Row
}

// Report is the interface every report implementation must satisfy.
type Report interface {
	Name() string
	Description() string
	// Roles lists which user roles may run this report (empty = all roles).
	Roles() []string
	Run(ctx context.Context, db *mongo.Database, p Params) (*Result, error)
}

// Registry holds all registered reports.
type Registry struct {
	byName map[string]Report
	list   []Report
}

// NewRegistry builds a registry pre-loaded with the provided reports.
func NewRegistry(reps ...Report) *Registry {
	r := &Registry{byName: make(map[string]Report, len(reps))}
	for _, rep := range reps {
		r.byName[rep.Name()] = rep
		r.list = append(r.list, rep)
	}
	return r
}

// List returns all reports in registration order.
func (r *Registry) List() []Report { return r.list }

// Get returns a report by name and whether it was found.
func (r *Registry) Get(name string) (Report, bool) {
	rep, ok := r.byName[name]
	return rep, ok
}

// ── Stub implementations — P6 replaces these with real MongoDB aggregations ──

type stubReport struct {
	name  string
	desc  string
	roles []string
}

func (s *stubReport) Name() string        { return s.name }
func (s *stubReport) Description() string { return s.desc }
func (s *stubReport) Roles() []string     { return s.roles }
func (s *stubReport) Run(_ context.Context, _ *mongo.Database, _ Params) (*Result, error) {
	return &Result{Name: s.name, Columns: []string{"status"}, Rows: []Row{{"status": "not yet implemented"}}}, nil
}

// DefaultRegistry returns a registry pre-loaded with R1–R6 stub reports.
// P6 should replace each stub with a real implementation using the same name.
func DefaultRegistry() *Registry {
	return NewRegistry(
		&stubReport{
			name:  "recent_exams",
			desc:  "Medical exams in the last 30 days",
			roles: []string{},
		},
		&stubReport{
			name:  "upcoming_expirations",
			desc:  "Exams expiring in the next 30 days",
			roles: []string{},
		},
		&stubReport{
			name:  "compliance_percentage",
			desc:  "Percentage of workers with valid medical clearance",
			roles: []string{},
		},
		&stubReport{
			name:  "anonymized_export",
			desc:  "Anonymized research export (k-anonymity ≥ 5)",
			roles: []string{"researcher", "admin"},
		},
		&stubReport{
			name:  "ocr_performance",
			desc:  "OCR latency and confidence statistics",
			roles: []string{"admin"},
		},
		&stubReport{
			name:  "review_queue_stats",
			desc:  "Review queue throughput and resolution times",
			roles: []string{"admin", "doctor"},
		},
	)
}
