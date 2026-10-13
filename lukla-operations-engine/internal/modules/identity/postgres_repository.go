package identity

import "database/sql"

// PostgresRepository is the PostgreSQL implementation of Repository.
// It receives *sql.DB which is backed by the pgx/v5/stdlib driver.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository constructs a PostgresRepository.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// TODO: implement Repository interface methods using db.QueryContext / db.ExecContext.
// Always use parameterised queries — never interpolate user input into SQL strings.
// Wrap multi-step operations with database.WithTransaction from platform/database.
