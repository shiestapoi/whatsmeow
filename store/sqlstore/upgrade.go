// Copyright (c) 2021 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package sqlstore

import (
	"database/sql"
	"fmt"
	"strings"
)

type upgradeFunc func(*sql.Tx, *Container) error

// Upgrades is a list of functions that will upgrade a database to the latest version.
//
// This may be of use if you want to manage the database fully manually, but in most cases you
// should just call Container.Upgrade to let the library handle everything.
var Upgrades = [...]upgradeFunc{upgradeV1, upgradeV2, upgradeV3, upgradeV4, upgradeV5, upgradeV6, upgradeV7}

func (c *Container) getVersion() (int, error) {
	_, err := c.db.Exec("CREATE TABLE IF NOT EXISTS whatsmeow_version (version INTEGER)")
	if err != nil {
		return -1, err
	}

	version := 0
	row := c.db.QueryRow("SELECT version FROM whatsmeow_version LIMIT 1")
	if row != nil {
		_ = row.Scan(&version)
	}
	return version, nil
}

func (c *Container) setVersion(tx *sql.Tx, version int) error {
	_, err := tx.Exec("DELETE FROM whatsmeow_version")
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT INTO whatsmeow_version (version) VALUES (?)", version)
	return err
}

// Upgrade upgrades the database from the current to the latest version available.
func (c *Container) Upgrade() error {
	if c.dialect == "sqlite" {
		var foreignKeysEnabled bool
		err := c.db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeysEnabled)
		if err != nil {
			return fmt.Errorf("failed to check if foreign keys are enabled: %w", err)
		} else if !foreignKeysEnabled {
			return fmt.Errorf("foreign keys are not enabled")
		}
	} else if c.dialect == "mysql" {
		var foreignKeysEnabled int
		err := c.db.QueryRow("SELECT @@foreign_key_checks").Scan(&foreignKeysEnabled)
		if err != nil {
			return fmt.Errorf("failed to check if foreign keys are enabled: %w", err)
		} else if foreignKeysEnabled != 1 {
			return fmt.Errorf("foreign keys are not enabled")
		}
	}

	version, err := c.getVersion()
	if err != nil {
		return err
	}

	for ; version < len(Upgrades); version++ {
		var tx *sql.Tx
		tx, err = c.db.Begin()
		if err != nil {
			return err
		}

		migrateFunc := Upgrades[version]
		c.log.Infof("Upgrading database to v%d", version+1)
		err = migrateFunc(tx, c)
		if err != nil {
			_ = tx.Rollback()
			return err
		}

		if err = c.setVersion(tx, version+1); err != nil {
			return err
		}

		if err = tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}

