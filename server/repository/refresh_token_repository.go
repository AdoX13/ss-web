package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"mqtt-streaming-server/domain"
)

type refreshTokenRepository struct {
	db *mongo.Database
}

func NewRefreshTokenRepository(db *mongo.Database) domain.RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

func (r *refreshTokenRepository) col() *mongo.Collection {
	return r.db.Collection("refresh_tokens")
}

func (r *refreshTokenRepository) Save(ctx context.Context, t *domain.RefreshToken) error {
	_, err := r.col().InsertOne(ctx, t)
	return err
}

func (r *refreshTokenRepository) FindByToken(ctx context.Context, token string) (*domain.RefreshToken, error) {
	var rt domain.RefreshToken
	err := r.col().FindOne(ctx, bson.M{"token": token, "revoked": false}).Decode(&rt)
	if err != nil {
		return nil, err
	}
	return &rt, nil
}

func (r *refreshTokenRepository) Revoke(ctx context.Context, token string) error {
	_, err := r.col().UpdateOne(ctx,
		bson.M{"token": token},
		bson.M{"$set": bson.M{"revoked": true}},
	)
	return err
}

func (r *refreshTokenRepository) RevokeAllForEmail(ctx context.Context, email string) error {
	_, err := r.col().UpdateMany(ctx,
		bson.M{"email": email, "revoked": false},
		bson.M{"$set": bson.M{"revoked": true}},
	)
	return err
}
