package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Tanmoy095/LogiSynapse/services/authentication-service/internal/domain/audit"
	domainErr "github.com/Tanmoy095/LogiSynapse/services/authentication-service/internal/domain/errors"
	"github.com/Tanmoy095/LogiSynapse/services/authentication-service/internal/domain/membership"
	"github.com/Tanmoy095/LogiSynapse/services/authentication-service/internal/domain/session"
	"github.com/Tanmoy095/LogiSynapse/services/authentication-service/internal/domain/tenant"
	"github.com/Tanmoy095/LogiSynapse/services/authentication-service/internal/domain/user"
	"github.com/Tanmoy095/LogiSynapse/services/authentication-service/internal/ports/repository"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

var (
	_ repository.UserStore         = (*PostgresStore)(nil)
	_ repository.TenantStore       = (*PostgresStore)(nil)
	_ repository.MemberShipStore   = (*PostgresStore)(nil)
	_ repository.RefreshTokenStore = (*PostgresStore)(nil)
	_ repository.AuditStore        = (*PostgresStore)(nil)
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func NewPostgresUserStore(db *sql.DB) *PostgresStore {
	return NewPostgresStore(db)
}

type runner interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (s *PostgresStore) dbRunner(ctx context.Context) runner {
	if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return tx
	}
	return s.db
}

func mapPQError(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "23505":
			if pqErr.Constraint == "unique_user_email" {
				return domainErr.ErrEmailAlreadyExists
			}
			if pqErr.Constraint == "unique_membership" {
				return domainErr.ErrDuplicateMembership
			}
		case "23503":
			return domainErr.ErrInvalidInput
		}
	}
	return err
}

func (s *PostgresStore) CreateUser(ctx context.Context, u *user.User) error {
	query := `INSERT INTO users (id, email, first_name, last_name, password_hash, status, is_super_admin, created_at, updated_at)
VALUES ($1, lower($2), $3, $4, $5, $6, $7, $8, $9)`
	_, err := s.dbRunner(ctx).ExecContext(ctx, query, u.UserID, u.UserEmail, u.FirstName, u.LastName, u.PasswordHash, u.Status, u.IsSuperAdmin, u.CreatedAt, u.UpdatedAt)
	return mapPQError(err)
}

func (s *PostgresStore) GetUserByEmail(ctx context.Context, email string) (*user.User, error) {
	query := `SELECT id, email, first_name, last_name, COALESCE(password_hash, ''), status, is_super_admin, created_at, updated_at
FROM users WHERE email = lower($1)`
	return s.scanUser(s.dbRunner(ctx).QueryRowContext(ctx, query, email))
}

func (s *PostgresStore) GetUserByID(ctx context.Context, userID uuid.UUID) (*user.User, error) {
	query := `SELECT id, email, first_name, last_name, COALESCE(password_hash, ''), status, is_super_admin, created_at, updated_at
FROM users WHERE id = $1`
	return s.scanUser(s.dbRunner(ctx).QueryRowContext(ctx, query, userID))
}