func upgradeV1(tx *sql.Tx, c *Container) error {
	if c.dialect == "mysql" {
		return upgradeV1MySQL(tx, c)
	}

	// PostgreSQL and SQLite version
	_, err := tx.Exec(`CREATE TABLE whatsmeow_device (
		jid TEXT PRIMARY KEY,

		registration_id BIGINT NOT NULL CHECK ( registration_id >= 0 AND registration_id < 4294967296 ),

		noise_key    bytea NOT NULL CHECK ( length(noise_key) = 32 ),
		identity_key bytea NOT NULL CHECK ( length(identity_key) = 32 ),

		signed_pre_key     bytea   NOT NULL CHECK ( length(signed_pre_key) = 32 ),
		signed_pre_key_id  INTEGER NOT NULL CHECK ( signed_pre_key_id >= 0 AND signed_pre_key_id < 16777216 ),
		signed_pre_key_sig bytea   NOT NULL CHECK ( length(signed_pre_key_sig) = 64 ),

		adv_key         bytea NOT NULL,
		adv_details     bytea NOT NULL,
		adv_account_sig bytea NOT NULL CHECK ( length(adv_account_sig) = 64 ),
		adv_device_sig  bytea NOT NULL CHECK ( length(adv_device_sig) = 64 ),

		platform      TEXT NOT NULL DEFAULT '',
		business_name TEXT NOT NULL DEFAULT '',
		push_name     TEXT NOT NULL DEFAULT ''
	)`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE TABLE whatsmeow_identity_keys (
		our_jid  TEXT,
		their_id TEXT,
		identity bytea NOT NULL CHECK ( length(identity) = 32 ),

		PRIMARY KEY (our_jid, their_id),
		FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
	)`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE TABLE whatsmeow_pre_keys (
		jid      TEXT,
		key_id   INTEGER          CHECK ( key_id >= 0 AND key_id < 16777216 ),
		key      bytea   NOT NULL CHECK ( length(key) = 32 ),
		uploaded BOOLEAN NOT NULL,

		PRIMARY KEY (jid, key_id),
		FOREIGN KEY (jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
	)`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE TABLE whatsmeow_sessions (
		our_jid  TEXT,
		their_id TEXT,
		session  bytea,

		PRIMARY KEY (our_jid, their_id),
		FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
	)`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE TABLE whatsmeow_sender_keys (
		our_jid    TEXT,
		chat_id    TEXT,
		sender_id  TEXT,
		sender_key bytea NOT NULL,

		PRIMARY KEY (our_jid, chat_id, sender_id),
		FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
	)`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE TABLE whatsmeow_app_state_sync_keys (
		jid         TEXT,
		key_id      bytea,
		key_data    bytea  NOT NULL,
		timestamp   BIGINT NOT NULL,
		fingerprint bytea  NOT NULL,

		PRIMARY KEY (jid, key_id),
		FOREIGN KEY (jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
	)`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE TABLE whatsmeow_app_state_version (
		jid     TEXT,
		name    TEXT,
		version BIGINT NOT NULL,
		hash    bytea  NOT NULL CHECK ( length(hash) = 128 ),

		PRIMARY KEY (jid, name),
		FOREIGN KEY (jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
	)`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE TABLE whatsmeow_app_state_mutation_macs (
		jid       TEXT,
		name      TEXT,
		version   BIGINT,
		index_mac bytea          CHECK ( length(index_mac) = 32 ),
		value_mac bytea NOT NULL CHECK ( length(value_mac) = 32 ),

		PRIMARY KEY (jid, name, version, index_mac),
		FOREIGN KEY (jid, name) REFERENCES whatsmeow_app_state_version(jid, name) ON DELETE CASCADE ON UPDATE CASCADE
	)`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE TABLE whatsmeow_contacts (
		our_jid       TEXT,
		their_jid     TEXT,
		first_name    TEXT,
		full_name     TEXT,
		push_name     TEXT,
		business_name TEXT,

		PRIMARY KEY (our_jid, their_jid),
		FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
	)`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`CREATE TABLE whatsmeow_chat_settings (
		our_jid       TEXT,
		chat_jid      TEXT,
		muted_until   BIGINT  NOT NULL DEFAULT 0,
		pinned        BOOLEAN NOT NULL DEFAULT false,
		archived      BOOLEAN NOT NULL DEFAULT false,

		PRIMARY KEY (our_jid, chat_jid),
		FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
	)`)
	if err != nil {
		return err
	}
	return nil
}

func upgradeV1MySQL(tx *sql.Tx, c *Container) error {
	// First, detect if we're using MariaDB
	var version string
	c.db.QueryRow("SELECT VERSION()").Scan(&version)

	// Create device table - basic table without constraints
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS whatsmeow_device (
        jid VARCHAR(255) PRIMARY KEY,
        registration_id BIGINT NOT NULL,
        noise_key    LONGBLOB NOT NULL,
        identity_key LONGBLOB NOT NULL,
        signed_pre_key     LONGBLOB NOT NULL,
        signed_pre_key_id  INTEGER NOT NULL,
        signed_pre_key_sig LONGBLOB NOT NULL,
        adv_key         LONGBLOB NOT NULL,
        adv_details     LONGBLOB NOT NULL,
        adv_account_sig LONGBLOB NOT NULL,
        adv_device_sig  LONGBLOB NOT NULL,
        platform      TEXT NOT NULL,
        business_name TEXT NOT NULL,
        push_name     TEXT NOT NULL
    )`)
	if err != nil {
		return fmt.Errorf("failed to create whatsmeow_device table: %w", err)
	}

	// Create identity keys table
	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS whatsmeow_identity_keys (
        our_jid  VARCHAR(255),
        their_id VARCHAR(255),
        identity LONGBLOB NOT NULL,
        PRIMARY KEY (our_jid, their_id),
        FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
    )`)
	if err != nil {
		return fmt.Errorf("failed to create whatsmeow_identity_keys table: %w", err)
	}

	// Create pre keys table - use backticks for reserved keyword 'key' in MariaDB
	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS whatsmeow_pre_keys (
        jid VARCHAR(255),
        key_id INTEGER,
        ` + "`key`" + ` LONGBLOB NOT NULL,
        uploaded TINYINT(1) NOT NULL,
        PRIMARY KEY (jid, key_id),
        FOREIGN KEY (jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
    )`)
	if err != nil {
		return fmt.Errorf("failed to create whatsmeow_pre_keys table: %w", err)
	}

	// Create sessions table
	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS whatsmeow_sessions (
        our_jid VARCHAR(255),
        their_id VARCHAR(255),
        session LONGBLOB,
        PRIMARY KEY (our_jid, their_id),
        FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
    )`)
	if err != nil {
		return fmt.Errorf("failed to create whatsmeow_sessions table: %w", err)
	}

	// Create sender keys table
	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS whatsmeow_sender_keys (
        our_jid VARCHAR(255),
        chat_id VARCHAR(255),
        sender_id VARCHAR(255),
        sender_key LONGBLOB NOT NULL,
        PRIMARY KEY (our_jid, chat_id, sender_id),
        FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
    )`)
	if err != nil {
		return fmt.Errorf("failed to create whatsmeow_sender_keys table: %w", err)
	}

	// Create app state sync keys table
	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS whatsmeow_app_state_sync_keys (
        jid VARCHAR(255),
        key_id LONGBLOB,
        key_data LONGBLOB NOT NULL,
        timestamp BIGINT NOT NULL,
        fingerprint LONGBLOB NOT NULL,
        PRIMARY KEY (jid, key_id(255)),
        FOREIGN KEY (jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
    )`)
	if err != nil {
		return fmt.Errorf("failed to create whatsmeow_app_state_sync_keys table: %w", err)
	}

	// Create app state version table
	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS whatsmeow_app_state_version (
        jid VARCHAR(255),
        name VARCHAR(255),
        version BIGINT NOT NULL,
        hash LONGBLOB NOT NULL,
        PRIMARY KEY (jid, name),
        FOREIGN KEY (jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
    )`)
	if err != nil {
		return fmt.Errorf("failed to create whatsmeow_app_state_version table: %w", err)
	}

	// Create app state mutation macs table
	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS whatsmeow_app_state_mutation_macs (
        jid VARCHAR(255),
        name VARCHAR(255),
        version BIGINT,
        index_mac LONGBLOB,
        value_mac LONGBLOB NOT NULL,
        PRIMARY KEY (jid, name, version, index_mac(32)),
        FOREIGN KEY (jid, name) REFERENCES whatsmeow_app_state_version(jid, name) ON DELETE CASCADE ON UPDATE CASCADE
    )`)
	if err != nil {
		return fmt.Errorf("failed to create whatsmeow_app_state_mutation_macs table: %w", err)
	}

	// Create contacts table
	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS whatsmeow_contacts (
        our_jid VARCHAR(255),
        their_jid VARCHAR(255),
        first_name TEXT,
        full_name TEXT,
        push_name TEXT,
        business_name TEXT,
        PRIMARY KEY (our_jid, their_jid),
        FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
    )`)
	if err != nil {
		return fmt.Errorf("failed to create whatsmeow_contacts table: %w", err)
	}

	// Create chat settings table
	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS whatsmeow_chat_settings (
        our_jid VARCHAR(255),
        chat_jid VARCHAR(255),
        muted_until BIGINT NOT NULL DEFAULT 0,
        pinned TINYINT(1) NOT NULL DEFAULT 0,
        archived TINYINT(1) NOT NULL DEFAULT 0,
        PRIMARY KEY (our_jid, chat_jid),
        FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
    )`)
	if err != nil {
		return fmt.Errorf("failed to create whatsmeow_chat_settings table: %w", err)
	}

	return nil
}

