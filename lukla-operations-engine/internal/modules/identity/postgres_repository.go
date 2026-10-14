package identity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Repository defines data access operations for identity module.
type Repository interface {
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
}

// PostgresRepository is the PostgreSQL implementation of Repository.
// It receives *sql.DB which is backed by the pgx/v5/stdlib driver.
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository constructs a PostgresRepository.
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// FindByEmail retrieves an active user by their email address with role information.
func (r *PostgresRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT
			u.id::text,
			u.employee_code,
			u.full_name,
			u.email,
			u.phone_number,
			u.password_hash,
			u.role_id,
			r.code AS role_code,
			r.name AS role_name,
			u.client_id::text,
			u.can_access_app,
			u.is_active,
			u.created_at,
			u.updated_at
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE LOWER(TRIM(u.email)) = LOWER(TRIM($1)) AND u.deleted_at IS NULL
		LIMIT 1;
	`

	var u User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&u.ID,
		&u.EmployeeCode,
		&u.FullName,
		&u.Email,
		&u.PhoneNumber,
		&u.PasswordHash,
		&u.RoleID,
		&u.RoleCode,
		&u.RoleName,
		&u.ClientID,
		&u.CanAccessApp,
		&u.IsActive,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	return &u, nil
}

// FindByID retrieves a user by their UUID.
func (r *PostgresRepository) FindByID(ctx context.Context, id string) (*User, error) {
	query := `
		SELECT
			u.id::text,
			u.employee_code,
			u.full_name,
			u.email,
			u.phone_number,
			u.password_hash,
			u.role_id,
			r.code AS role_code,
			r.name AS role_name,
			u.client_id::text,
			u.can_access_app,
			u.is_active,
			u.created_at,
			u.updated_at
		FROM users u
		JOIN roles r ON r.id = u.role_id
		WHERE u.id::text = $1 AND u.deleted_at IS NULL
		LIMIT 1;
	`

	var u User
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID,
		&u.EmployeeCode,
		&u.FullName,
		&u.Email,
		&u.PhoneNumber,
		&u.PasswordHash,
		&u.RoleID,
		&u.RoleCode,
		&u.RoleName,
		&u.ClientID,
		&u.CanAccessApp,
		&u.IsActive,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	return &u, nil
}
