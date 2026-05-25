package reports

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	roleAdmin      = "admin"
	roleDoctor     = "doctor"
	roleResearcher = "researcher"
	roleAuditor    = "auditor"
)

type photoDoc struct {
	ID                primitive.ObjectID `bson:"_id,omitempty"`
	Timestamp         time.Time          `bson:"timestamp"`
	DeviceID          string             `bson:"device_id"`
	Nume              string             `bson:"nume"`
	Prenume           string             `bson:"prenume"`
	ProfesieFunctie   string             `bson:"profesie_functie"`
	LocDeMunca        string             `bson:"loc_de_munca"`
	TipControl        string             `bson:"tip_control"`
	AvizMedical       string             `bson:"aviz_medical"`
	AvizApt           bool               `bson:"aviz_apt"`
	AvizAptConditionat bool              `bson:"aviz_apt_conditionat"`
	Data              time.Time          `bson:"data"`
	DataUrmExaminari  time.Time          `bson:"data_urm_examinari"`
	OverallConfidence float64            `bson:"overall_confidence"`
	NeedsReview       bool               `bson:"needs_review"`
}

type reviewItemDoc struct {
	ID                 primitive.ObjectID `bson:"_id,omitempty"`
	ImageID            primitive.ObjectID `bson:"image_id"`
	FieldName          string             `bson:"field_name"`
	OriginalConfidence float64            `bson:"original_confidence"`
	Status             string             `bson:"status"`
	CreatedAt          time.Time          `bson:"created_at"`
	ReviewedAt         *time.Time         `bson:"reviewed_at,omitempty"`
}

type ocrResultDoc struct {
	DocumentID        string    `bson:"document_id"`
	ExtractedAt       time.Time `bson:"extracted_at"`
	OverallConfidence float64   `bson:"overall_confidence"`
	NeedsReview       bool      `bson:"needs_review"`
	ProcessingMS       int64     `bson:"processing_ms"`
}

type recentExamsReport struct{}

func (recentExamsReport) Name() string        { return "recent_exams" }
func (recentExamsReport) Description() string { return "Recent medical exams in the selected range" }
func (recentExamsReport) Roles() []string     { return []string{roleAdmin, roleDoctor} }
func (recentExamsReport) Run(ctx context.Context, db *mongo.Database, p Params) (*Result, error) {
	filter := bson.M{"data": rangeFilter(p.From, p.To)}
	photos, err := findPhotos(ctx, db, filter, "data", -1, 200)
	if err != nil {
		return nil, err
	}
	rows := make([]Row, 0, len(photos))
	for _, ph := range photos {
		rows = append(rows, Row{
			"document_id":     ph.ID.Hex(),
			"patient":         patientDisplay(ph),
			"profession":      ph.ProfesieFunctie,
			"control_type":    cleanControlType(ph.TipControl),
			"medical_opinion": cleanOpinion(ph.AvizMedical),
			"exam_date":       formatDate(ph.Data),
			"expires_at":      formatDate(ph.DataUrmExaminari),
			"confidence":      round(ph.OverallConfidence, 3),
		})
	}
	return &Result{
		Name:    "recent_exams",
		Columns: []string{"document_id", "patient", "profession", "control_type", "medical_opinion", "exam_date", "expires_at", "confidence"},
		Rows:    rows,
	}, nil
}

type upcomingExpirationsReport struct{}

