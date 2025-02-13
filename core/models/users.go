package models

import (
	"database/sql"
)

type UserDetails struct {
	ID           int64        `json:"id" db:"id"`
	FirstName    string       `json:"first_name" db:"first_name"`
	LastName     string       `json:"last_name" db:"last_name"`
	EmailAddress string       `json:"email_address" db:"email_address"`
	CreatedAt    sql.NullTime `json:"created_at" db:"created_at"`
	DeletedAt    sql.NullTime `json:"deleted_at" db:"deleted_at"`
	MergedAt     sql.NullTime `json:"merged_at" db:"merged_at"`
	ParentUserId int64        `json:"parent_user_id" db:"parent_user_id"`
}