const fillSigKeyPostgres = `
UPDATE whatsmeow_device SET adv_account_sig_key=(
	SELECT identity
	FROM whatsmeow_identity_keys
	WHERE our_jid=whatsmeow_device.jid
	  AND their_id=concat(split_part(whatsmeow_device.jid, '.', 1), ':0')
);
DELETE FROM whatsmeow_device WHERE adv_account_sig_key IS NULL;
ALTER TABLE whatsmeow_device ALTER COLUMN adv_account_sig_key SET NOT NULL;
`

const fillSigKeySQLite = `
UPDATE whatsmeow_device SET adv_account_sig_key=(
	SELECT identity
	FROM whatsmeow_identity_keys
	WHERE our_jid=whatsmeow_device.jid
	  AND their_id=substr(whatsmeow_device.jid, 0, instr(whatsmeow_device.jid, '.')) || ':0'
)
`

const fillSigKeyMySQL = `
UPDATE whatsmeow_device SET adv_account_sig_key=(
	SELECT identity
	FROM whatsmeow_identity_keys
	WHERE our_jid=whatsmeow_device.jid
	  AND their_id=CONCAT(SUBSTRING_INDEX(whatsmeow_device.jid, '.', 1), ':0')
)
`

func upgradeV2(tx *sql.Tx, container *Container) error {
	var err error
	if container.dialect == "mysql" {
		_, err = tx.Exec("ALTER TABLE whatsmeow_device ADD COLUMN adv_account_sig_key BLOB")
	} else {
		_, err = tx.Exec("ALTER TABLE whatsmeow_device ADD COLUMN adv_account_sig_key bytea")
	}

	if err != nil {
		if container.dialect == "mysql" && strings.Contains(err.Error(), "Duplicate column name") {
			// Column might already exist, continue
		} else {
			return err
		}
	}

	if strings.Contains(container.dialect, "postgres") || container.dialect == "pgx" {
		_, err = tx.Exec(fillSigKeyPostgres)
	} else if container.dialect == "mysql" {
		// For MySQL/MariaDB, execute statements separately
		_, err = tx.Exec(fillSigKeyMySQL)
		if err != nil {
			return err
		}
		_, err = tx.Exec("DELETE FROM whatsmeow_device WHERE adv_account_sig_key IS NULL")
		if err != nil {
			return err
		}
		_, err = tx.Exec("ALTER TABLE whatsmeow_device MODIFY adv_account_sig_key BLOB NOT NULL")
	} else {
		_, err = tx.Exec(fillSigKeySQLite)
	}
	return err
}