func (upcomingExpirationsReport) Name() string { return "upcoming_expirations" }
func (upcomingExpirationsReport) Description() string {
	return "Medical clearances expiring in the selected range"
}
func (upcomingExpirationsReport) Roles() []string { return []string{roleAdmin, roleDoctor} }
func (upcomingExpirationsReport) Run(ctx context.Context, db *mongo.Database, p Params) (*Result, error) {
	from, to := p.From, p.To
	now := time.Now().UTC()
	if !from.After(now) && !to.After(now) {
		from = now
		to = now.AddDate(0, 0, 30)
	}
	filter := bson.M{"data_urm_examinari": rangeFilter(from, to)}
	photos, err := findPhotos(ctx, db, filter, "data_urm_examinari", 1, 200)
	if err != nil {
		return nil, err
	}
	rows := make([]Row, 0, len(photos))
	for _, ph := range photos {
		rows = append(rows, Row{
			"document_id":     ph.ID.Hex(),
			"patient":         patientDisplay(ph),
			"profession":      ph.ProfesieFunctie,
			"medical_opinion": cleanOpinion(ph.AvizMedical),
			"expires_at":      formatDate(ph.DataUrmExaminari),
			"days_until":      int(math.Ceil(ph.DataUrmExaminari.Sub(now).Hours() / 24)),
		})
	}
	return &Result{
		Name:    "upcoming_expirations",
		Columns: []string{"document_id", "patient", "profession", "medical_opinion", "expires_at", "days_until"},
		Rows:    rows,
	}, nil
}

type complianceReport struct{}

func (complianceReport) Name() string { return "compliance_percentage" }
func (complianceReport) Description() string {
	return "Percentage of workers with a valid medical clearance"
}
func (complianceReport) Roles() []string { return []string{roleAdmin, roleDoctor, roleAuditor} }
func (complianceReport) Run(ctx context.Context, db *mongo.Database, p Params) (*Result, error) {
	filter := bson.M{"timestamp": bson.M{"$lte": p.To}}
	photos, err := findPhotos(ctx, db, filter, "timestamp", -1, 0)
	if err != nil {
		return nil, err
	}
	total := len(photos)
	valid := 0
	for _, ph := range photos {
		if (ph.AvizApt || ph.AvizAptConditionat || cleanOpinion(ph.AvizMedical) == "APT" || cleanOpinion(ph.AvizMedical) == "APT Conditionat") &&
			!ph.DataUrmExaminari.IsZero() &&
			!ph.DataUrmExaminari.Before(p.To) {
			valid++
		}
	}
	invalid := total - valid
	rows := []Row{
		{"status": "valid", "count": valid, "percentage": percent(valid, total)},
		{"status": "expired_or_invalid", "count": invalid, "percentage": percent(invalid, total)},
	}
	return &Result{Name: "compliance_percentage", Columns: []string{"status", "count", "percentage"}, Rows: rows}, nil
}

type anonymizedExportReport struct{}

func (anonymizedExportReport) Name() string { return "anonymized_export" }
func (anonymizedExportReport) Description() string {
	return "Anonymized research export with k-anonymity threshold 5"
}
func (anonymizedExportReport) Roles() []string { return []string{roleResearcher, roleAdmin} }
func (anonymizedExportReport) Run(ctx context.Context, db *mongo.Database, p Params) (*Result, error) {
	photos, err := findPhotos(ctx, db, bson.M{"data": rangeFilter(p.From, p.To)}, "data", -1, 0)
	if err != nil {
		return nil, err
	}
	rows := anonymizePhotos(photos, 5)
	return &Result{
		Name:    "anonymized_export",
		Columns: []string{"profession", "exam_month", "documents", "control_types", "medical_opinions"},
		Rows:    rows,
	}, nil
}

type ocrPerformanceReport struct{}

func (ocrPerformanceReport) Name() string { return "ocr_performance" }
func (ocrPerformanceReport) Description() string {
	return "OCR confidence and review routing metrics"
}
func (ocrPerformanceReport) Roles() []string { return []string{roleAdmin, roleAuditor} }
func (ocrPerformanceReport) Run(ctx context.Context, db *mongo.Database, p Params) (*Result, error) {
	results, err := findOCRResults(ctx, db, bson.M{"extracted_at": rangeFilter(p.From, p.To)})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return ocrPerformanceFromPhotos(ctx, db, p)
	}

	total := len(results)
	needsReview := 0
	highConfidence := 0
	sum := 0.0
	processingSum := int64(0)
	processingCount := 0
	for _, item := range results {
		sum += item.OverallConfidence
		if item.NeedsReview {
			needsReview++
		}
		if item.OverallConfidence >= 0.95 {
			highConfidence++
		}
		if item.ProcessingMS > 0 {
			processingSum += item.ProcessingMS
			processingCount++
		}
	}
	avg := 0.0
	if total > 0 {
		avg = sum / float64(total)
	}
	avgProcessing := 0.0
	if processingCount > 0 {
		avgProcessing = float64(processingSum) / float64(processingCount)
	}
	rows := []Row{
		{"metric": "documents", "value": total},
		{"metric": "avg_confidence", "value": round(avg, 3)},
		{"metric": "needs_review", "value": needsReview},
		{"metric": "high_confidence", "value": highConfidence},
		{"metric": "avg_processing_ms", "value": round(avgProcessing, 2)},
	}
	return &Result{Name: "ocr_performance", Columns: []string{"metric", "value"}, Rows: rows}, nil
}

