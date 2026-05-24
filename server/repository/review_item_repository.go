package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"mqtt-streaming-server/domain"
)

type reviewItemRepository struct {
	db *mongo.Database
}

func NewReviewItemRepository(db *mongo.Database) domain.ReviewItemRepository {
	return &reviewItemRepository{db: db}
}

func (r *reviewItemRepository) col() *mongo.Collection {
	return r.db.Collection("review_items")
}

func (r *reviewItemRepository) Save(ctx context.Context, item *domain.ReviewItem) error {
	_, err := r.col().InsertOne(ctx, item)
	return err
}

func (r *reviewItemRepository) List(ctx context.Context, f domain.ReviewItemFilters) ([]*domain.ReviewItem, error) {
	filter := bson.M{}
	if f.Status != "" {
		filter["status"] = f.Status
	}
	if f.FieldName != "" {
		filter["field_name"] = f.FieldName
	}
	if f.ImageID != "" {
		if id, err := primitive.ObjectIDFromHex(f.ImageID); err == nil {
			filter["image_id"] = id
		}
	}

	limit := int64(100)
	if f.Limit > 0 && f.Limit <= 500 {
		limit = int64(f.Limit)
	}
	skip := int64(f.Offset)

	opts := options.Find().
		SetSort(bson.D{{Key: "original_confidence", Value: 1}}). // lowest confidence first
		SetLimit(limit).
		SetSkip(skip)

	cursor, err := r.col().Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []*domain.ReviewItem
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *reviewItemRepository) GetByID(ctx context.Context, id string) (*domain.ReviewItem, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	var item domain.ReviewItem
	err = r.col().FindOne(ctx, bson.M{"_id": objID}).Decode(&item)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *reviewItemRepository) UpdateStatus(ctx context.Context, id string, u domain.ReviewItemUpdate) error {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	update := bson.M{
		"status":         u.Status,
		"reviewer_email": u.ReviewerEmail,
		"reviewed_at":    u.ReviewedAt,
	}
	if u.CorrectedValue != nil {
		update["corrected_value"] = *u.CorrectedValue
	}
	_, err = r.col().UpdateOne(ctx,
		bson.M{"_id": objID},
		bson.M{"$set": update},
	)
	return err
}

// ensure interface is satisfied at compile time
var _ domain.ReviewItemRepository = (*reviewItemRepository)(nil)

// buildReviewedAt is a helper used by route handlers.
func nowPtr() *time.Time { t := time.Now().UTC(); return &t }
