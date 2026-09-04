package repository

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserBanned         = errors.New("user is banned")
)

type User struct {
	ID       string
	Email    string
	Password string
	RoleID   int
	Status   bool
	Rules    []string
}

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Login(ctx context.Context, email, password string) (User, error) {
	var user User
	err := r.pool.QueryRow(ctx, `
		SELECT user_id::text, email, password, role, status
		FROM t_user
		WHERE email = $1
	`, email).Scan(&user.ID, &user.Email, &user.Password, &user.RoleID, &user.Status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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
	rows, err := r.pool.Query(ctx, `
		SELECT ar.code
		FROM t_role_acl_rule rar
		JOIN t_acl_rule ar ON ar.acl_rule_id = rar.acl_rule_id
		WHERE rar.role_id = $1 AND ar.active = TRUE
		ORDER BY ar.code ASC
	`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules := make([]string, 0)
	for rows.Next() {
		var rule string
		if err := rows.Scan(&rule); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	return rules, rows.Err()
}

func hashPassword(password, salt string) string {
	sum := sha1.Sum([]byte(password + salt))
	return hex.EncodeToString(sum[:])
}