func ocrPerformanceFromPhotos(ctx context.Context, db *mongo.Database, p Params) (*Result, error) {
	photos, err := findPhotos(ctx, db, bson.M{"timestamp": rangeFilter(p.From, p.To)}, "timestamp", -1, 0)
	if err != nil {
		return nil, err
	}
	total := len(photos)
	needsReview := 0
	highConfidence := 0
	sum := 0.0
	for _, ph := range photos {
		sum += ph.OverallConfidence
		if ph.NeedsReview {
			needsReview++
		}
		if ph.OverallConfidence >= 0.95 {
			highConfidence++
		}
	}
	avg := 0.0
	if total > 0 {
		avg = sum / float64(total)
	}
	rows := []Row{
		{"metric": "documents", "value": total},
		{"metric": "avg_confidence", "value": round(avg, 3)},
		{"metric": "needs_review", "value": needsReview},
		{"metric": "high_confidence", "value": highConfidence},
		{"metric": "avg_processing_ms", "value": 0},
	}
	return &Result{Name: "ocr_performance", Columns: []string{"metric", "value"}, Rows: rows}, nil
}

type reviewQueueStatsReport struct{}

func (reviewQueueStatsReport) Name() string { return "review_queue_stats" }
func (reviewQueueStatsReport) Description() string {
	return "Review queue volume and resolution metrics"
}
func (reviewQueueStatsReport) Roles() []string { return []string{roleAdmin, roleDoctor, roleAuditor} }
func (reviewQueueStatsReport) Run(ctx context.Context, db *mongo.Database, p Params) (*Result, error) {
	items, err := findReviewItems(ctx, db, bson.M{"created_at": rangeFilter(p.From, p.To)})
	if err != nil {
		return nil, err
	}
	counts := map[string]int{"pending": 0, "approved": 0, "corrected": 0, "rejected": 0}
	var resolved int
	var resolutionHours float64
	for _, item := range items {
		counts[item.Status]++
		if item.ReviewedAt != nil && !item.CreatedAt.IsZero() {
			resolved++
			resolutionHours += item.ReviewedAt.Sub(item.CreatedAt).Hours()
		}
	}
	avgResolution := 0.0
	if resolved > 0 {
		avgResolution = resolutionHours / float64(resolved)
	}
	rows := []Row{
		{"metric": "pending", "value": counts["pending"]},
		{"metric": "approved", "value": counts["approved"]},
		{"metric": "corrected", "value": counts["corrected"]},
		{"metric": "rejected", "value": counts["rejected"]},
		{"metric": "avg_resolution_hours", "value": round(avgResolution, 2)},
	}
	return &Result{Name: "review_queue_stats", Columns: []string{"metric", "value"}, Rows: rows}, nil
}

func findPhotos(ctx context.Context, db *mongo.Database, filter bson.M, sortField string, sortDir int, limit int64) ([]photoDoc, error) {
	findOpts := options.Find()
	if sortField != "" {
		findOpts.SetSort(bson.D{{Key: sortField, Value: sortDir}})
	}
	if limit > 0 {
		findOpts.SetLimit(limit)
	}
	cursor, err := db.Collection("photos").Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var photos []photoDoc
	if err := cursor.All(ctx, &photos); err != nil {
		return nil, err
	}
	return photos, nil
}

