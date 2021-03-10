package kea

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
)

type LeaseFile struct {
	file   *os.File
	reader *csv.Reader
	writer *csv.Writer
}

func CreateLeaseFile(name string) (*LeaseFile, error) {
	f, err := os.Create(name)
	if err != nil {
		return nil, err
	}
	lf := &LeaseFile{
		file:   f,
		writer: csv.NewWriter(f),
	}
	header := []string{
		"address",
		"hwaddr",
		"client_id",
		"valid_lifetime",
		/*		"expire",
				"subnet_id",
				"fqdn_fwd",
				"fqdn_rev",
				"hostname",
				"state",
				"user_context", */
		"app_id",
	}
	err = lf.writer.Write(header)
	if err != nil {
		f.Close()
		return nil, err
	}
	lf.writer.Flush()
	err = lf.writer.Error()
	if err != nil {
		f.Close()
		return nil, err
	}
	return lf, nil
}

func OpenLeaseFile(name string) (*LeaseFile, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	lf := &LeaseFile{
		file:   f,
		reader: csv.NewReader(f),
	}
	_, err = lf.reader.Read()
	if err != nil {
		return nil, err
	}
	return lf, nil
}

func (lf *LeaseFile) Close() {
	lf.file.Close()
}

func (lf *LeaseFile) CopyToDatabase(db *dbops.PgDB, appID uint64) error {
	path, err := filepath.Abs(lf.file.Name())
	if err != nil {
		return err
	}
	_, err = db.Exec(fmt.Sprintf("DELETE FROM lease_update WHERE app_id=%d", appID))
	if err != nil {
		return err
	}

	_, err = db.Exec(fmt.Sprintf("COPY lease_update(address, hw_address, client_id, valid_lifetime, app_id) FROM '%s' WITH CSV HEADER", path))
	if err != nil {
		return err
	}
	_, err = db.Exec(fmt.Sprintf(`
        INSERT INTO lease (address, hw_address, client_id, valid_lifetime, app_id) SELECT * FROM lease_update WHERE app_id = %d ON CONFLICT(address, app_id) DO UPDATE SET valid_lifetime = EXCLUDED.valid_lifetime;
    `, appID))
	if err != nil {
		return err
	}

	return nil
}

func (lf *LeaseFile) Read() (*dbmodel.Lease, error) {
	record, err := lf.reader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, nil
		}
		return nil, err
	}
	validLifetime, err := strconv.ParseUint(record[3], 10, 32)
	if err != nil {
		return nil, err
	}
	appID, err := strconv.ParseUint(record[4], 10, 64)
	if err != nil {
		return nil, err
	}
	lease := &dbmodel.Lease{
		Address:       record[0],
		HWAddress:     nil,
		ClientID:      nil,
		ValidLifetime: uint32(validLifetime),
		AppID:         appID,
	}
	return lease, nil
}

func (lf *LeaseFile) Write(lease *dbmodel.Lease) error {
	record := []string{
		lease.Address,
		"",
		"",
		strconv.FormatUint(uint64(lease.ValidLifetime), 10),
		strconv.FormatUint(lease.AppID, 10),
	}
	err := lf.writer.Write(record)
	if err != nil {
		return err
	}
	return nil
}

func (lf *LeaseFile) Flush() error {
	lf.writer.Flush()
	err := lf.writer.Error()
	if err != nil {
		return err
	}
	return nil
}
