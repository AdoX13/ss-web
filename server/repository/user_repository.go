package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"mqtt-streaming-server/domain"
)

type userRepository struct {
	db *mongo.Database
}

func NewUserRepository(db *mongo.Database) domain.UserRepository {
	return &userRepository{db: db}
}

func (repo *userRepository) col() *mongo.Collection {
	return repo.db.Collection("users")
}

func (repo *userRepository) Save(ctx context.Context, email, password string) error {
	_, err := repo.col().InsertOne(ctx, domain.User{
		Email:    email,
		Password: password,
		Role:     domain.RoleDoctor, // default role for self-registration
		Active:   true,
	})
	return err
}

func (repo *userRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := repo.col().FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (repo *userRepository) GetAll(ctx context.Context) ([]*domain.User, error) {
	cursor, err := repo.col().Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var users []*domain.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}

func (repo *userRepository) UpdateRole(ctx context.Context, email, role string) error {
	_, err := repo.col().UpdateOne(ctx,
		bson.M{"email": email},
		bson.M{"$set": bson.M{"role": role}},
	)
	return err
}

func (repo *userRepository) UpdatePassword(ctx context.Context, email, hash string) error {
	_, err := repo.col().UpdateOne(ctx,
		bson.M{"email": email},
		bson.M{"$set": bson.M{"password": hash}},
	)
	return err
}

func (repo *userRepository) Deactivate(ctx context.Context, email string) error {
	_, err := repo.col().UpdateOne(ctx,
		bson.M{"email": email},
		bson.M{"$set": bson.M{"active": false}},
	)
	return err
}
