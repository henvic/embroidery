package goods

import (
	"context"
	"database/sql"

	"github.com/hashicorp/errwrap"
	"github.com/henvic/embroidery/server"
	"github.com/kisielk/sqlstruct"
	uuid "github.com/satori/go.uuid"
)

var db = server.Instance.DB

// Good of a Client
type Good struct {
	GoodID     string `schema:"good_id"`
	JobID      string `schema:"job_id"`
	EmployeeID string `schema:"employee_id"`
	OwnerID    string `schema:"owner_id"`
	Type       string `schema:"type"`
	Amount     int    `schema:"amount"`
	Unit       string `schema:"unit"`
	Notes      string `schema:"notes"`
	Date       string `schema:"date"`
	Status     string `schema:"status"`
}

// ListFilter sets the filter settings
type ListFilter struct {
	OwnerID string
	JobID   string
	Status  string
}

// List good
func List(ctx context.Context, f ListFilter) (good []Good, err error) {
	var q = "SELECT good_id,job_id,employee_id,owner_id,type,amount,unit,notes,`date`,status FROM `goods`"
	var i []interface{}

	// horrible 'WHERE'...
	if f.OwnerID != "" || f.JobID != "" || f.Status != "" {
		q += " WHERE"
	}

	if f.OwnerID != "" {
		q += " owner_id = ?"
		i = append(i, f.OwnerID)

		if f.OwnerID != "" || f.Status != "" {
			q += " AND"
		}
	}

	if f.Status != "" {
		q += " status = ?"
		i = append(i, f.Status)
	}

	if f.JobID != "" && f.Status != "" {
		q += " AND"
	}

	if f.JobID != "" {
		q += " job_id = ?"
		i = append(i, f.JobID)
	}

	q += " ORDER BY `date` DESC"

	stmt, err := db().PrepareContext(ctx, q)

	if err != nil {
		return nil, errwrap.Wrapf("Error preparing good query: {{err}}", err)
	}

	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, i...)

	if err != nil {
		return nil, errwrap.Wrapf("Error querying good: {{err}}", err)
	}

	for rows.Next() {
		var a Good
		err = sqlstruct.Scan(&a, rows)

		if err != nil {
			errwrap.Wrapf("Error scanning good rows: {{err}}", err)
			break
		}

		good = append(good, a)
	}

	return good, err
}

// Insert good on database
func Insert(ctx context.Context, good Good) (uid string, err error) {
	var query = `INSERT INTO goods (
		good_id,
		job_id,
		employee_id,
		owner_id,
		type,
		amount,
		unit,
		notes,
		status,
		date
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`

	stmt, err := db().PrepareContext(ctx, query)

	if err != nil {
		return "", err
	}

	defer stmt.Close()

	id := uuid.NewV4().String()
	_, err = stmt.ExecContext(
		ctx,
		id,
		good.JobID,
		good.EmployeeID,
		good.OwnerID,
		good.Type,
		good.Amount,
		good.Unit,
		good.Notes,
		good.Status,
	)

	if err != nil {
		return "", err
	}

	return id, err
}

// Update an good
func Update(ctx context.Context, good Good) error {
	var q = "UPDATE `goods` SET type = ?, amount = ?, unit = ?, notes = ?, status = ? WHERE good_id = ?"

	stmt, err := db().PrepareContext(ctx, q)

	if err != nil {
		return errwrap.Wrapf("Error preparing employee update query: {{err}}", err)
	}

	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, good.Type, good.Amount, good.Unit, good.Notes, good.Status, good.GoodID)

	return err
}

// Get good by ID
func Get(ctx context.Context, goodID string) (Good, error) {
	stmt, err := db().PrepareContext(ctx,
		`SELECT good_id,job_id,employee_id,owner_id,type,amount,unit,notes,date,status FROM goods WHERE good_id = ?`)

	if err != nil {
		return Good{}, errwrap.Wrapf("Error preparing good query: {{err}}", err)
	}

	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, goodID)

	if err != nil {
		return Good{}, errwrap.Wrapf("Error querying good: {{err}}", err)
	}

	var good Good

	if ok := rows.Next(); !ok {
		return good, sql.ErrNoRows
	}

	if err := sqlstruct.Scan(&good, rows); err != nil {
		return good, errwrap.Wrapf("Error scanning good rows: {{err}}", err)
	}

	return good, nil
}

// GetAvailableTypes for goods
func GetAvailableTypes() map[string]string {
	return availableTypes
}

var availableTypes = map[string]string{
	"towel":   "towel",
	"line":    "line",
	"shirt":   "shirt",
	"uniform": "uniform",
	"other":   "other",
}

// GetAvailableUnits for goods
func GetAvailableUnits() map[string]string {
	return availableUnits
}

var availableUnits = map[string]string{
	"mm":        "mm",
	"square_cm": "square cm",
	"ml":        "ml",
	"units":     "units",
}

// GetStatusFilter for goods
func GetStatusFilter() map[string]string {
	return allStatusFilter
}

var allStatusFilter = map[string]string{
	"":               "all",
	"acquired":       "acquired",
	"in_stock":       "in stock",
	"in_use":         "in use",
	"missing":        "missing",
	"decommissioned": "decommissioned",
}
