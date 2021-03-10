package dbmodel

import (
	pkgerrors "github.com/pkg/errors"

	dbops "isc.org/stork/server/database"
)

type Lease struct {
	ID            int64
	Address       string
	HWAddress     []byte
	ClientID      []byte
	ValidLifetime uint32
	/*	Cltt          uint64
		SubnetID      uint32
		FqdnFwd       bool
		FqdnRev       bool
		Hostname      string
		State         int
		UserContext   string */
	AppID uint64
}

func AddLease(dbIface interface{}, lease *Lease) error {
	tx, rollback, commit, err := dbops.Transaction(dbIface)
	if err != nil {
		err = pkgerrors.WithMessagef(err, "problem with starting transaction for adding new lease %s",
			lease.Address)
		return err
	}
	defer rollback()

	_, err = tx.Model(lease).Insert()
	if err != nil {
		err = pkgerrors.WithMessagef(err, "problem with adding lease %s", lease.Address)
		return err
	}

	err = commit()
	if err != nil {
		err = pkgerrors.WithMessagef(err, "problem with committing new lease %s into the database",
			lease.Address)
	}

	return err
}
