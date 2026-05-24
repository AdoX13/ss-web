package domain

import "context"

const (
	RoleAdmin      = "admin"
	RoleDoctor     = "doctor"
	RoleResearcher = "researcher"
	RoleAuditor    = "auditor"
)

type User struct {
	Email    string `json:"email" bson:"email"`
	Password string `json:"password,omitempty" bson:"password"`
	Role     string `json:"role,omitempty" bson:"role"`
	Active   bool   `json:"active" bson:"active"`
}

type UserRepository interface {
	Save(ctx context.Context, email, password string) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	GetAll(ctx context.Context) ([]*User, error)
	UpdateRole(ctx context.Context, email, role string) error
	UpdatePassword(ctx context.Context, email, hash string) error
	Deactivate(ctx context.Context, email string) error
}
