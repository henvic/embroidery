package employees

import (
	"context"

	"database/sql"

	"github.com/hashicorp/errwrap"
	"github.com/henvic/embroidery/server"
	"github.com/kisielk/sqlstruct"
)

var db = server.Instance.DB

// Employee structure
type Employee struct {
	EmployeeID  string `schema:"employee_id"`
	Email       string `schema:"email"`
	AccessLevel string `schema:"access_level"`
	Password    string `schema:"password"`
}

// List employees
func List(ctx context.Context) (employees []Employee, err error) {
	stmt, err := db().PrepareContext(ctx, "SELECT employee_id,email,access_level FROM authentication")

	if err != nil {
		return nil, errwrap.Wrapf("Error preparing employee query: {{err}}", err)
	}

	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx)

	if err != nil {
		return nil, errwrap.Wrapf("Error querying employee: {{err}}", err)
	}

	for rows.Next() {
		var e Employee
		err = sqlstruct.Scan(&e, rows)

		if err != nil {
			errwrap.Wrapf("Error scanning Employee rows: {{err}}", err)
			break
		}

		employees = append(employees, e)
	}

	return employees, err
}

// Get gets an employee row by ID
func Get(ctx context.Context, employeeID string) (Employee, error) {
	stmt, err := db().PrepareContext(ctx,
		"SELECT employee_id,email,password,access_level FROM authentication WHERE employee_id = ?")

	if err != nil {
		return Employee{}, errwrap.Wrapf("Error preparing employee query: {{err}}", err)
	}

	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, employeeID)

	if err != nil {
		return Employee{}, errwrap.Wrapf("Error querying employee: {{err}}", err)
	}

	var employee Employee

	if ok := rows.Next(); !ok {
		return employee, sql.ErrNoRows
	}

	if err := sqlstruct.Scan(&employee, rows); err != nil {
		return employee, errwrap.Wrapf("Error scanning employee rows: {{err}}", err)
	}

	return employee, nil
}

// Create employee
func Create(ctx context.Context, employee Employee) error {
	stmt, err := db().PrepareContext(ctx, "INSERT INTO authentication (employee_id, email, password, access_level) VALUES (?, ?, ?, ?)")

	if err != nil {
		return errwrap.Wrapf("Error preparing employee update query: {{err}}", err)
	}

	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, employee.EmployeeID, employee.Email, employee.Password, employee.AccessLevel)
	return err
}

// Update employee's data
func Update(ctx context.Context, employee Employee) error {
	stmt, err := db().PrepareContext(ctx, "UPDATE authentication SET email = ?, password = ?, access_level = ? WHERE employee_id = ?")

	if err != nil {
		return errwrap.Wrapf("Error preparing employee update query: {{err}}", err)
	}

	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, employee.Email, employee.Password, employee.AccessLevel, employee.EmployeeID)
	return err
}
