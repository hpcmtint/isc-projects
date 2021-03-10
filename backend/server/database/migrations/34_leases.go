package dbmigs

import (
	"github.com/go-pg/migrations/v7"
)

func init() {
	migrations.MustRegisterTx(func(db migrations.DB) error {
		_, err := db.Exec(`
            CREATE TABLE lease_update (
                address TEXT NOT NULL,
                hw_address BYTEA,
                client_id BYTEA,
                valid_lifetime INTEGER,
                app_id BIGINT
            );

            CREATE INDEX lease_update_app_id ON lease_update (app_id);

            CREATE TABLE lease (
                id SERIAL PRIMARY KEY,
                address TEXT NOT NULL,
                hw_address BYTEA,
                client_id BYTEA,
                valid_lifetime INTEGER,
                app_id BIGINT
            );

            CREATE UNIQUE INDEX lease_ip_address_unique ON lease (address, app_id);
        `)
		return err
	}, func(db migrations.DB) error {
		_, err := db.Exec(`
             DROP TABLE lease;
        `)
		return err
	})
}