func upgradeV3(tx *sql.Tx, container *Container) error {
	if container.dialect == "mysql" {
		_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS whatsmeow_message_secrets (
            our_jid VARCHAR(255),
            chat_jid VARCHAR(255),
            sender_jid VARCHAR(255),
            message_id VARCHAR(255),
            ` + "`key`" + ` LONGBLOB NOT NULL,
            PRIMARY KEY (our_jid, chat_jid, sender_jid, message_id),
            FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
        )`)
		return err
	} else {
		blobType := "bytea"
		_, err := tx.Exec(`CREATE TABLE whatsmeow_message_secrets (
            our_jid TEXT,
            chat_jid TEXT,
            sender_jid TEXT,
            message_id TEXT,
            key ` + blobType + ` NOT NULL,
            PRIMARY KEY (our_jid, chat_jid, sender_jid, message_id),
            FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
        )`)
		return err
	}
}

func upgradeV4(tx *sql.Tx, container *Container) error {
	if container.dialect == "mysql" {
		_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS whatsmeow_privacy_tokens (
            our_jid VARCHAR(255),
            their_jid VARCHAR(255),
            token LONGBLOB NOT NULL,
            timestamp BIGINT NOT NULL,
            PRIMARY KEY (our_jid, their_jid)
        )`)
		return err
	} else {
		blobType := "bytea"
		_, err := tx.Exec(`CREATE TABLE whatsmeow_privacy_tokens (
            our_jid TEXT,
            their_jid TEXT,
            token ` + blobType + ` NOT NULL,
            timestamp BIGINT NOT NULL,
            PRIMARY KEY (our_jid, their_jid)
        )`)
		return err
	}
}

