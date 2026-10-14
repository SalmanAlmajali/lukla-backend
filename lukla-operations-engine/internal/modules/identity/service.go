package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("Email atau password salah")
	ErrAccountInactive    = errors.New("Akun Anda telah dinonaktifkan")
	ErrNoAppAccess        = errors.New("Akun Anda tidak memiliki hak akses aplikasi")
	ErrUserNotFound       = errors.New("Pengguna tidak ditemukan")
)

// Service provides identity and authentication business logic.
type Service struct {
	repo        Repository
	tokenSecret string
	tokenTTL    time.Duration
}

// NewService constructs an identity Service.
func NewService(repo Repository, tokenSecret string, tokenTTL time.Duration) *Service {
	if tokenTTL <= 0 {
		tokenTTL = 24 * time.Hour
	}
	return &Service{
		repo:        repo,
		tokenSecret: tokenSecret,
		tokenTTL:    tokenTTL,
	}
}

// Login verifies email and password, checks account permissions, and returns a signed JWT token.
func (s *Service) Login(ctx context.Context, req LoginRequest) (*LoginData, error) {
	user, err := s.repo.FindByEmail(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("login lookup: %w", err)
	}

	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, ErrAccountInactive
	}

	if !user.CanAccessApp {
		return nil, ErrNoAppAccess
	}

	if user.PasswordHash == nil || *user.PasswordHash == "" {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"sub": user.ID,
		"iat": now.Unix(),
		"exp": now.Add(s.tokenTTL).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(s.tokenSecret))
	if err != nil {
		return nil, fmt.Errorf("sign jwt: %w", err)
	}

	return &LoginData{
		Token: signedToken,
	}, nil
}

// GetProfile retrieves the profile for an authenticated user by their ID.
func (s *Service) GetProfile(ctx context.Context, userID string) (*UserResponse, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get profile lookup: %w", err)
	}

	if user == nil {
		return nil, ErrUserNotFound
	}

	if !user.IsActive {
		return nil, ErrAccountInactive
	}

	username := ""
	if user.EmployeeCode != nil && *user.EmployeeCode != "" {
		username = *user.EmployeeCode
	} else if user.Email != nil {
		username = *user.Email
	}

	email := ""
	if user.Email != nil {
		email = *user.Email
	}

	return &UserResponse{
		ID:              user.ID,
		Username:        username,
		Name:            user.FullName,
		Email:           email,
		Role:            mapRoleCodeToEnum(user.RoleCode),
		ClientID:        user.ClientID,
		RoleDescription: user.RoleName,
	}, nil
}

// Logout terminates the current session for the user.
// In the current stateless JWT implementation, logout acknowledges session termination.
// Hook point for future token blocklist/revocation storage (e.g. Redis).
func (s *Service) Logout(ctx context.Context, userID string) error {
	if userID == "" {
		return ErrUserNotFound
	}
	return nil
}

func mapRoleCodeToEnum(roleCode string) string {
	switch strings.ToLower(roleCode) {
	case "owner_executive", "executive":
		return "OWNER"
	case "sample_adis", "adis_staff":
		return "TIM_SAMPLE"
	case "warehouse", "warehouse_staff":
		return "GUDANG"
	case "production_head":
		return "KEPALA_PRODUKSI"
	case "tailor":
		return "PENJAHIT"
	case "qc_inspector", "quality_control":
		return "QC_INSPECTOR"
	case "finance_payroll", "finance_admin":
		return "FINANCE_PAYROLL"
	case "client":
		return "CLIENT_PORTAL"
	default:
		return strings.ToUpper(roleCode)
	}
}
