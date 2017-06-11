package address

import (
	"context"
	"database/sql"

	"github.com/hashicorp/errwrap"
	"github.com/henvic/embroidery/server"
	"github.com/kisielk/sqlstruct"
	uuid "github.com/satori/go.uuid"
)

var db = server.Instance.DB

// Address of a Client
type Address struct {
	AddressID    string `schema:"address_id"`
	ClientID     string `schema:"client_id"`
	Name         string `schema:"name"`
	AddressLine1 string `schema:"address_line1"`
	AddressLine2 string `schema:"address_line2"`
	City         string `schema:"city"`
	State        string `schema:"state"`
	Country      string `schema:"country"`
	ZipCode      string `schema:"zip_code"`
	Phone        string `schema:"phone"`
	Status       string `schema:"status"`
}

// ListFilter sets the filter settings
type ListFilter struct {
	ClientID     string
	ShowArchived bool
}

// List addresses
func List(ctx context.Context, f ListFilter) (addresses []Address, err error) {
	var q = "SELECT address_id,client_id,name,address_line1,address_line2,city,state,country,zip_code,phone,status FROM address"

	// horrible 'WHERE'...
	if f.ClientID != "" || !f.ShowArchived {
		q += " WHERE"
	}

	if !f.ShowArchived {
		q += " status != 'ARCHIVED'"
	}

	if f.ClientID != "" && !f.ShowArchived {
		q += " AND"
	}

	var i []interface{}

	if f.ClientID != "" {
		q += " client_id = ?"
		i = append(i, f.ClientID)
	}

	q += " ORDER BY name DESC"

	stmt, err := db().PrepareContext(ctx, q)

	if err != nil {
		return nil, errwrap.Wrapf("Error preparing addresses query: {{err}}", err)
	}

	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, i...)

	if err != nil {
		return nil, errwrap.Wrapf("Error querying addresses: {{err}}", err)
	}

	for rows.Next() {
		var a Address
		err = sqlstruct.Scan(&a, rows)

		if err != nil {
			errwrap.Wrapf("Error scanning addresses rows: {{err}}", err)
			break
		}

		addresses = append(addresses, a)
	}

	return addresses, err
}

// Insert address on database
func Insert(ctx context.Context, address Address) (uid string, err error) {
	var query = `INSERT INTO address (
		address_id,
		client_id,
		name,
		address_line1,
		address_line2,
		city,
		state,
		country,
		zip_code,
		phone,
		status
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	stmt, err := db().PrepareContext(ctx, query)

	if err != nil {
		return "", err
	}

	defer stmt.Close()

	id := uuid.NewV4().String()
	_, err = stmt.ExecContext(
		ctx,
		id,
		address.ClientID,
		address.Name,
		address.AddressLine1,
		address.AddressLine2,
		address.City,
		address.State,
		address.Country,
		address.ZipCode,
		address.Phone,
		address.Status,
	)

	if err != nil {
		return "", err
	}

	return id, err
}

// Update address' data
func Update(ctx context.Context, address Address) error {
	stmt, err := db().PrepareContext(ctx,
		`UPDATE address SET 
name = ?, address_line1 = ?, address_line2 = ?, city = ?, state = ?, country = ?, zip_code = ?, phone = ?, status = ?
WHERE address_id = ?`)

	if err != nil {
		return errwrap.Wrapf("Error preparing employee update query: {{err}}", err)
	}

	defer stmt.Close()

	_, err = stmt.ExecContext(ctx,
		address.Name,
		address.AddressLine1,
		address.AddressLine2,
		address.City,
		address.State,
		address.Country,
		address.ZipCode,
		address.Phone,
		address.Status,
		address.AddressID,
	)

	return err
}

// Get address by ID
func Get(ctx context.Context, clientID, addressID string) (Address, error) {
	stmt, err := db().PrepareContext(ctx,
		`SELECT address_id,client_id,name,address_line1,address_line2,city,state,country,zip_code,phone,status
FROM address WHERE client_id = ? AND address_id = ?`)

	if err != nil {
		return Address{}, errwrap.Wrapf("Error preparing address query: {{err}}", err)
	}

	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, clientID, addressID)

	if err != nil {
		return Address{}, errwrap.Wrapf("Error querying address: {{err}}", err)
	}

	var address Address

	if ok := rows.Next(); !ok {
		return address, sql.ErrNoRows
	}

	if err := sqlstruct.Scan(&address, rows); err != nil {
		return address, errwrap.Wrapf("Error scanning address rows: {{err}}", err)
	}

	return address, nil
}
