package asset

import (
	"context"
	"database/sql"

	"github.com/hashicorp/errwrap"
	"github.com/henvic/embroidery/server"
	"github.com/kisielk/sqlstruct"
	uuid "github.com/satori/go.uuid"
)

var db = server.Instance.DB

// Asset of a client or store
type Asset struct {
	AssetID          string `schema:"asset_id"`
	ClientID         string `schema:"client_id"`
	Filepath         string `schema:"filepath"`
	Status           string `schema:"status"`
	OriginalFilepath string `schema:"original_filepath"`
	ReceivedDate     string `schema:"received_date"`
}

// ListFilter sets the filter settings
type ListFilter struct {
	ClientID     string
	ShowArchived bool
}

// List assets
func List(ctx context.Context, f ListFilter) (assets []Asset, err error) {
	var q = "SELECT asset_id,client_id,filepath,status,original_filepath,received_date FROM asset"

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

	q += " ORDER BY received_date DESC"

	stmt, err := db().PrepareContext(ctx, q)

	if err != nil {
		return nil, errwrap.Wrapf("Error preparing assets query: {{err}}", err)
	}

	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, i...)

	if err != nil {
		return nil, errwrap.Wrapf("Error querying assets: {{err}}", err)
	}

	for rows.Next() {
		var a Asset
		err = sqlstruct.Scan(&a, rows)

		if err != nil {
			errwrap.Wrapf("Error scanning assets rows: {{err}}", err)
			break
		}

		assets = append(assets, a)
	}

	return assets, err
}

// Insert asset on database
func Insert(ctx context.Context, asset Asset) (uid string, err error) {
	var query = `INSERT INTO asset (
		asset_id,
		client_id,
		filepath,
		status,
		original_filepath
		)
		VALUES (?, ?, ?, ?, ?)`

	stmt, err := db().PrepareContext(ctx, query)

	if err != nil {
		return "", err
	}

	defer stmt.Close()

	id := uuid.NewV4().String()
	_, err = stmt.ExecContext(
		ctx,
		id,
		asset.ClientID,
		asset.Filepath,
		asset.Status,
		asset.OriginalFilepath,
	)

	if err != nil {
		return "", err
	}

	return id, err
}

// Update asset
func Update(ctx context.Context, asset Asset) error {
	stmt, err := db().PrepareContext(ctx,
		`UPDATE asset SET filepath = ?, status = ? WHERE asset_id = ?`)

	if err != nil {
		return errwrap.Wrapf("Error preparing employee update query: {{err}}", err)
	}

	defer stmt.Close()

	_, err = stmt.ExecContext(ctx,
		asset.Filepath,
		asset.Status,
		asset.AssetID,
	)

	return err
}

// Get asset by ID
func Get(ctx context.Context, clientID, assetID string) (Asset, error) {
	stmt, err := db().PrepareContext(ctx,
		`SELECT asset_id,client_id,filepath,status,original_filepath,received_date
FROM asset WHERE client_id = ? AND asset_id = ?`)

	if err != nil {
		return Asset{}, errwrap.Wrapf("Error preparing asset query: {{err}}", err)
	}

	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, clientID, assetID)

	if err != nil {
		return Asset{}, errwrap.Wrapf("Error querying asset: {{err}}", err)
	}

	var asset Asset

	if ok := rows.Next(); !ok {
		return asset, sql.ErrNoRows
	}

	if err := sqlstruct.Scan(&asset, rows); err != nil {
		return asset, errwrap.Wrapf("Error scanning asset rows: {{err}}", err)
	}

	return asset, nil
}
