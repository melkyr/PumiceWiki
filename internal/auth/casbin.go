package auth

import (
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/util"
	sqlxadapter "github.com/memwey/casbin-sqlx-adapter"
)

// NewEnforcer creates and configures a new Casbin enforcer.
// It sets up the database adapter, loads the model from the specified path,
// and loads all authorization policies from the database.
//
// Parameters:
//   - driverName: The name of the database driver (e.g., "mysql").
//   - dsn: The Data Source Name for the database connection.
//   - modelPath: The file path to the Casbin model configuration (`.conf`).
//
// Returns a fully configured Casbin enforcer or an error if setup fails.
func NewEnforcer(driverName, dsn, modelPath string) (*casbin.Enforcer, error) {
	// Initialize the database adapter for Casbin. This allows Casbin to store
	// its policies in our application's database.
	opts := &sqlxadapter.AdapterOptions{
		DriverName:     driverName,
		DataSourceName: dsn,
		TableName:      "casbin_rule",
	}
	adapter := sqlxadapter.NewAdapterFromOptions(opts)

	// Create a new enforcer with the model file and the database adapter.
	enforcer, err := casbin.NewEnforcer(modelPath, adapter)
	if err != nil {
		return nil, err
	}

	// Add the keyMatch2 function to the enforcer's function map.
	// This is a built-in Casbin function that allows for wildcard matching in paths
	// (e.g., matching "/view/*" to "/view/SomePage"). It's required by our model.
	enforcer.AddFunction("keyMatch2", util.KeyMatch2Func)

	// Load all authorization policies from the database. This is a crucial step
	// to ensure the enforcer has the current set of rules to work with.
	if err := enforcer.LoadPolicy(); err != nil {
		return nil, err
	}

	// Note: Seeding default policies is handled in main.go's seedDefaultPolicies function
	// to keep this constructor focused on creating the enforcer instance.

	return enforcer, nil
}
