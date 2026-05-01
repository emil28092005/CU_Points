// seed inserts development test users into the database.
// Run via: make seed  (only use against local dev DB, never production)
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type seedUser struct {
	email     string
	name      string
	password  string
	role      string
	studentID string // empty string → NULL in DB
}

var devUsers = []seedUser{
	{
		email:     "student@cu.ru",
		name:      "Иван Студентов",
		password:  "password123",
		role:      "student",
		studentID: "STU001",
	},
	{
		email:    "partner@cu.ru",
		name:     "Кофейня Уют",
		password: "password123",
		role:     "partner",
	},
	{
		email:    "admin@cu.ru",
		name:     "Администратор ЦУ",
		password: "password123",
		role:     "admin",
	},
}

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		slog.Error("DATABASE_URL is not set")
		os.Exit(1)
	}

	ctx := context.Background()
	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		slog.Error("connect failed", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	for _, u := range devUsers {
		hash, err := bcrypt.GenerateFromPassword([]byte(u.password), bcrypt.DefaultCost)
		if err != nil {
			slog.Error("bcrypt failed", "email", u.email, "err", err)
			os.Exit(1)
		}

		// NULLIF converts empty string to NULL for student_id
		_, err = db.Exec(ctx, `
			INSERT INTO users (email, name, password_hash, role, student_id)
			VALUES ($1, $2, $3, $4, NULLIF($5, ''))
			ON CONFLICT (email) DO UPDATE
				SET password_hash = EXCLUDED.password_hash,
				    name          = EXCLUDED.name
		`, u.email, u.name, string(hash), u.role, u.studentID)
		if err != nil {
			slog.Error("seed insert failed", "email", u.email, "err", err)
			os.Exit(1)
		}

		fmt.Printf("seeded: %-30s  role=%-8s  password=%s\n", u.email, u.role, u.password)
	}

	fmt.Println("\nDone. Test credentials above are for local development only.")
}
