package domain

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RefreshToken struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Token     string             `bson:"token"`
	Email     string             `bson:"email"`
	CreatedAt time.Time          `bson:"created_at"`
	ExpiresAt time.Time          `bson:"expires_at"`
	Revoked   bool               `bson:"revoked"`
}

type RefreshTokenRepository interface {
	Save(ctx context.Context, t *RefreshToken) error
	FindByToken(ctx context.Context, token string) (*RefreshToken, error)
	Revoke(ctx context.Context, token string) error
	RevokeAllForEmail(ctx context.Context, email string) error
}
