package repository

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"
	"uuid"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserBanned         = errors.New("user is banned")
)

func (r *Repository) Register(ctx context.Context, email string, password string) (User, Portfolio, error) {
	userID := uuid.NewV7().String()
	user := User{
		ID:       userID,
		Email:    email,
		Password: hashPassword(password, userID),
		Role:     1,
		RoleID:   1,
		Status:   true,
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return User{}, Portfolio{}, err
	}
	defer tx.Rollback()

	err = tx.GetContext(ctx, &user.CreatedAt, `
		INSERT INTO t_user (user_id, email, password, role, role_id, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at
	`, user.ID, user.Email, user.Password, user.Role, user.RoleID, user.Status)
	if err != nil {
		return User{}, Portfolio{}, err
	}

	var portfolio Portfolio
	err = tx.GetContext(ctx, &portfolio, `
		INSERT INTO portfolios (user_id, name, base_currency_code, is_main)
		VALUES ($1, 'main', 'IDR', TRUE)
		RETURNING portfolio_id::text AS portfolio_id, user_id::text AS user_id, name AS name, base_currency_code AS base_currency_code, is_main AS is_main, created_at AS created_at, updated_at AS updated_at `, user.ID)
	if err != nil {
		return User{}, Portfolio{}, err
	}

	if err := tx.Commit(); err != nil {
		return User{}, Portfolio{}, err
	}

	return user, portfolio, nil
}

func (r *Repository) Login(ctx context.Context, email string, password string) (User, error) {
	var user User
	err := r.db.GetContext(ctx, &user, `
		SELECT user_id::text AS user_id, email, password, created_at, role, COALESCE(role_id, role) AS role_id, status FROM t_user
		WHERE email = $1
	`, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrInvalidCredentials
		}
		return User{}, err
	}

	if user.Password != hashPassword(password, user.ID) {
		return User{}, ErrInvalidCredentials
	}
	if !user.Status {
		return User{}, ErrUserBanned
	}

	// `role` is the existing account role field. Keep RoleID aligned so ACL lookup,
	// login responses, and JWT claims all use the same effective role.
	user.RoleID = user.Role
	rules, err := r.listRoleRules(ctx, user.Role)
	if err != nil {
		return User{}, err
	}
	user.Rules = rules

	return user, nil
}

func (r *Repository) listRoleRules(ctx context.Context, roleID int) ([]string, error) {
	rules := make([]string, 0)
	err := r.db.SelectContext(ctx, &rules, `
SELECT ar.code FROM t_role_acl_rule rar JOIN t_acl_rule ar ON ar.acl_rule_id=rar.acl_rule_id
WHERE rar.role_id=$1 AND ar.active=TRUE ORDER BY ar.code ASC
`, roleID)
	return rules, err
}

func hashPassword(password, salt string) string {
	sum := sha1.Sum([]byte(password + salt))
	return hex.EncodeToString(sum[:])
}

type User struct {
	ID        string    `db:"user_id" json:"user_id"`
	Email     string    `db:"email" json:"email"`
	Password  string    `db:"password" json:"-"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	Role      int       `db:"role"`
	RoleID    int       `db:"role_id" json:"role_id"`
	Status    bool      `db:"status" json:"status"`
	Rules     []string  `db:"-" json:"rules"`
}
