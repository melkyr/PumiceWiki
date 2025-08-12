package auth

import (
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/util"
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

	// Add the custom keyMatch2 function to the enforcer.
	// This is required for the wildcard matching in our model.
	enforcer.AddFunction("keyMatch2", util.KeyMatch2Func)

	// Load all policies from the database.
	// This is essential to ensure the enforcer has the current set of rules.
	if err := enforcer.LoadPolicy(); err != nil {
		return nil, err
	}

	// Add default policies if they don't exist
	if hasPolicy, _ := enforcer.HasPolicy("editor", "/view/*", "GET"); !hasPolicy {
		enforcer.AddPolicy("editor", "/view/*", "GET")
	}
	if hasPolicy, _ := enforcer.HasPolicy("editor", "/edit/*", "GET"); !hasPolicy {
		enforcer.AddPolicy("editor", "/edit/*", "GET")
	}
	if hasPolicy, _ := enforcer.HasPolicy("editor", "/save/*", "POST"); !hasPolicy {
		enforcer.AddPolicy("editor", "/save/*", "POST")
	}
	if hasPolicy, _ := enforcer.HasPolicy("editor", "/list", "GET"); !hasPolicy {
		enforcer.AddPolicy("editor", "/list", "GET")
	}
	if hasPolicy, _ := enforcer.HasPolicy("anonymous", "/view/Home", "GET"); !hasPolicy {
		enforcer.AddPolicy("anonymous", "/view/Home", "GET")
	}

	return enforcer, nil
}
