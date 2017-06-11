package auth

import (
	"context"
	"database/sql"

	"github.com/hashicorp/errwrap"
	"github.com/henvic/embroidery/server"
	"github.com/kisielk/sqlstruct"
)

var db = server.Instance.DB

// Authentication object
type Authentication struct {
	EmployeeID  string `schema:"employee_id"`
	Email       string `schema:"email"`
	AccessLevel string `schema:"access_level"`
	Password    string `schema:"password"`
}

// GetAuthenticationByEmail of users that are not revoked
func GetAuthenticationByEmail(ctx context.Context, email string) (Authentication, error) {
	stmt, err := db().PrepareContext(ctx,
		"SELECT employee_id,email,password,access_level FROM authentication WHERE email = ? AND access_level != 'REVOKED'")

	if err != nil {
		return Authentication{}, errwrap.Wrapf("Error preparing authentication query: {{err}}", err)
	}

	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, email)

	if err != nil {
		return Authentication{}, errwrap.Wrapf("Error querying authentication: {{err}}", err)
	}

	var auth Authentication

	if ok := rows.Next(); !ok {
		return auth, sql.ErrNoRows
	}

	if err := sqlstruct.Scan(&auth, rows); err != nil {
		return Authentication{}, errwrap.Wrapf("Error scanning authentication rows: {{err}}", err)
	}

	return auth, nil
}