func (s *PostgresStore) scanUser(row *sql.Row) (*user.User, error) {
	var u user.User
	if err := row.Scan(&u.UserID, &u.UserEmail, &u.FirstName, &u.LastName, &u.PasswordHash, &u.Status, &u.IsSuperAdmin, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, domainErr.ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (s *PostgresStore) UpdateStatus(ctx context.Context, id uuid.UUID, status user.UserStatus) error {
	_, err := s.dbRunner(ctx).ExecContext(ctx, `UPDATE users SET status=$1 WHERE id=$2`, status, id)
	return err
}

func (s *PostgresStore) SetPasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) error {
	_, err := s.dbRunner(ctx).ExecContext(ctx, `UPDATE users SET password_hash=$1, password_changed_at=NOW() WHERE id=$2`, passwordHash, id)
	return err
}

func (s *PostgresStore) CreateTenantWithOwnership(ctx context.Context, t *tenant.Tenant) error {
	query := `INSERT INTO tenants (id, name, status, owner_user_id, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := s.dbRunner(ctx).ExecContext(ctx, query, t.TenantID, t.TenantName, t.TenantStatus, t.OwnerUserID, t.CreatedAt, t.UpdatedAt)
	return mapPQError(err)
}

func (s *PostgresStore) GetTenantByID(ctx context.Context, tenantID uuid.UUID) (*tenant.Tenant, error) {
	query := `SELECT id, name, status, owner_user_id, created_at, updated_at FROM tenants WHERE id=$1`
	var t tenant.Tenant
	if err := s.dbRunner(ctx).QueryRowContext(ctx, query, tenantID).Scan(&t.TenantID, &t.TenantName, &t.TenantStatus, &t.OwnerUserID, &t.CreatedAt, &t.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, domainErr.ErrTenantNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (s *PostgresStore) UpdateTenantStatus(ctx context.Context, tenantID uuid.UUID, status tenant.TenantStatus) error {
	_, err := s.dbRunner(ctx).ExecContext(ctx, `UPDATE tenants SET status=$1 WHERE id=$2`, status, tenantID)
	return err
}

func (s *PostgresStore) ListTenantsByOwnerID(ctx context.Context, ownerUserID uuid.UUID) ([]tenant.Tenant, error) {
	rows, err := s.dbRunner(ctx).QueryContext(ctx, `SELECT id, name, status, owner_user_id, created_at, updated_at FROM tenants WHERE owner_user_id=$1 ORDER BY created_at DESC`, ownerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tenants []tenant.Tenant
	for rows.Next() {
		var t tenant.Tenant
		if err := rows.Scan(&t.TenantID, &t.TenantName, &t.TenantStatus, &t.OwnerUserID, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tenants = append(tenants, t)
	}
	return tenants, rows.Err()
}

func (s *PostgresStore) UpdateTenant(ctx context.Context, t *tenant.Tenant) error {
	_, err := s.dbRunner(ctx).ExecContext(ctx, `UPDATE tenants SET name=$1, status=$2, owner_user_id=$3 WHERE id=$4`, t.TenantName, t.TenantStatus, t.OwnerUserID, t.TenantID)
	return err
}

func (s *PostgresStore) CreateMembership(ctx context.Context, m *membership.MemberShip) error {
	query := `INSERT INTO memberships (user_id, tenant_id, role, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := s.dbRunner(ctx).ExecContext(ctx, query, m.UserID, m.TenantID, m.MemberShipRole, m.MemberShipStatus, m.CreatedAt, m.UpdatedAt)
	return mapPQError(err)
}

func (s *PostgresStore) UpdateMembershipStatus(ctx context.Context, m *membership.MemberShip) error {
	_, err := s.dbRunner(ctx).ExecContext(ctx, `UPDATE memberships SET status=$1 WHERE user_id=$2 AND tenant_id=$3`, m.MemberShipStatus, m.UserID, m.TenantID)
	return err
}

func (s *PostgresStore) GetMembersByTenantID(ctx context.Context, tenantID uuid.UUID) ([]membership.MemberShip, error) {
	rows, err := s.dbRunner(ctx).QueryContext(ctx, `SELECT user_id, tenant_id, role, status, created_at, updated_at FROM memberships WHERE tenant_id=$1`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []membership.MemberShip
	for rows.Next() {
		m, err := scanMembership(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

func (s *PostgresStore) ListMembersByUserID(ctx context.Context, userID uuid.UUID) ([]*membership.MemberShip, error) {
	rows, err := s.dbRunner(ctx).QueryContext(ctx, `SELECT user_id, tenant_id, role, status, created_at, updated_at FROM memberships WHERE user_id=$1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*membership.MemberShip
	for rows.Next() {
		m, err := scanMembership(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *PostgresStore) GetMember(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) (*membership.MemberShip, error) {
	query := `SELECT user_id, tenant_id, role, status, created_at, updated_at FROM memberships WHERE user_id=$1 AND tenant_id=$2`
	var m membership.MemberShip
	if err := s.dbRunner(ctx).QueryRowContext(ctx, query, userID, tenantID).Scan(&m.UserID, &m.TenantID, &m.MemberShipRole, &m.MemberShipStatus, &m.CreatedAt, &m.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, domainErr.ErrMembershipNotFound
		}
		return nil, err
	}
	return &m, nil
}

func (s *PostgresStore) UpdateMemberRole(ctx context.Context, userID, tenantID uuid.UUID, role membership.Role) error {
	_, err := s.dbRunner(ctx).ExecContext(ctx, `UPDATE memberships SET role=$1 WHERE user_id=$2 AND tenant_id=$3`, role, userID, tenantID)
	return err
}

func (s *PostgresStore) UpsertMembership(ctx context.Context, m *membership.MemberShip) error {
	query := `INSERT INTO memberships (user_id, tenant_id, role, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id, tenant_id) DO UPDATE SET role=EXCLUDED.role, status=EXCLUDED.status`
	_, err := s.dbRunner(ctx).ExecContext(ctx, query, m.UserID, m.TenantID, m.MemberShipRole, m.MemberShipStatus, m.CreatedAt, m.UpdatedAt)
	return err
}

func scanMembership(rows *sql.Rows) (*membership.MemberShip, error) {
	var m membership.MemberShip
	if err := rows.Scan(&m.UserID, &m.TenantID, &m.MemberShipRole, &m.MemberShipStatus, &m.CreatedAt, &m.UpdatedAt); err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *PostgresStore) CreateRefreshToken(ctx context.Context, token *session.RefreshToken) error {
	query := `INSERT INTO refresh_tokens (id, user_id, tenant_id, token_hash, family_id, issued_at, expires_at, revoked_at, replaced_by_token_id, device_fingerprint)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := s.dbRunner(ctx).ExecContext(ctx, query, token.TokenID, token.UserID, token.TenantID, token.TokenHash, token.FamilyID, token.IssuedAt, token.ExpiresAt, token.RevokedAt, token.ReplacedBy, token.DeviceMetadata)
	return mapPQError(err)
}

func (s *PostgresStore) GetTokenByHash(ctx context.Context, hash string) (*session.RefreshToken, error) {
	query := `SELECT id, user_id, tenant_id, token_hash, family_id, issued_at, expires_at, revoked_at, replaced_by_token_id, COALESCE(device_fingerprint, '')
FROM refresh_tokens WHERE token_hash=$1`
	var token session.RefreshToken
	if err := s.dbRunner(ctx).QueryRowContext(ctx, query, hash).Scan(&token.TokenID, &token.UserID, &token.TenantID, &token.TokenHash, &token.FamilyID, &token.IssuedAt, &token.ExpiresAt, &token.RevokedAt, &token.ReplacedBy, &token.DeviceMetadata); err != nil {
		if err == sql.ErrNoRows {
			return nil, domainErr.ErrInvalidCredentials
		}
		return nil, err
	}
	return &token, nil
}

func (s *PostgresStore) RevokeTokenFamily(ctx context.Context, familyID uuid.UUID) error {
	_, err := s.dbRunner(ctx).ExecContext(ctx, `UPDATE refresh_tokens SET revoked_at=NOW() WHERE family_id=$1 AND revoked_at IS NULL`, familyID)
	return err
}

func (s *PostgresStore) RotateToken(ctx context.Context, oldTokenID uuid.UUID, newToken *session.RefreshToken) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	ctx = context.WithValue(ctx, txKey{}, tx)
	if err = s.CreateRefreshToken(ctx, newToken); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE refresh_tokens SET replaced_by_token_id=$1 WHERE id=$2`, newToken.TokenID, oldTokenID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *PostgresStore) Append(ctx context.Context, event *audit.AuditEvent) error {
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("marshal audit metadata: %w", err)
	}
	query := `INSERT INTO audit_logs (id, actor_user_id, tenant_id, action, target_id, ip_address, metadata, created_at)
VALUES ($1, $2, $3, $4, $5, NULLIF($6, '')::inet, $7::jsonb, $8)`
	_, err = s.dbRunner(ctx).ExecContext(ctx, query, event.ID, event.ActorUserID, event.TenantID, event.Action, event.TargetID, event.IPAddress, string(metadata), event.CreatedAt)
	return err
}
