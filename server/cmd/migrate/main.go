package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"

	"github.com/yourusername/incident-ticketing/internal/db"
)

func main() {
	_ = godotenv.Load()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DB_URL")
	}
	if dsn == "" {
		fmt.Println("DB_URL or DATABASE_URL is missing; set one in .env or environment")
		os.Exit(1)
	}

	database, err := db.Connect(dsn)
	if err != nil {
		fmt.Printf("failed to connect database: %v\n", err)
		os.Exit(1)
	}

	migrations := []string{
		"internal/db/migrations/001_init.sql",
		"internal/db/migrations/002_migrate.sql",
		"internal/db/migrations/003_migrate.sql",
		"internal/db/migrations/004_ai_problem_management.sql",
		"internal/db/migrations/005_skill_assignment.sql",
	}

	for _, migrationPath := range migrations {
		content, err := os.ReadFile(filepath.Clean(migrationPath))
		if err != nil {
			fmt.Printf("failed to read migration %s: %v\n", migrationPath, err)
			os.Exit(1)
		}

		if err := database.Exec(string(content)).Error; err != nil {
			fmt.Printf("failed to apply migration %s: %v\n", migrationPath, err)
			os.Exit(1)
		}

		fmt.Printf("applied %s\n", migrationPath)
	}

	fmt.Println("all migrations applied successfully")
}
