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
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Rows    []Row    `json:"rows"`
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

// DefaultRegistry returns the P6 report set (R1-R6).
func DefaultRegistry() *Registry {
	return NewRegistry(
		recentExamsReport{},
		upcomingExpirationsReport{},
		complianceReport{},
		anonymizedExportReport{},
		ocrPerformanceReport{},
		reviewQueueStatsReport{},
	)
}