func upgradeV5(tx *sql.Tx, container *Container) error {
	_, err := tx.Exec("UPDATE whatsmeow_device SET jid=REPLACE(jid, '.0', '')")
	return err
}

func upgradeV6(tx *sql.Tx, container *Container) error {
	var err error
	if container.dialect == "mysql" {
		// MySQL doesn't have a UUID type, use VARCHAR(36)
		_, err = tx.Exec("ALTER TABLE whatsmeow_device ADD COLUMN facebook_uuid VARCHAR(36)")
	} else {
		_, err = tx.Exec("ALTER TABLE whatsmeow_device ADD COLUMN facebook_uuid uuid")
	}
	return err
}

func upgradeV7(tx *sql.Tx, container *Container) error {
	// This upgrade is only needed for MySQL/MariaDB to support emoji characters
	if container.dialect == "mysql" {
			// Use standard MySQL syntax that works in all versions
			// First disable foreign key checks to avoid constraint issues
			_, err := tx.Exec("SET FOREIGN_KEY_CHECKS = 0")
			if err != nil {
				return fmt.Errorf("failed to disable foreign key checks: %w", err)
			}

			// Set database default character set
			_, err = tx.Exec("SET NAMES utf8mb4")
			if err != nil {
				// Re-enable foreign key checks before returning
				_, _ = tx.Exec("SET FOREIGN_KEY_CHECKS = 1")
				return fmt.Errorf("failed to set default character set: %w", err)
			}

			// List tables to update
			tables := []string{
				"whatsmeow_contacts",
				"whatsmeow_device",
				"whatsmeow_chat_settings",
				"whatsmeow_message_secrets",
				"whatsmeow_app_state_sync_keys",
				"whatsmeow_app_state_version",
				"whatsmeow_identity_keys",
				"whatsmeow_pre_keys",
				"whatsmeow_sessions",
				"whatsmeow_sender_keys",
				"whatsmeow_privacy_tokens",
			}

			// For each table, modify character set using standard ALTER TABLE syntax
			for _, table := range tables {
				// Use the more compatible ALTER TABLE syntax
				_, err := tx.Exec("ALTER TABLE " + table + " CHARACTER SET = utf8mb4 COLLATE utf8mb4_unicode_ci")
				if err != nil {
					// Re-enable foreign key checks before returning error
					_, _ = tx.Exec("SET FOREIGN_KEY_CHECKS = 1")
					return fmt.Errorf("failed to convert %s table to utf8mb4: %w", table, err)
				}
			}

			// Define text columns that need explicit conversion - use safe ALTER TABLE statements
			textColumns := map[string][]string{
				"whatsmeow_contacts": {
					"first_name", "full_name", "push_name", "business_name",
				},
				"whatsmeow_device": {
					"platform", "business_name", "push_name",
				},
			}

			// Update text columns to utf8mb4
			for table, columns := range textColumns {
				for _, column := range columns {
					// Execute separate statements for each column to ensure compatibility
					_, err := tx.Exec(fmt.Sprintf("ALTER TABLE %s MODIFY %s TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
						table, column))
					if err != nil {
						// Re-enable foreign key checks before returning
						_, _ = tx.Exec("SET FOREIGN_KEY_CHECKS = 1")
						return fmt.Errorf("failed to modify %s.%s column to utf8mb4: %w", table, column, err)
					}
				}
			}

			// Re-enable foreign key checks
			_, err = tx.Exec("SET FOREIGN_KEY_CHECKS = 1")
			if err != nil {
				return fmt.Errorf("failed to re-enable foreign key checks: %w", err)
			}
		}
		return nil
	}
