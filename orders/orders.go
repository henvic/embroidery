package orders

import (
	"context"
	"database/sql"

	"github.com/hashicorp/errwrap"
	"github.com/henvic/embroidery/server"
	"github.com/kisielk/sqlstruct"
	uuid "github.com/satori/go.uuid"
)

var db = server.Instance.DB

// Order of a Client
type Order struct {
	OrderID         string  `schema:"order_id"`
	ClientID        string  `schema:"client_id"`
	ClientAddressID string  `schema:"client_address_id"`
	OpenTime        string  `schema:"open_time"`
	CloseTime       *string `schema:"close_time"`
	Status          string  `schema:"status"`
	PriceTotal      int64   `schema:"price_total"`
}

// ListFilter sets the filter settings
type ListFilter struct {
	ClientID string
	Status   string
}

// List order
func List(ctx context.Context, f ListFilter) (order []Order, err error) {
	var q = "SELECT order_id,client_id,client_address_id,open_time,close_time,status,price_total FROM `order`"
	var i []interface{}

	// horrible 'WHERE'...
	if f.ClientID != "" || f.Status != "" {
		q += " WHERE"
	}

	if f.Status != "" {
		q += " status = ?"
		i = append(i, f.Status)
	}

	if f.ClientID != "" && f.Status != "" {
		q += " AND"
	}

	if f.ClientID != "" {
		q += " client_id = ?"
		i = append(i, f.ClientID)
	}

	q += " ORDER BY open_time DESC"

	stmt, err := db().PrepareContext(ctx, q)

	if err != nil {
		return nil, errwrap.Wrapf("Error preparing order query: {{err}}", err)
	}

	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, i...)

	if err != nil {
		return nil, errwrap.Wrapf("Error querying order: {{err}}", err)
	}

	for rows.Next() {
		var a Order
		err = sqlstruct.Scan(&a, rows)

		if err != nil {
			errwrap.Wrapf("Error scanning order rows: {{err}}", err)
			break
		}

		order = append(order, a)
	}

	return order, err
}

// Insert order on database
func Insert(ctx context.Context, order Order) (uid string, err error) {
	var query = `INSERT INTO ` + "`order`" + ` (
		order_id,
		client_id,
		client_address_id,
		open_time,
		close_time,
		status,
		price_total
		)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ?, 0)`

	stmt, err := db().PrepareContext(ctx, query)

	if err != nil {
		return "", err
	}

	defer stmt.Close()

	id := uuid.NewV4().String()
	_, err = stmt.ExecContext(
		ctx,
		id,
		order.ClientID,
		order.ClientAddressID,
		"OPEN",
	)

	if err != nil {
		return "", err
	}

	return id, err
}

// Update an order
func Update(ctx context.Context, order Order, newStatus, newAddressID string) error {
	var q = "UPDATE `order` SET "

	var i []interface{}

	if order.Status != newStatus {
		q += "status = ?, "
		i = append(i, newStatus)

		if newStatus == "done" {
			q += "close_time = CURRENT_TIMESTAMP, "
		}
	}

	q += "client_address_id = ? WHERE order_id = ?"

	stmt, err := db().PrepareContext(ctx, q)

	if err != nil {
		return errwrap.Wrapf("Error preparing employee update query: {{err}}", err)
	}

	defer stmt.Close()

	i = append(i, newAddressID, order.OrderID)
	_, err = stmt.ExecContext(ctx, i...)

	return err
}

// Get order by ID
func Get(ctx context.Context, orderID string) (Order, error) {
	stmt, err := db().PrepareContext(ctx,
		`SELECT order_id,client_id,client_address_id,open_time,close_time,status,price_total
FROM `+"`order`"+` WHERE order_id = ?`)

	if err != nil {
		return Order{}, errwrap.Wrapf("Error preparing order query: {{err}}", err)
	}

	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, orderID)

	if err != nil {
		return Order{}, errwrap.Wrapf("Error querying order: {{err}}", err)
	}

	var order Order

	if ok := rows.Next(); !ok {
		return order, sql.ErrNoRows
	}

	if err := sqlstruct.Scan(&order, rows); err != nil {
		return order, errwrap.Wrapf("Error scanning order rows: {{err}}", err)
	}

	return order, nil
}

// GetStatusFilter for orders
func GetStatusFilter() map[string]string {
	return allStatusFilter
}

var allStatusFilter = map[string]string{
	"":                    "all",
	"open":                "open",
	"waiting_for_payment": "waiting for payment",
	"stand_by":            "stand by",
	"queue":               "queue",
	"in_progress":         "in progress",
	"canceled":            "canceled",
	"done":                "done",
}
