package repository

import (
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"errors"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserBanned         = errors.New("user is banned")
)

type User struct {
	ID       string `db:"user_id"`
	Email    string `db:"email"`
	Password string `db:"password"`
	RoleID   int    `db:"role"`
	Status   bool   `db:"status"`
	Rules    []string
}

func (r *Repository) Login(ctx context.Context, email, password string) (User, error) {
	var user User
	err := r.db.GetContext(ctx, &user, `
		SELECT user_id::text AS user_id, email, password, role, status
		FROM t_user
		WHERE email = $1
	`, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrInvalidCredentials
		}
		return User{}, err
	}

	// Retains be_saham_v2's existing password format for account compatibility.
	if user.Password != hashPassword(password, user.ID) {
		return User{}, ErrInvalidCredentials
	}
	if !user.Status {
		return User{}, ErrUserBanned
	}

	rules, err := r.listRoleRules(ctx, user.RoleID)
	if err != nil {
		return User{}, err
	}
	user.Rules = rules

	return user, nil
}

func (r *Repository) listRoleRules(ctx context.Context, roleID int) ([]string, error) {
	rules := make([]string, 0)
	err := r.db.SelectContext(ctx, &rules, `
		SELECT ar.code
		FROM t_role_acl_rule rar
		JOIN t_acl_rule ar ON ar.acl_rule_id = rar.acl_rule_id
		WHERE rar.role_id = $1 AND ar.active = TRUE
		ORDER BY ar.code ASC
	`, roleID)
	if err != nil {
		return nil, err
	}

	return rules, nil
}

func hashPassword(password, salt string) string {
	sum := sha1.Sum([]byte(password + salt))
	return hex.EncodeToString(sum[:])
}
