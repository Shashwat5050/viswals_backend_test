package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/lib/pq"
	"github.com/viswals_backend_test/core/models"
	"log"
	"strings"
)

var (
	ErrNoData          = errors.New("requested data does not exist")
	ErrDuplicate       = errors.New("data to create already exists")
	DefaultFieldsCount = 8
)

// using default exec/query syntax as go package internally has protection for SQL injection. (alternatively, we can use a prepare statement)

type Database struct {
	db *sql.DB
}

func New(connectionUrl string) (*Database, error) {
	db, err := sql.Open("postgres", connectionUrl)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(2)

	return &Database{
		db: db,
	}, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) CreateUser(ctx context.Context, userDetails *models.UserDetails) error {
	// insert data in database.
	_, err := d.db.ExecContext(ctx, "INSERT INTO user_details (id,first_name,last_name,email_address,created_at,deleted_at,merged_at,parent_user_id) VALUES ($1,$2,$3,$4,$5,$6,$7,$8);", userDetails.ID, userDetails.FirstName, userDetails.LastName, userDetails.EmailAddress, userDetails.CreatedAt, userDetails.DeletedAt, userDetails.MergedAt, userDetails.ParentUserId)
	if err != nil {
		// check for data already exists.
		var e *pq.Error
		if errors.As(err, &e) && e.Code == "23505" {
			return ErrDuplicate
		}
		return err
	}
	return nil
}

func (d *Database) CreateBulkUsers(ctx context.Context, userDetails []*models.UserDetails) error {

	query := "INSERT INTO user_details (id,first_name,last_name,email_address,created_at,deleted_at,merged_at,parent_user_id) VALUES "

	var queryHolders = make([]string, 0, len(userDetails)*DefaultFieldsCount)
	var valueHolder = make([]interface{}, 0, len(userDetails)*DefaultFieldsCount)

	for i, user := range userDetails {
		queryHolders = append(queryHolders, fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)", i*DefaultFieldsCount+1, i*DefaultFieldsCount+2, i*DefaultFieldsCount+3, i*DefaultFieldsCount+4, i*DefaultFieldsCount+5, i*DefaultFieldsCount+6, i*DefaultFieldsCount+7, i*DefaultFieldsCount+8))
		valueHolder = append(valueHolder, user.ID, user.FirstName, user.LastName, user.EmailAddress, user.CreatedAt, user.DeletedAt, user.MergedAt, user.ParentUserId)
	}

	query += strings.Join(queryHolders, ",")

	// insert data in database.
	_, err := d.db.ExecContext(ctx, query, valueHolder...)
	if err != nil {
		log.Println(err)
		// check for data already exists.
		var e *pq.Error
		if errors.As(err, &e) && e.Code == "23505" {
			return ErrDuplicate
		}
		return err
	}
	return nil
}

func (d *Database) GetUserByID(ctx context.Context, id string) (*models.UserDetails, error) {
	var userDetails models.UserDetails

	row := d.db.QueryRowContext(ctx, "SELECT id,first_name,last_name,email_address,created_at,deleted_at,merged_at,parent_user_id FROM user_details WHERE id = $1;", id)

	err := row.Scan(&userDetails.ID, &userDetails.FirstName, &userDetails.LastName, &userDetails.EmailAddress, &userDetails.CreatedAt, &userDetails.DeletedAt, &userDetails.MergedAt, &userDetails.ParentUserId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoData
		}
		return nil, err
	}
	return &userDetails, nil
}

func (d *Database) GetAllUsers(ctx context.Context) ([]*models.UserDetails, error) {
	var userDetails []*models.UserDetails
	rows, err := d.db.QueryContext(ctx, "SELECT id,first_name,last_name,email_address,created_at,deleted_at,merged_at,parent_user_id FROM user_details")
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var userDetail models.UserDetails
		err := rows.Scan(&userDetail.ID, &userDetail.FirstName, &userDetail.LastName, &userDetail.EmailAddress, &userDetail.CreatedAt, &userDetail.DeletedAt, &userDetail.MergedAt, &userDetail.ParentUserId)
		if err != nil {
			return nil, err
		}

		userDetails = append(userDetails, &userDetail)
	}

	return userDetails, nil
}

func (d *Database) ListUsers(ctx context.Context, limit, offset int64) ([]*models.UserDetails, error) {
	var userDetails []*models.UserDetails

	rows, err := d.db.QueryContext(ctx, "SELECT id,first_name,last_name,email_address,created_at,deleted_at,merged_at,parent_user_id FROM user_details ORDER BY id LIMIT $1 OFFSET $2;", limit, offset)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoData
		}
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var userDetail models.UserDetails
		err := rows.Scan(&userDetail.ID, &userDetail.FirstName, &userDetail.LastName, &userDetail.EmailAddress, &userDetail.CreatedAt, &userDetail.DeletedAt, &userDetail.MergedAt, &userDetail.ParentUserId)
		if err != nil {
			return nil, err
		}

		userDetails = append(userDetails, &userDetail)
	}

	return userDetails, nil
}

func (d *Database) DeleteUser(ctx context.Context, id string) error {
	_, err := d.db.ExecContext(ctx, "DELETE FROM user_details WHERE id = $1;", id)
	if err != nil {
		return err
	}

	return nil
}

func (d *Database) Migrate(databaseName string) error {
	driver, err := postgres.WithInstance(d.db, &postgres.Config{})
	if err != nil {
		return err
	}

	// path is relative from cmd dir as mentioned in dockerFile.
	m, err := migrate.NewWithDatabaseInstance("file://./migration", databaseName, driver)
	if err != nil {
		return err
	}

	if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	v, dirty, err := m.Version()
	if err != nil {
		return err
	}

	if dirty {
		return errors.New(fmt.Sprintf("Migration ERROR: version %v is in dirty state please solve.", v))
	}

	return nil
}
