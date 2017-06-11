package clients

import (
	"context"
	"database/sql"

	"github.com/hashicorp/errwrap"
	"github.com/henvic/embroidery/server"
	"github.com/kisielk/sqlstruct"
	uuid "github.com/satori/go.uuid"
)

var db = server.Instance.DB

// Client object
type Client struct {
	ClientID  string `schema:"client_id"`
	FirstName string `schema:"first_name"`
	LastName  string `schema:"last_name"`
	Email     string `schema:"email"`
	Status    string `schema:"status"`
}

// GetClientsMapFromSlice creates a map from a slice
func GetClientsMapFromSlice(cs []Client) (m map[string]Client) {
	m = map[string]Client{}

	for _, c := range cs {
		m[c.ClientID] = c
	}

	return m
}

// List clients
func List(ctx context.Context, f ListFilter) (clients []Client, err error) {
	var q = "SELECT client_id,first_name,last_name,email,status FROM clients"

	if !f.ShowArchived {
		q += " WHERE status != 'ARCHIVED'"
	}

	q += " ORDER BY first_name,last_name DESC"

	stmt, err := db().PrepareContext(ctx, q)

	if err != nil {
		return nil, errwrap.Wrapf("Error preparing clients query: {{err}}", err)
	}

	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx)

	if err != nil {
		return nil, errwrap.Wrapf("Error querying clients: {{err}}", err)
	}

	for rows.Next() {
		var c Client
		err = sqlstruct.Scan(&c, rows)

		if err != nil {
			errwrap.Wrapf("Error scanning clients rows: {{err}}", err)
			break
		}

		clients = append(clients, c)
	}

	return clients, err
}

// Insert on database
func Insert(ctx context.Context, client Client) (uid string, err error) {
	var query = "INSERT INTO clients (client_id, first_name, last_name, email, status) VALUES (?, ?, ?, ?, ?)"
	stmt, err := db().PrepareContext(ctx, query)

	if err != nil {
		return "", err
	}

	defer stmt.Close()

	id := uuid.NewV4().String()
	_, err = stmt.ExecContext(ctx, id, client.FirstName, client.LastName, client.Email, client.Status)

	if err != nil {
		return "", err
	}

	return id, err
}

// ListFilter sets the filter settings
type ListFilter struct {
	ShowArchived bool
}

// Get client by ID
func Get(ctx context.Context, clientID string) (Client, error) {
	stmt, err := db().PrepareContext(ctx,
		"SELECT client_id,first_name,last_name,email,status FROM clients WHERE client_id = ?")

	if err != nil {
		return Client{}, errwrap.Wrapf("Error preparing client query: {{err}}", err)
	}

	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, clientID)

	if err != nil {
		return Client{}, errwrap.Wrapf("Error querying client: {{err}}", err)
	}

	var client Client

	if ok := rows.Next(); !ok {
		return client, sql.ErrNoRows
	}

	if err := sqlstruct.Scan(&client, rows); err != nil {
		return client, errwrap.Wrapf("Error scanning client rows: {{err}}", err)
	}

	return client, nil
}

// Update client's data
func Update(ctx context.Context, client Client) error {
	stmt, err := db().PrepareContext(ctx, "UPDATE clients SET first_name = ?, last_name = ?, email = ?, status = ?  WHERE client_id = ?")

	if err != nil {
		return errwrap.Wrapf("Error preparing employee update query: {{err}}", err)
	}

	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, client.FirstName, client.LastName, client.Email, client.Status, client.ClientID)
	return err
}
