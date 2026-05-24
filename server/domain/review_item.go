package domain

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ReviewItemStatus string

const (
	ReviewItemPending   ReviewItemStatus = "pending"
	ReviewItemApproved  ReviewItemStatus = "approved"
	ReviewItemCorrected ReviewItemStatus = "corrected"
	ReviewItemRejected  ReviewItemStatus = "rejected"
)

type ReviewItem struct {
	ID                 primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	ImageID            primitive.ObjectID  `json:"image_id" bson:"image_id"`
	FieldName          string              `json:"field_name" bson:"field_name"`
	OriginalValue      *string             `json:"original_value" bson:"original_value"`
	OriginalConfidence float64             `json:"original_confidence" bson:"original_confidence"`
	Status             ReviewItemStatus    `json:"status" bson:"status"`
	ReviewerEmail      string              `json:"reviewer_email,omitempty" bson:"reviewer_email,omitempty"`
	CorrectedValue     *string             `json:"corrected_value,omitempty" bson:"corrected_value,omitempty"`
	ReviewedAt         *time.Time          `json:"reviewed_at,omitempty" bson:"reviewed_at,omitempty"`
	CreatedAt          time.Time           `json:"created_at" bson:"created_at"`
}

type ReviewItemFilters struct {
	Status    ReviewItemStatus
	FieldName string
	ImageID   string
	Limit     int
	Offset    int
}

type ReviewItemUpdate struct {
	Status         ReviewItemStatus
	ReviewerEmail  string
	CorrectedValue *string
	ReviewedAt     time.Time
}

type ReviewItemRepository interface {
	Save(ctx context.Context, item *ReviewItem) error
	List(ctx context.Context, f ReviewItemFilters) ([]*ReviewItem, error)
	GetByID(ctx context.Context, id string) (*ReviewItem, error)
	UpdateStatus(ctx context.Context, id string, u ReviewItemUpdate) error
}
