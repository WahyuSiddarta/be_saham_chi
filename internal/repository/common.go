package repository

import "github.com/jmoiron/sqlx"

type Repository struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// scanRows reads struct rows without changing ownership of the rows or transaction.
func scanRows[T any](rows *sqlx.Rows) ([]T, error) {
	items := make([]T, 0)
	for rows.Next() {
		var item T
		if err := rows.StructScan(&item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