func findReviewItems(ctx context.Context, db *mongo.Database, filter bson.M) ([]reviewItemDoc, error) {
	cursor, err := db.Collection("review_items").Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var items []reviewItemDoc
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func findOCRResults(ctx context.Context, db *mongo.Database, filter bson.M) ([]ocrResultDoc, error) {
	cursor, err := db.Collection("ocr_results").Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var results []ocrResultDoc
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

func rangeFilter(from, to time.Time) bson.M {
	return bson.M{"$gte": from, "$lte": to}
}

func patientDisplay(ph photoDoc) string {
	name := strings.TrimSpace(strings.TrimSpace(ph.Nume) + " " + strings.TrimSpace(ph.Prenume))
	if name == "" {
		return "unknown"
	}
	return name
}

func formatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02")
}

func formatMonth(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	return t.UTC().Format("2006-01")
}

func cleanControlType(s string) string {
	s = strings.TrimSpace(strings.TrimPrefix(s, "Control "))
	if s == "Control medical periodic" {
		return "Periodic"
	}
	if s == "" {
		return "unknown"
	}
	return s
}

func cleanOpinion(s string) string {
	s = strings.TrimSpace(s)
	switch strings.ToUpper(s) {
	case "APT CONDITIONAT", "APT CONDIȚIONAT":
		return "APT Conditionat"
	case "INAPT TEMPORAR":
		return "Inapt Temporar"
	case "INAPT":
		return "Inapt"
	case "APT":
		return "APT"
	default:
		if s == "" {
			return "unknown"
		}
		return s
	}
}

func percent(part, total int) float64 {
	if total == 0 {
		return 0
	}
	return round(float64(part)*100/float64(total), 2)
}

func round(v float64, places int) float64 {
	pow := math.Pow(10, float64(places))
	return math.Round(v*pow) / pow
}

func anonymizePhotos(photos []photoDoc, k int) []Row {
	buckets := map[string]*bucket{}
	suppressed := &bucket{
		profession: "suppressed",
		month:      "suppressed",
		controls:   map[string]bool{},
		opinions:   map[string]bool{},
	}
	for _, ph := range photos {
		profession := strings.TrimSpace(ph.ProfesieFunctie)
		if profession == "" {
			profession = "unknown"
		}
		month := formatMonth(ph.Data)
		key := profession + "|" + month
		b, ok := buckets[key]
		if !ok {
			b = &bucket{profession: profession, month: month, controls: map[string]bool{}, opinions: map[string]bool{}}
			buckets[key] = b
		}
		b.count++
		b.controls[cleanControlType(ph.TipControl)] = true
		b.opinions[cleanOpinion(ph.AvizMedical)] = true
	}

	keys := make([]string, 0, len(buckets))
	for key := range buckets {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	rows := make([]Row, 0, len(buckets))
	for _, key := range keys {
		b := buckets[key]
		if b.count < k {
			suppressed.count += b.count
			for c := range b.controls {
				suppressed.controls[c] = true
			}
			for o := range b.opinions {
				suppressed.opinions[o] = true
			}
			continue
		}
		rows = append(rows, bucketRow(b))
	}
	if suppressed.count > 0 {
		rows = append(rows, bucketRow(suppressed))
	}
	return rows
}

func bucketRow(b *bucket) Row {
	return Row{
		"profession":       b.profession,
		"exam_month":       b.month,
		"documents":        b.count,
		"control_types":    joinSet(b.controls),
		"medical_opinions": joinSet(b.opinions),
	}
}

type bucket struct {
	profession string
	month      string
	count      int
	controls   map[string]bool
	opinions   map[string]bool
}

func joinSet(values map[string]bool) string {
	out := make([]string, 0, len(values))
	for v := range values {
		if v != "" && v != "unknown" {
			out = append(out, v)
		}
	}
	sort.Strings(out)
	if len(out) == 0 {
		return "unknown"
	}
	return strings.Join(out, ", ")
}
