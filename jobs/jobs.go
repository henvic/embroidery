package jobs

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/hashicorp/errwrap"
	"github.com/henvic/embroidery/server"
	"github.com/kisielk/sqlstruct"
	uuid "github.com/satori/go.uuid"
)

var db = server.Instance.DB

// Job of a Client
type Job struct {
	JobID      string  `schema:"job_id"`
	OrderID    string  `schema:"order_id"`
	ClientID   string  `schema:"client_id"`
	AssetID    string  `schema:"asset_id"`
	Status     string  `schema:"status"`
	Type       string  `schema:"type"`
	Amount     int     `schema:"amount"`
	Price      int64   `schema:"price"`
	StartTime  *string `schema:"start_time"`
	EndTime    *string `schema:"end_time"`
	Complexity int64   `schema:"amount"`
}

// ListFilter sets the filter settings
type ListFilter struct {
	ClientID string
	OrderID  string
	Status   string
}

// List job
func List(ctx context.Context, f ListFilter) (job []Job, err error) {
	var q = "SELECT job_id,order_id,client_id,asset_id,status,type,amount,price,start_time,end_time,complexity FROM `job`"
	var i []interface{}

	// horrible 'WHERE'...
	if f.ClientID != "" || f.OrderID != "" || f.Status != "" {
		q += " WHERE"
	}

	if f.ClientID != "" {
		q += " client_id = ?"
		i = append(i, f.ClientID)

		if f.OrderID != "" || f.Status != "" {
			q += " AND"
		}
	}

	if f.Status != "" {
		q += " status = ?"
		i = append(i, f.Status)
	}

	if f.OrderID != "" && f.Status != "" {
		q += " AND"
	}

	if f.OrderID != "" {
		q += " order_id = ?"
		i = append(i, f.OrderID)
	}

	q += " ORDER BY start_time DESC"

	stmt, err := db().PrepareContext(ctx, q)

	if err != nil {
		return nil, errwrap.Wrapf("Error preparing job query: {{err}}", err)
	}

	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, i...)

	if err != nil {
		return nil, errwrap.Wrapf("Error querying job: {{err}}", err)
	}

	for rows.Next() {
		var a Job
		err = sqlstruct.Scan(&a, rows)

		if err != nil {
			errwrap.Wrapf("Error scanning job rows: {{err}}", err)
			break
		}

		job = append(job, a)
	}

	return job, err
}

func updateOrderPrice(ctx context.Context, tx *sql.Tx, orderID string, priceDelta int64) error {
	var q = "UPDATE `order` SET price_total = price_total + ? WHERE order_id = ?"
	stmt, err := tx.PrepareContext(ctx, q)

	if err != nil {
		return errwrap.Wrapf("Error preparing order price_total update query: {{err}}", err)
	}

	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, priceDelta, orderID)
	return err
}

// Insert job on database
func Insert(ctx context.Context, job Job) (uid string, err error) {
	// we don't want to spend all time here if something goes wrong
	// let's try for one second
	// if it doesn't work it means a bigger problem is happening...
	var ctxTransaction, cancel = context.WithTimeout(ctx, time.Second)
	defer cancel()
	tx, err := db().BeginTx(ctxTransaction, nil)
	defer tx.Rollback()

	if err != nil {
		return "", err
	}

	uid, err = insert(ctxTransaction, tx, job)

	if err != nil {
		return "", err
	}

	if err := updateOrderPrice(ctxTransaction, tx, job.OrderID, job.Price); err != nil {
		return "", err
	}

	return uid, tx.Commit()
}

func insert(ctx context.Context, tx *sql.Tx, job Job) (uid string, err error) {
	uid = uuid.NewV4().String()

	var query = `INSERT INTO job (
		job_id,
		order_id,
		client_id,
		asset_id,
		status,
		type,
		amount,
		price,
		complexity
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	stmt, err := tx.PrepareContext(ctx, query)

	if err != nil {
		return "", err
	}

	defer stmt.Close()

	_, err = stmt.ExecContext(
		ctx,
		uid,
		job.OrderID,
		job.ClientID,
		job.AssetID,
		"CREATED",
		job.Type,
		job.Amount,
		job.Price,
		job.Complexity,
	)

	if err != nil {
		return "", err
	}

	return uid, err
}

// UpdateStatus of a job
func UpdateStatus(ctx context.Context, job Job) error {
	var oldJob, err = Get(ctx, job.JobID)

	if err != nil {
		return err
	}

	var q = "UPDATE `job` SET "

	var i []interface{}
	job.Status = strings.ToUpper(job.Status)

	if job.Status != oldJob.Status {
		q += "status = ?, "
		i = append(i, job.Status)

		if job.Status == "DONE" {
			q += "end_time = CURRENT_TIMESTAMP, "
		}

		if job.Status == "IN_PROGRESS" && job.StartTime == nil {
			q += "start_time = CURRENT_TIMESTAMP, "
		}
	}

	q += "asset_id = ? WHERE job_id = ?"

	stmt, err := db().PrepareContext(ctx, q)

	if err != nil {
		return errwrap.Wrapf("Error preparing employee update query: {{err}}", err)
	}

	defer stmt.Close()

	i = append(i, job.AssetID, job.JobID)
	_, err = stmt.ExecContext(ctx, i...)

	return err
}

// Get job by ID
func Get(ctx context.Context, jobID string) (Job, error) {
	stmt, err := db().PrepareContext(ctx,
		`SELECT job_id,order_id,client_id,asset_id,status,type,amount,price,start_time,end_time,complexity FROM job WHERE job_id = ?`)

	if err != nil {
		return Job{}, errwrap.Wrapf("Error preparing job query: {{err}}", err)
	}

	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, jobID)

	if err != nil {
		return Job{}, errwrap.Wrapf("Error querying job: {{err}}", err)
	}

	var job Job

	if ok := rows.Next(); !ok {
		return job, sql.ErrNoRows
	}

	if err := sqlstruct.Scan(&job, rows); err != nil {
		return job, errwrap.Wrapf("Error scanning job rows: {{err}}", err)
	}

	return job, nil
}

// GetStatusFilter for jobs
func GetStatusFilter() map[string]string {
	return allStatusFilter
}

var allStatusFilter = map[string]string{
	"":            "all",
	"created":     "created",
	"queue":       "queue",
	"in_progress": "in progress",
	"canceled":    "canceled",
	"done":        "done",
}
