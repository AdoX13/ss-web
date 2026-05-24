package routes

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"

	"mqtt-streaming-server/auth"
	"mqtt-streaming-server/domain"
	"mqtt-streaming-server/reports"
)

type reportsController struct {
	registry *reports.Registry
	db       *mongo.Database
	secret   string
}

func initReportRoutes(cfg *Config, mux *http.ServeMux) {
	c := &reportsController{
		registry: cfg.ReportRegistry,
		db:       cfg.DB,
		secret:   cfg.JWTSecret,
	}
	withAuth := auth.WithAuth(cfg.JWTSecret)

	mux.Handle("/api/v1/reports", withAuth(http.HandlerFunc(c.list)))
	mux.Handle("/api/v1/reports/", withAuth(http.HandlerFunc(c.run)))
}

func (c *reportsController) list(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	role := auth.RoleFromCtx(r.Context())
	type summary struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Roles       []string `json:"roles,omitempty"`
	}
	var out []summary
	for _, rep := range c.registry.List() {
		if !roleAllowed(role, rep.Roles()) {
			continue
		}
		out = append(out, summary{Name: rep.Name(), Description: rep.Description(), Roles: rep.Roles()})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (c *reportsController) run(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := strings.TrimPrefix(r.URL.Path, "/api/v1/reports/")
	// strip any trailing /export
	name = strings.TrimSuffix(name, "/export")
	if name == "" {
		c.list(w, r)
		return
	}

	rep, ok := c.registry.Get(name)
	if !ok {
		http.Error(w, "report not found", http.StatusNotFound)
		return
	}

	role := auth.RoleFromCtx(r.Context())
	if !roleAllowed(role, rep.Roles()) {
		http.Error(w, "insufficient permissions", http.StatusForbidden)
		return
	}

	params, err := parseReportParams(r)
	if err != nil {
		http.Error(w, "invalid params: "+err.Error(), http.StatusBadRequest)
		return
	}

	result, err := rep.Run(r.Context(), c.db, params)
	if err != nil {
		slog.Error("report run", "name", name, "err", err)
		http.Error(w, "report execution failed", http.StatusInternalServerError)
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = params.Format
	}
	switch format {
	case "csv":
		writeCSV(w, result)
	default:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

func parseReportParams(r *http.Request) (reports.Params, error) {
	q := r.URL.Query()
	p := reports.Params{
		Format: q.Get("format"),
	}
	if p.Format == "" {
		p.Format = "json"
	}

	now := time.Now().UTC()
	p.From = now.AddDate(0, -1, 0)
	p.To = now

	if s := q.Get("from"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return p, fmt.Errorf("from: %w", err)
		}
		p.From = t
	}
	if s := q.Get("to"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return p, fmt.Errorf("to: %w", err)
		}
		p.To = t
	}
	return p, nil
}

func writeCSV(w http.ResponseWriter, result *reports.Result) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, result.Name))
	cw := csv.NewWriter(w)
	_ = cw.Write(result.Columns)
	for _, row := range result.Rows {
		record := make([]string, len(result.Columns))
		for i, col := range result.Columns {
			v := row[col]
			if v != nil {
				record[i] = fmt.Sprintf("%v", v)
			}
		}
		_ = cw.Write(record)
	}
	cw.Flush()
}

func roleAllowed(role string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, a := range allowed {
		if a == role {
			return true
		}
	}
	return false
}

// ensure domain import is used (for role constants)
var _ = domain.RoleAdmin
