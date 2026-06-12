package sql

import (
	"fmt"

	"github.com/go-gorp/gorp/v3"
)

type migration_2_18_25 struct {
	db *SqlDb
}

func (m migration_2_18_25) PreApply(tx *gorp.Transaction) error {
	switch m.db.Sql().Dialect.(type) {
	case gorp.SqliteDialect:
		return m.sqliteDropStartTemplateColumns(tx)
	case gorp.MySQLDialect:
		m.dropMySQLFK(tx, "project__device_settings", "start_template_id")
		return nil
	case gorp.PostgresDialect:
		_, _ = tx.Exec(m.db.PrepareQuery(
			`alter table "project__device_settings" drop constraint if exists "project__device_settings_start_template_id_fkey"`))
		return nil
	default:
		return nil
	}
}

func (m migration_2_18_25) dropMySQLFK(tx *gorp.Transaction, table, column string) {
	fkName, err := tx.SelectStr(
		`SELECT CONSTRAINT_NAME FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
         WHERE TABLE_SCHEMA = DATABASE()
           AND TABLE_NAME = ?
           AND COLUMN_NAME = ?
           AND REFERENCED_TABLE_NAME IS NOT NULL
         LIMIT 1`, table, column)
	if err == nil && fkName != "" {
		_, _ = tx.Exec(fmt.Sprintf("alter table `%s` drop foreign key `%s`", table, fkName))
	}
}

// SQLite cannot DROP COLUMN when a FK references it; rebuild project__device_settings.
func (m migration_2_18_25) sqliteDropStartTemplateColumns(tx *gorp.Transaction) error {
	hasCol, err := m.sqliteTableHasColumn(tx, "project__device_settings", "start_template_id")
	if err != nil {
		return err
	}
	if hasCol {
		if _, err = tx.Exec(`PRAGMA foreign_keys=OFF`); err != nil {
			return err
		}
		_, err = tx.Exec(`
CREATE TABLE ` + "`project__device_settings_new`" + ` (
  ` + "`project_id`" + `                   integer primary key,
  ` + "`discover_template_id`" + `         integer null,
  ` + "`stop_template_id`" + `             integer null,
  ` + "`restart_template_id`" + `          integer null,
  ` + "`status_template_id`" + `           integer null,
  ` + "`config_template_id`" + `           integer null,
  ` + "`status_refresh_interval_min`" + `  integer not null default 0,
  ` + "`last_status_refresh_at`" + `       datetime null,
  ` + "`default_inventory_id`" + `         integer null,
  ` + "`default_ansible_user`" + `         varchar(255) not null default '',
  ` + "`default_ansible_password`" + `     text not null default '',
  ` + "`default_ansible_connection`" + `   varchar(64) not null default 'winrm',
  ` + "`default_ansible_winrm_transport`" + ` varchar(64) not null default 'basic',
  ` + "`default_ansible_winrm_scheme`" + ` varchar(64) not null default 'http',
  ` + "`default_ansible_port`" + `         integer not null default 5985,
  ` + "`default_ansible_winrm_server_cert_validation`" + ` varchar(64) not null default 'ignore',
  ` + "`default_config_json`" + `          text not null default '',
  foreign key (` + "`project_id`" + `) references ` + "`project`" + `(` + "`id`" + `) on delete cascade,
  foreign key (` + "`discover_template_id`" + `) references ` + "`project__template`" + `(` + "`id`" + `) on delete set null,
  foreign key (` + "`stop_template_id`" + `) references ` + "`project__template`" + `(` + "`id`" + `) on delete set null,
  foreign key (` + "`restart_template_id`" + `) references ` + "`project__template`" + `(` + "`id`" + `) on delete set null,
  foreign key (` + "`status_template_id`" + `) references ` + "`project__template`" + `(` + "`id`" + `) on delete set null,
  foreign key (` + "`config_template_id`" + `) references ` + "`project__template`" + `(` + "`id`" + `) on delete set null
)`)
		if err != nil {
			return err
		}
		_, err = tx.Exec(`
INSERT INTO ` + "`project__device_settings_new`" + ` (
  ` + "`project_id`" + `, ` + "`discover_template_id`" + `, ` + "`stop_template_id`" + `, ` + "`restart_template_id`" + `,
  ` + "`status_template_id`" + `, ` + "`config_template_id`" + `, ` + "`status_refresh_interval_min`" + `,
  ` + "`last_status_refresh_at`" + `, ` + "`default_inventory_id`" + `, ` + "`default_ansible_user`" + `,
  ` + "`default_ansible_password`" + `, ` + "`default_ansible_connection`" + `,
  ` + "`default_ansible_winrm_transport`" + `, ` + "`default_ansible_winrm_scheme`" + `,
  ` + "`default_ansible_port`" + `, ` + "`default_ansible_winrm_server_cert_validation`" + `,
  ` + "`default_config_json`" + `
)
SELECT
  ` + "`project_id`" + `, ` + "`discover_template_id`" + `, ` + "`stop_template_id`" + `, ` + "`restart_template_id`" + `,
  ` + "`status_template_id`" + `, ` + "`config_template_id`" + `, ` + "`status_refresh_interval_min`" + `,
  ` + "`last_status_refresh_at`" + `, ` + "`default_inventory_id`" + `, ` + "`default_ansible_user`" + `,
  ` + "`default_ansible_password`" + `, ` + "`default_ansible_connection`" + `,
  ` + "`default_ansible_winrm_transport`" + `, ` + "`default_ansible_winrm_scheme`" + `,
  ` + "`default_ansible_port`" + `, ` + "`default_ansible_winrm_server_cert_validation`" + `,
  ` + "`default_config_json`" + `
FROM ` + "`project__device_settings`" + ``)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(`DROP TABLE ` + "`project__device_settings`"); err != nil {
			return err
		}
		if _, err = tx.Exec(`ALTER TABLE ` + "`project__device_settings_new`" + ` RENAME TO ` + "`project__device_settings`"); err != nil {
			return err
		}
		if _, err = tx.Exec(`PRAGMA foreign_keys=ON`); err != nil {
			return err
		}
	}
	return nil
}

func (m migration_2_18_25) sqliteTableHasColumn(tx *gorp.Transaction, table, column string) (bool, error) {
	rows, err := tx.Query(fmt.Sprintf("PRAGMA table_info(`%s`)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dfltValue any
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}
