package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte("super-secret-key-change-me")

type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

type Team struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	OwnerID   int64     `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
}

type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateTeamRequest struct {
	Name string `json:"name"`
}

type AddMemberRequest struct {
	UserID int64 `json:"user_id"`
}

type Claims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

func main() {
	dbURL := envOrDefault("DATABASE_URL", "postgres://app:app@localhost:5432/collab?sslmode=disable")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	log.Println("Connected to Postgres successfully!")

	if err := runMigrations(db); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/users/signup", signupHandler(db))
	r.Post("/users/login", loginHandler(db))

	r.Group(func(pr chi.Router) {
		pr.Use(authMiddleware)
		pr.Post("/teams", createTeamHandler(db))
		pr.Get("/teams", listTeamsHandler(db))
		pr.Post("/teams/{teamID}/members", addMemberHandler(db))
	})

	log.Println("User & Team Service running on :8081")
	log.Fatal(http.ListenAndServe(":8081", r))
}

func envOrDefault(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func runMigrations(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
            id SERIAL PRIMARY KEY,
            email TEXT UNIQUE NOT NULL,
            password TEXT NOT NULL,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        );`,
		`CREATE TABLE IF NOT EXISTS teams (
            id SERIAL PRIMARY KEY,
            name TEXT NOT NULL,
            owner_id INTEGER NOT NULL REFERENCES users(id),
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        );`,
		`CREATE TABLE IF NOT EXISTS team_members (
            team_id INTEGER NOT NULL REFERENCES teams(id),
            user_id INTEGER NOT NULL REFERENCES users(id),
            PRIMARY KEY (team_id, user_id)
        );`,
	}
	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

func signupHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var creds Credentials
		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if creds.Email == "" || creds.Password == "" {
			http.Error(w, "email and password required", http.StatusBadRequest)
			return
		}

		hashed, err := bcrypt.GenerateFromPassword([]byte(creds.Password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "failed to hash password", http.StatusInternalServerError)
			return
		}

		var id int64
		err = db.QueryRow(
			`INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id`,
			creds.Email, string(hashed),
		).Scan(&id)
		if err != nil {
			http.Error(w, "failed to create user (maybe email already used)", http.StatusBadRequest)
			return
		}

		token, err := generateToken(id)
		if err != nil {
			http.Error(w, "failed to generate token", http.StatusInternalServerError)
			return
		}

		resp := map[string]any{
			"user_id": id,
			"token":   token,
		}
		writeJSON(w, http.StatusCreated, resp)
	}
}

func loginHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var creds Credentials
		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		var id int64
		var hashed string
		err := db.QueryRow(
			`SELECT id, password FROM users WHERE email = $1`,
			creds.Email,
		).Scan(&id, &hashed)
		if err != nil {
			http.Error(w, "invalid email or password", http.StatusUnauthorized)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(creds.Password)); err != nil {
			http.Error(w, "invalid email or password", http.StatusUnauthorized)
			return
		}

		token, err := generateToken(id)
		if err != nil {
			http.Error(w, "failed to generate token", http.StatusInternalServerError)
			return
		}

		resp := map[string]any{
			"user_id": id,
			"token":   token,
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func createTeamHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromContext(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var req CreateTeamRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.Name == "" {
			http.Error(w, "team name required", http.StatusBadRequest)
			return
		}

		var teamID int64
		err := db.QueryRow(
			`INSERT INTO teams (name, owner_id) VALUES ($1, $2) RETURNING id`,
			req.Name, userID,
		).Scan(&teamID)
		if err != nil {
			http.Error(w, "failed to create team", http.StatusInternalServerError)
			return
		}

		_, _ = db.Exec(
			`INSERT INTO team_members (team_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			teamID, userID,
		)

		resp := map[string]any{
			"team_id": teamID,
			"name":    req.Name,
		}
		writeJSON(w, http.StatusCreated, resp)
	}
}

func listTeamsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromContext(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		rows, err := db.Query(`
            SELECT t.id, t.name, t.owner_id, t.created_at
            FROM teams t
            JOIN team_members tm ON tm.team_id = t.id
            WHERE tm.user_id = $1
            ORDER BY t.created_at DESC
        `, userID)
		if err != nil {
			http.Error(w, "failed to list teams", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var teams []Team
		for rows.Next() {
			var t Team
			if err := rows.Scan(&t.ID, &t.Name, &t.OwnerID, &t.CreatedAt); err != nil {
				http.Error(w, "failed to scan team", http.StatusInternalServerError)
				return
			}
			teams = append(teams, t)
		}

		writeJSON(w, http.StatusOK, teams)
	}
}

func addMemberHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := userIDFromContext(r)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		teamIDStr := chi.URLParam(r, "teamID")
		teamID, err := strconv.ParseInt(teamIDStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid team id", http.StatusBadRequest)
			return
		}

		var ownerID int64
		err = db.QueryRow(`SELECT owner_id FROM teams WHERE id = $1`, teamID).Scan(&ownerID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "team not found", http.StatusNotFound)
				return
			}
			http.Error(w, "failed to load team", http.StatusInternalServerError)
			return
		}
		if ownerID != userID {
			http.Error(w, "only owner can add members", http.StatusForbidden)
			return
		}

		var req AddMemberRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if req.UserID == 0 {
			http.Error(w, "user_id required", http.StatusBadRequest)
			return
		}

		_, err = db.Exec(
			`INSERT INTO team_members (team_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			teamID, req.UserID,
		)
		if err != nil {
			http.Error(w, "failed to add member", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func generateToken(userID int64) (string, error) {
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

type contextKey string

const userIDKey contextKey = "userID"

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, "missing or invalid Authorization header", http.StatusUnauthorized)
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		ctx = contextWithUserID(ctx, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func contextWithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func userIDFromContext(r *http.Request) (int64, bool) {
	val := r.Context().Value(userIDKey)
	if val == nil {
		return 0, false
	}
	id, ok := val.(int64)
	return id, ok
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
