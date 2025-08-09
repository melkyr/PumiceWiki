package auth

import (
	"github.com/casbin/casbin/v2"
	sqlxadapter "github.com/memwey/casbin-sqlx-adapter"
)

// NewEnforcer creates a new Casbin enforcer with policies loaded from the database.
func NewEnforcer(driverName, dsn, modelPath string) (*casbin.Enforcer, error) {
	// Create a new sqlx adapter for Casbin using the database connection details.
	opts := &sqlxadapter.AdapterOptions{
		DriverName:     driverName, // e.g., "sqlite3"
		DataSourceName: dsn,        // e.g., "wiki.db"
		TableName:      "casbin_rule",
	}
	adapter := sqlxadapter.NewAdapterFromOptions(opts)

	// Create a new enforcer with the model and adapter.
	// The model file defines the RBAC structure, and the adapter provides the policy storage.
	enforcer, err := casbin.NewEnforcer(modelPath, adapter)
	if err != nil {
		return nil, err
	}

	// Load all policies from the database.
	// This is essential to ensure the enforcer has the current set of rules.
	if err := enforcer.LoadPolicy(); err != nil {
		return nil, err
	}

	// Here you could add default policies if needed, for example:
	// if hasPolicy := enforcer.HasPolicy("admin", "pages", "write"); !hasPolicy {
	//     enforcer.AddPolicy("admin", "pages", "write")
	// }

	return enforcer, nil
}
