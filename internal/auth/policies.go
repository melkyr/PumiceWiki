package auth

import (
	"fmt"
	"go-wiki-app/internal/logger"

	"github.com/casbin/casbin/v2"
)

// SeedDefaultPolicies ensures that the application has a baseline set of authorization rules.
// It checks if each default policy exists before adding it, making the operation idempotent
// and safe to run on every application start.
func SeedDefaultPolicies(e casbin.IEnforcer, log logger.Logger) {
	log.Info("Seeding default authorization policies...")

	// Default policies grant basic access to anonymous users and content management
	// permissions to editors. Note that the 'editor' role inherits from 'anonymous'.
	policies := [][]string{
		// Anonymous users can view pages and access login/callback routes.
		{"anonymous", "/view/*", "GET"},
		{"anonymous", "/auth/login", "GET"},
		{"anonymous", "/auth/callback", "GET"},
		{"anonymous", "/categories", "GET"},
		{"anonymous", "/category/*", "GET"},
		{"anonymous", "/api/search/categories", "GET"},

		// Editors can do everything anonymous users can, plus edit, save, and list pages.
		{"editor", "/edit/*", "GET"},
		{"editor", "/save/*", "POST"},
		{"editor", "/list", "GET"},
	}
	for _, p := range policies {
		if has, _ := e.HasPolicy(p); !has {
			if _, err := e.AddPolicy(p); err != nil {
				log.Error(err, fmt.Sprintf("Failed to add policy %v", p))
			}
		}
	}

	// Granting the 'editor' role all permissions of the 'anonymous' role.
	if has, _ := e.HasRoleForUser("editor", "anonymous"); !has {
		if _, err := e.AddRoleForUser("editor", "anonymous"); err != nil {
			log.Error(err, "Failed to add role 'editor' -> 'anonymous'")
		}
	}
	log.Info("Policy seeding complete.")
}
