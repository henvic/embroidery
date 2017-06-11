package payment

import (
	"context"
	"database/sql"

	"github.com/hashicorp/errwrap"
	"github.com/henvic/embroidery/server"
	"github.com/kisielk/sqlstruct"
	uuid "github.com/satori/go.uuid"
)

var db = server.Instance.DB

// Payment of order
type Payment struct {
	PaymentID  string `schema:"payment_id"`
	ClientID   string `schema:"client_id"`
	OrderID    string `schema:"order_id"`
	PriceTotal int64  `schema:"price_total"`
	Provider   string `schema:"provider"`
	Date       string `schema:"date"`
}

// ListFilter sets the filter settings
type ListFilter struct {
	ClientID string
	OrderID  string
	Provider string
}

// List payment
func List(ctx context.Context, f ListFilter) (payment []Payment, err error) {
	var q = "SELECT payment_id,client_id,order_id,price_total,provider,date FROM `payment`"
	var i []interface{}

	// horrible 'WHERE'...
	if f.ClientID != "" || f.OrderID != "" || f.Provider != "" {
		q += " WHERE"
	}

	if f.ClientID != "" {
		q += " client_id = ?"
		i = append(i, f.ClientID)

		if f.OrderID != "" || f.Provider != "" {
			q += " AND"
		}
	}

	if f.Provider != "" {
		q += " provider = ?"
		i = append(i, f.Provider)
	}

	if f.OrderID != "" && f.Provider != "" {
		q += " AND"
	}

	if f.OrderID != "" {
		q += " order_id = ?"
		i = append(i, f.OrderID)
	}

	q += " ORDER BY `date` DESC"

	stmt, err := db().PrepareContext(ctx, q)

	if err != nil {
		return nil, errwrap.Wrapf("Error preparing payment query: {{err}}", err)
	}

	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, i...)

	if err != nil {
		return nil, errwrap.Wrapf("Error querying payment: {{err}}", err)
	}

	for rows.Next() {
		var a Payment
		err = sqlstruct.Scan(&a, rows)

		if err != nil {
			errwrap.Wrapf("Error scanning payment rows: {{err}}", err)
			break
		}

		payment = append(payment, a)
	}

	return payment, err
}

// Insert payment on database
func Insert(ctx context.Context, payment Payment) (uid string, err error) {
	var query = `INSERT INTO payment (
		payment_id,
		client_id,
		order_id,
		price_total,
		provider,
		date
		)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`

	stmt, err := db().PrepareContext(ctx, query)

	if err != nil {
		return "", err
	}

	defer stmt.Close()

	id := uuid.NewV4().String()
	_, err = stmt.ExecContext(
		ctx,
		id,
		payment.ClientID,
		payment.OrderID,
		payment.PriceTotal,
		payment.Provider,
	)

	if err != nil {
		return "", err
	}

	return id, err
}

// Get payment by ID
func Get(ctx context.Context, paymentID string) (Payment, error) {
	stmt, err := db().PrepareContext(ctx,
		`SELECT payment_id,client_id,order_id,price_total,provider,date FROM payment WHERE payment_id = ?`)

	if err != nil {
		return Payment{}, errwrap.Wrapf("Error preparing payment query: {{err}}", err)
	}

	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, paymentID)

	if err != nil {
		return Payment{}, errwrap.Wrapf("Error querying payment: {{err}}", err)
	}

	var payment Payment

	if ok := rows.Next(); !ok {
		return payment, sql.ErrNoRows
	}

	if err := sqlstruct.Scan(&payment, rows); err != nil {
		return payment, errwrap.Wrapf("Error scanning payment rows: {{err}}", err)
	}

	return payment, nil
}

// GetProvidersFilter for payment providers
func GetProvidersFilter() map[string]string {
	return providersFilter
}

var providersFilter = map[string]string{
	"":               "all",
	"cash_flow":      "cash flow",
	"credit_card":    "credit card",
	"debit_card":     "debit card",
	"money_transfer": "money transfer",
}
