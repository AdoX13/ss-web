package routes

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"

	"mqtt-streaming-server/auth"
	"mqtt-streaming-server/domain"
	"mqtt-streaming-server/repository"
)

type UserController struct {
	UserRepository domain.UserRepository
	jwtSecret      string
}

func InitUserRoutes(db *mongo.Database, mux *http.ServeMux) {
	// The JWT secret is read here for the legacy /login handler. The
	// canonical auth endpoints live at /api/v1/auth/*.
	secret := jwtSecretFromEnv()
	ctrl := &UserController{
		UserRepository: repository.NewUserRepository(db),
		jwtSecret:      secret,
	}

	mux.HandleFunc("/register", ctrl.Register)
	mux.HandleFunc("/login", ctrl.Login)
	mux.Handle("/profile", auth.WithAuth(secret)(http.HandlerFunc(ctrl.GetProfile)))
}

func (ctlr *UserController) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req domain.User
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	existing, err := ctlr.UserRepository.FindByEmail(r.Context(), req.Email)
	if err != nil && err != mongo.ErrNoDocuments {
		http.Error(w, "Failed to check existing user", http.StatusInternalServerError)
		return
	}
	if existing != nil {
		http.Error(w, "User already exists", http.StatusConflict)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}
	if err := ctlr.UserRepository.Save(r.Context(), req.Email, hash); err != nil {
		http.Error(w, "Failed to save user", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (ctlr *UserController) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req domain.User
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	user, err := ctlr.UserRepository.FindByEmail(r.Context(), req.Email)
	if err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	ok, err := auth.VerifyPassword(req.Password, user.Password)
	if err != nil || !ok {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	// Migrate bcrypt → Argon2id transparently.
	if strings.HasPrefix(user.Password, "$2") {
		if newHash, err := auth.HashPassword(req.Password); err == nil {
			_ = ctlr.UserRepository.UpdatePassword(r.Context(), user.Email, newHash)
		}
	}

	token, err := auth.GenerateAccessToken(user.Email, user.Role, ctlr.jwtSecret)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
		"email": user.Email,
		"role":  user.Role,
	})
}

func (ctlr *UserController) GetProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	email := auth.EmailFromCtx(r.Context())
	user, err := ctlr.UserRepository.FindByEmail(r.Context(), email)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	user.Password = ""
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func jwtSecretFromEnv() string {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return s
	}
	return "dev-secret-change-in-production"
}
