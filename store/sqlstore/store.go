// Copyright (c) 2022 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package sqlstore contains an SQL-backed implementation of the interfaces in the store package.
package sqlstore

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/shiestapoi/whatsmeow/store"
	"github.com/shiestapoi/whatsmeow/types"
	"github.com/shiestapoi/whatsmeow/util/keys"
)

// ErrInvalidLength is returned by some database getters if the database returned a byte array with an unexpected length.
// This should be impossible, as the database schema contains CHECK()s for all the relevant columns.
var ErrInvalidLength = errors.New("database returned byte array with illegal length")

// PostgresArrayWrapper is a function to wrap array values before passing them to the sql package.
//
// When using github.com/lib/pq, you should set
//
//	whatsmeow.PostgresArrayWrapper = pq.Array
var PostgresArrayWrapper func(interface{}) interface {
	driver.Valuer
	sql.Scanner
}

type SQLStore struct {
	*Container
	JID string

	preKeyLock sync.Mutex

	contactCache     map[types.JID]*types.ContactInfo
	contactCacheLock sync.Mutex
}

// NewSQLStore creates a new SQLStore with the given database container and user JID.
// It contains implementations of all the different stores in the store package.
//
// In general, you should use Container.NewDevice or Container.GetDevice instead of this.
func NewSQLStore(c *Container, jid types.JID) *SQLStore {
	return &SQLStore{
		Container:    c,
		JID:          jid.String(),
		contactCache: make(map[types.JID]*types.ContactInfo),
	}
}

var _ store.AllStores = (*SQLStore)(nil)

const (
	putIdentityQuery = `
		INSERT INTO whatsmeow_identity_keys (our_jid, their_id, identity) VALUES ($1, $2, $3)
		ON CONFLICT (our_jid, their_id) DO UPDATE SET identity=excluded.identity
	`
	deleteAllIdentitiesQuery = `DELETE FROM whatsmeow_identity_keys WHERE our_jid=$1 AND their_id LIKE $2`
	deleteIdentityQuery      = `DELETE FROM whatsmeow_identity_keys WHERE our_jid=$1 AND their_id=$2`
	getIdentityQuery         = `SELECT identity FROM whatsmeow_identity_keys WHERE our_jid=$1 AND their_id=$2`
	insertPreKeyQuery        = `INSERT INTO whatsmeow_pre_keys (jid, key_id, ` + "`key`" + `, uploaded) VALUES ($1, $2, $3, $4)`
)

func (s *SQLStore) dialectQuery(query string) string {
	if s.dialect == "mysql" || s.dialect == "sqlite3" {
		// Replace $N with ? for MySQL and SQLite
		result := query
		for i := 1; i <= 20; i++ {
			result = strings.ReplaceAll(result, fmt.Sprintf("$%d", i), "?")
		}

		// Replace key with `key` for MySQL
		if s.dialect == "mysql" {
			// Handle the field name correctly by escaping with backticks
			result = strings.ReplaceAll(result, "key FROM", "`key` FROM")
			result = strings.ReplaceAll(result, "key_id, key FROM", "key_id, `key` FROM")
			result = strings.ReplaceAll(result, "(jid, key_id, key, uploaded)", "(jid, key_id, `key`, uploaded)")
			result = strings.ReplaceAll(result, "message_id, key)", "message_id, `key`)")

			// Fix WHERE clause issues in ON DUPLICATE KEY statements
			result = strings.Replace(result,
				"WHERE VALUES(timestamp) > whatsmeow_app_state_sync_keys.timestamp",
				"", -1)

			result = strings.Replace(result,
				"ON CONFLICT (our_jid, their_id) DO UPDATE SET identity=excluded.identity",
				"ON DUPLICATE KEY UPDATE identity=VALUES(identity)",
				-1)

			result = strings.Replace(result,
				"ON CONFLICT (our_jid, their_id) DO UPDATE SET session=excluded.session",
				"ON DUPLICATE KEY UPDATE session=VALUES(session)",
				-1)

			result = strings.Replace(result,
				"ON CONFLICT (our_jid, chat_jid) DO UPDATE SET",
				"ON DUPLICATE KEY UPDATE",
				-1)

			result = strings.Replace(result,
				"ON CONFLICT (our_jid, chat_id, sender_id) DO UPDATE SET sender_key=excluded.sender_key",
				"ON DUPLICATE KEY UPDATE sender_key=VALUES(sender_key)",
				-1)

			result = strings.Replace(result,
				"ON CONFLICT (jid, key_id) DO UPDATE",
				"ON DUPLICATE KEY UPDATE",
				-1)

			result = strings.Replace(result,
				"ON CONFLICT (jid, name) DO UPDATE SET version=excluded.version, hash=excluded.hash",
				"ON DUPLICATE KEY UPDATE version=VALUES(version), hash=VALUES(hash)",
				-1)

			result = strings.Replace(result,
				"ON CONFLICT (our_jid, their_jid) DO UPDATE SET first_name=excluded.first_name, full_name=excluded.full_name",
				"ON DUPLICATE KEY UPDATE first_name=VALUES(first_name), full_name=VALUES(full_name)",
				-1)

			result = strings.Replace(result,
				"ON CONFLICT (our_jid, their_jid) DO UPDATE SET push_name=excluded.push_name",
				"ON DUPLICATE KEY UPDATE push_name=VALUES(push_name)",
				-1)

			result = strings.Replace(result,
				"ON CONFLICT (our_jid, their_jid) DO UPDATE SET business_name=excluded.business_name",
				"ON DUPLICATE KEY UPDATE business_name=VALUES(business_name)",
				-1)

			result = strings.Replace(result,
				"ON CONFLICT (our_jid, chat_jid, sender_jid, message_id) DO NOTHING",
				"ON DUPLICATE KEY UPDATE our_jid=our_jid",
				-1)

			result = strings.Replace(result,
				"ON CONFLICT (jid, key_id) DO UPDATE SET key_data=excluded.key_data, timestamp=excluded.timestamp, fingerprint=excluded.fingerprint WHERE excluded.timestamp > whatsmeow_app_state_sync_keys.timestamp",
				"ON DUPLICATE KEY UPDATE key_data=VALUES(key_data), timestamp=VALUES(timestamp), fingerprint=VALUES(fingerprint) WHERE VALUES(timestamp) > whatsmeow_app_state_sync_keys.timestamp",
				-1)
		}

		return result
	}
	return query
}

func (s *SQLStore) PutIdentity(address string, key [32]byte) error {
	var query string
	if s.dialect == "mysql" {
		query = `
		INSERT INTO whatsmeow_identity_keys (our_jid, their_id, identity)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE identity=VALUES(identity)
		`
	} else if s.dialect == "sqlite3" {
		query = `
		INSERT INTO whatsmeow_identity_keys (our_jid, their_id, identity)
		VALUES (?, ?, ?)
		ON CONFLICT (our_jid, their_id) DO UPDATE SET identity=excluded.identity
		`
	} else {
		query = putIdentityQuery
	}

	_, err := s.db.Exec(query, s.JID, address, key[:])
	return err
}

func (s *SQLStore) DeleteAllIdentities(phone string) error {
	_, err := s.db.Exec(s.dialectQuery(deleteAllIdentitiesQuery), s.JID, phone+":%")
	return err
}

func (s *SQLStore) DeleteIdentity(address string) error {
	_, err := s.db.Exec(s.dialectQuery(deleteAllIdentitiesQuery), s.JID, address)
	return err
}

func (s *SQLStore) IsTrustedIdentity(address string, key [32]byte) (bool, error) {
	var existingIdentity []byte
	err := s.db.QueryRow(s.dialectQuery(getIdentityQuery), s.JID, address).Scan(&existingIdentity)
	if errors.Is(err, sql.ErrNoRows) {
		// Trust if not known, it'll be saved automatically later
		return true, nil
	} else if err != nil {
		return false, err
	} else if len(existingIdentity) != 32 {
		return false, ErrInvalidLength
	}
	return *(*[32]byte)(existingIdentity) == key, nil
}

const (
	getSessionQuery = `SELECT session FROM whatsmeow_sessions WHERE our_jid=$1 AND their_id=$2`
	hasSessionQuery = `SELECT true FROM whatsmeow_sessions WHERE our_jid=$1 AND their_id=$2`
	putSessionQuery = `
		INSERT INTO whatsmeow_sessions (our_jid, their_id, session) VALUES ($1, $2, $3)
		ON CONFLICT (our_jid, their_id) DO UPDATE SET session=excluded.session
	`
	deleteAllSessionsQuery = `DELETE FROM whatsmeow_sessions WHERE our_jid=$1 AND their_id LIKE $2`
	deleteSessionQuery     = `DELETE FROM whatsmeow_sessions WHERE our_jid=$1 AND their_id=$2`
)

func (s *SQLStore) GetSession(address string) (session []byte, err error) {
	err = s.db.QueryRow(s.dialectQuery(getSessionQuery), s.JID, address).Scan(&session)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return
}

func (s *SQLStore) HasSession(address string) (has bool, err error) {
	err = s.db.QueryRow(s.dialectQuery(hasSessionQuery), s.JID, address).Scan(&has)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return
}

func (s *SQLStore) PutSession(address string, session []byte) error {
	var query string
	if s.dialect == "mysql" {
		query = `
		INSERT INTO whatsmeow_sessions (our_jid, their_id, session)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE session=VALUES(session)
		`
	} else if s.dialect == "sqlite3" {
		query = `
		INSERT INTO whatsmeow_sessions (our_jid, their_id, session)
		VALUES (?, ?, ?)
		ON CONFLICT (our_jid, their_id) DO UPDATE SET session=excluded.session
		`
	} else {
		query = putSessionQuery
	}

	_, err := s.db.Exec(query, s.JID, address, session)
	return err
}

func (s *SQLStore) DeleteAllSessions(phone string) error {
	_, err := s.db.Exec(s.dialectQuery(deleteAllSessionsQuery), s.JID, phone+":%")
	return err
}

func (s *SQLStore) DeleteSession(address string) error {
	_, err := s.db.Exec(s.dialectQuery(deleteSessionQuery), s.JID, address)
	return err
}

const (
	getLastPreKeyIDQuery        = `SELECT MAX(key_id) FROM whatsmeow_pre_keys WHERE jid=$1`
	getUnuploadedPreKeysQuery   = `SELECT key_id, ` + "`key`" + ` FROM whatsmeow_pre_keys WHERE jid=$1 AND uploaded=false ORDER BY key_id LIMIT $2`
	getPreKeyQuery              = `SELECT key_id, ` + "`key`" + ` FROM whatsmeow_pre_keys WHERE jid=$1 AND key_id=$2`
	deletePreKeyQuery           = `DELETE FROM whatsmeow_pre_keys WHERE jid=$1 AND key_id=$2`
	markPreKeysAsUploadedQuery  = `UPDATE whatsmeow_pre_keys SET uploaded=true WHERE jid=$1 AND key_id<=$2`
	getUploadedPreKeyCountQuery = `SELECT COUNT(*) FROM whatsmeow_pre_keys WHERE jid=$1 AND uploaded=true`
)

func (s *SQLStore) genOnePreKey(id uint32, markUploaded bool) (*keys.PreKey, error) {
	key := keys.NewPreKey(id)

	var query string
	if s.dialect == "mysql" {
		query = "INSERT INTO whatsmeow_pre_keys (jid, key_id, `key`, uploaded) VALUES (?, ?, ?, ?)"
	} else {
		query = s.dialectQuery(insertPreKeyQuery)
	}

	_, err := s.db.Exec(query, s.JID, key.KeyID, key.Priv[:], markUploaded)
	return key, err
}

func (s *SQLStore) getNextPreKeyID() (uint32, error) {
	var lastKeyID sql.NullInt32
	err := s.db.QueryRow(s.dialectQuery(getLastPreKeyIDQuery), s.JID).Scan(&lastKeyID)
	if err != nil {
		return 0, fmt.Errorf("failed to query next prekey ID: %w", err)
	}
	return uint32(lastKeyID.Int32) + 1, nil
}

func (s *SQLStore) GenOnePreKey() (*keys.PreKey, error) {
	s.preKeyLock.Lock()
	defer s.preKeyLock.Unlock()
	nextKeyID, err := s.getNextPreKeyID()
	if err != nil {
		return nil, err
	}
	return s.genOnePreKey(nextKeyID, true)
}

func (s *SQLStore) GetOrGenPreKeys(count uint32) ([]*keys.PreKey, error) {
	s.preKeyLock.Lock()
	defer s.preKeyLock.Unlock()

	res, err := s.db.Query(s.dialectQuery(getUnuploadedPreKeysQuery), s.JID, count)
	if err != nil {
		return nil, fmt.Errorf("failed to query existing prekeys: %w", err)
	}
	newKeys := make([]*keys.PreKey, count)
	var existingCount uint32
	for res.Next() {
		var key *keys.PreKey
		key, err = scanPreKey(res)
		if err != nil {
			return nil, err
		} else if key != nil {
			newKeys[existingCount] = key
			existingCount++
		}
	}

	if existingCount < uint32(len(newKeys)) {
		var nextKeyID uint32
		nextKeyID, err = s.getNextPreKeyID()
		if err != nil {
			return nil, err
		}
		for i := existingCount; i < count; i++ {
			newKeys[i], err = s.genOnePreKey(nextKeyID, false)
			if err != nil {
				return nil, fmt.Errorf("failed to generate prekey: %w", err)
			}
			nextKeyID++
		}
	}

	return newKeys, nil
}

func scanPreKey(row scannable) (*keys.PreKey, error) {
	var priv []byte
	var id uint32
	err := row.Scan(&id, &priv)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	} else if len(priv) != 32 {
		return nil, ErrInvalidLength
	}
	return &keys.PreKey{
		KeyPair: *keys.NewKeyPairFromPrivateKey(*(*[32]byte)(priv)),
		KeyID:   id,
	}, nil
}

func (s *SQLStore) GetPreKey(id uint32) (*keys.PreKey, error) {
	return scanPreKey(s.db.QueryRow(s.dialectQuery(getPreKeyQuery), s.JID, id))
}

func (s *SQLStore) RemovePreKey(id uint32) error {
	_, err := s.db.Exec(s.dialectQuery(deletePreKeyQuery), s.JID, id)
	return err
}

func (s *SQLStore) MarkPreKeysAsUploaded(upToID uint32) error {
	_, err := s.db.Exec(s.dialectQuery(markPreKeysAsUploadedQuery), s.JID, upToID)
	return err
}

func (s *SQLStore) UploadedPreKeyCount() (count int, err error) {
	err = s.db.QueryRow(s.dialectQuery(getUploadedPreKeyCountQuery), s.JID).Scan(&count)
	return
}

const (
	getSenderKeyQueryGeneric = `SELECT sender_key FROM whatsmeow_sender_keys WHERE our_jid=$1 AND chat_id=$2 AND sender_id=$3`
	getSenderKeyQueryMySQL   = `SELECT sender_key FROM whatsmeow_sender_keys WHERE our_jid=? AND chat_id=? AND sender_id=?`
	putSenderKeyQuery        = `
		INSERT INTO whatsmeow_sender_keys (our_jid, chat_id, sender_id, sender_key) VALUES ($1, $2, $3, $4)
		ON CONFLICT (our_jid, chat_id, sender_id) DO UPDATE SET sender_key=excluded.sender_key
	`
)

func (s *SQLStore) PutSenderKey(group, user string, session []byte) error {
	if s.dialect == "mysql" {
		// Use a direct MySQL query with proper backtick escaping
		_, err := s.db.Exec(`
			INSERT INTO whatsmeow_sender_keys (our_jid, chat_id, sender_id, sender_key)
			VALUES (?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE sender_key=VALUES(sender_key)
		`, s.JID, group, user, session)
		return err
	}
	_, err := s.db.Exec(s.dialectQuery(putSenderKeyQuery), s.JID, group, user, session)
	return err
}

func (s *SQLStore) GetSenderKey(group, user string) (key []byte, err error) {
	if s.dialect == "mysql" {
		// Use direct MySQL query to avoid any potential dialect conversion issues
		err = s.db.QueryRow(getSenderKeyQueryMySQL, s.JID, group, user).Scan(&key)
	} else {
		err = s.db.QueryRow(s.dialectQuery(getSenderKeyQueryGeneric), s.JID, group, user).Scan(&key)
	}
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return
}

const (
	putAppStateSyncKeyQuery = `
		INSERT INTO whatsmeow_app_state_sync_keys (jid, key_id, key_data, timestamp, fingerprint) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (jid, key_id) DO UPDATE
			SET key_data=excluded.key_data, timestamp=excluded.timestamp, fingerprint=excluded.fingerprint
			WHERE excluded.timestamp > whatsmeow_app_state_sync_keys.timestamp
	`
	getAppStateSyncKeyQuery         = `SELECT key_data, timestamp, fingerprint FROM whatsmeow_app_state_sync_keys WHERE jid=$1 AND key_id=$2`
	getLatestAppStateSyncKeyIDQuery = `SELECT key_id FROM whatsmeow_app_state_sync_keys WHERE jid=$1 ORDER BY timestamp DESC LIMIT 1`
)

func (s *SQLStore) PutAppStateSyncKey(id []byte, key store.AppStateSyncKey) error {
	if s.dialect == "mysql" {
		// Use standard MySQL syntax that works in all versions
		tx, err := s.db.Begin()
		if err != nil {
			return err
		}

		// Insert or update as necessary
		_, err = tx.Exec(`
				INSERT INTO whatsmeow_app_state_sync_keys (jid, key_id, key_data, timestamp, fingerprint)
				VALUES (?, ?, ?, ?, ?)
				ON DUPLICATE KEY UPDATE
					key_data=VALUES(key_data),
					timestamp=VALUES(timestamp),
					fingerprint=VALUES(fingerprint)
			`, s.JID, id, key.Data, key.Timestamp, key.Fingerprint)

		if err != nil {
			tx.Rollback()
			return err
		}

		// Log the key we're storing to aid in debugging
		s.log.Debugf("Stored app state key: %X with timestamp %d", id, key.Timestamp)

		return tx.Commit()
	}

	// For other databases, use the standard query
	_, err := s.db.Exec(s.dialectQuery(putAppStateSyncKeyQuery), s.JID, id, key.Data, key.Timestamp, key.Fingerprint)
	return err
}

func (s *SQLStore) GetAppStateSyncKey(id []byte) (*store.AppStateSyncKey, error) {
	var key store.AppStateSyncKey

	// Special handling for MySQL - binary key lookup can be problematic
	if s.dialect == "mysql" {
		// First try with standard query
		err := s.db.QueryRow("SELECT key_data, timestamp, fingerprint FROM whatsmeow_app_state_sync_keys WHERE jid=? AND key_id=?",
			s.JID, id).Scan(&key.Data, &key.Timestamp, &key.Fingerprint)

		if err != nil && errors.Is(err, sql.ErrNoRows) {
			// Try searching by hex representation as backup
			hexID := fmt.Sprintf("%X", id)
			s.log.Debugf("Key %s not found directly, trying hex lookup", hexID)

			err = s.db.QueryRow("SELECT key_data, timestamp, fingerprint FROM whatsmeow_app_state_sync_keys WHERE jid=? AND HEX(key_id)=?",
				s.JID, hexID).Scan(&key.Data, &key.Timestamp, &key.Fingerprint)

			if errors.Is(err, sql.ErrNoRows) {
				// Log the exact key we're looking for to help debugging
				s.log.Warnf("App state key not found: %X", id)
				return nil, nil
			}
			return &key, err
		}

		return &key, err
	}

	// Standard path for other databases
	err := s.db.QueryRow(s.dialectQuery(getAppStateSyncKeyQuery), s.JID, id).Scan(&key.Data, &key.Timestamp, &key.Fingerprint)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &key, err
}

func (s *SQLStore) GetLatestAppStateSyncKeyID() ([]byte, error) {
	var keyID []byte
	err := s.db.QueryRow(s.dialectQuery(getLatestAppStateSyncKeyIDQuery), s.JID).Scan(&keyID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return keyID, err
}

const (
	putAppStateVersionQuery = `
		INSERT INTO whatsmeow_app_state_version (jid, name, version, hash) VALUES ($1, $2, $3, $4)
		ON CONFLICT (jid, name) DO UPDATE SET version=excluded.version, hash=excluded.hash
	`
	getAppStateVersionQuery                 = `SELECT version, hash FROM whatsmeow_app_state_version WHERE jid=$1 AND name=$2`
	deleteAppStateVersionQuery              = `DELETE FROM whatsmeow_app_state_version WHERE jid=$1 AND name=$2`
	putAppStateMutationMACsQuery            = `INSERT INTO whatsmeow_app_state_mutation_macs (jid, name, version, index_mac, value_mac) VALUES `
	deleteAppStateMutationMACsQueryPostgres = `DELETE FROM whatsmeow_app_state_mutation_macs WHERE jid=$1 AND name=$2 AND index_mac=ANY($3::bytea[])`
	deleteAppStateMutationMACsQueryGeneric  = `DELETE FROM whatsmeow_app_state_mutation_macs WHERE jid=$1 AND name=$2 AND index_mac IN `
	getAppStateMutationMACQuery             = `SELECT value_mac FROM whatsmeow_app_state_mutation_macs WHERE jid=$1 AND name=$2 AND index_mac=$3 ORDER BY version DESC LIMIT 1`
)

func (s *SQLStore) PutAppStateVersion(name string, version uint64, hash [128]byte) error {
	_, err := s.db.Exec(s.dialectQuery(putAppStateVersionQuery), s.JID, name, version, hash[:])
	return err
}

func (s *SQLStore) GetAppStateVersion(name string) (version uint64, hash [128]byte, err error) {
	var uncheckedHash []byte
	err = s.db.QueryRow(s.dialectQuery(getAppStateVersionQuery), s.JID, name).Scan(&version, &uncheckedHash)
	if errors.Is(err, sql.ErrNoRows) {
		// version will be 0 and hash will be an empty array, which is the correct initial state
		err = nil
	} else if err != nil {
		// There's an error, just return it
	} else if len(uncheckedHash) != 128 {
		// This shouldn't happen
		err = ErrInvalidLength
	} else {
		// No errors, convert hash slice to array
		hash = *(*[128]byte)(uncheckedHash)
	}
	return
}

func (s *SQLStore) DeleteAppStateVersion(name string) error {
	_, err := s.db.Exec(s.dialectQuery(deleteAppStateVersionQuery), s.JID, name)
	return err
}

type execable interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

// Fix the putAppStateMutationMACs function for better MySQL compatibility
func (s *SQLStore) putAppStateMutationMACs(tx execable, name string, version uint64, mutations []store.AppStateMutationMAC) error {
	// Use simpler, more compatible approach for MySQL
	if s.dialect == "mysql" {
		// For MySQL, use individual INSERT statements in a transaction for maximum compatibility
		for _, mutation := range mutations {
			query := "INSERT INTO whatsmeow_app_state_mutation_macs (jid, name, version, index_mac, value_mac) VALUES (?, ?, ?, ?, ?)"
			_, err := tx.Exec(query, s.JID, name, version, mutation.IndexMAC, mutation.ValueMAC)
			if err != nil {
				return err
			}
		}
		return nil
	} else if s.dialect == "sqlite3" {
		placeholders := make([]string, len(mutations))
		args := make([]interface{}, 0, len(mutations)*5) // 5 args per mutation

		for i, mutation := range mutations {
			placeholders[i] = "(?, ?, ?, ?, ?)"
			args = append(args, s.JID, name, version, mutation.IndexMAC, mutation.ValueMAC)
		}

		query := "INSERT INTO whatsmeow_app_state_mutation_macs (jid, name, version, index_mac, value_mac) VALUES " +
			strings.Join(placeholders, ",")

		_, err := tx.Exec(query, args...)
		return err
	} else {
		// PostgreSQL specific code for handling array parameters
		placeholders := make([]string, len(mutations))
		argsCount := 1
		args := make([]interface{}, 0, len(mutations)*2+3)
		args = append(args, s.JID, name, version)

		for i, mutation := range mutations {
			baseIndex := argsCount
			args = append(args, mutation.IndexMAC, mutation.ValueMAC)
			placeholders[i] = fmt.Sprintf("($1, $2, $3, $%d, $%d)", baseIndex+1, baseIndex+2)
			argsCount += 2
		}

		query := "INSERT INTO whatsmeow_app_state_mutation_macs (jid, name, version, index_mac, value_mac) VALUES " +
			strings.Join(placeholders, ",")

		_, err := tx.Exec(query, args...)
		return err
	}
}

const mutationBatchSize = 400

func (s *SQLStore) PutAppStateMutationMACs(name string, version uint64, mutations []store.AppStateMutationMAC) error {
	if len(mutations) > mutationBatchSize {
		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}
		for i := 0; i < len(mutations); i += mutationBatchSize {
			var mutationSlice []store.AppStateMutationMAC
			if len(mutations) > i+mutationBatchSize {
				mutationSlice = mutations[i : i+mutationBatchSize]
			} else {
				mutationSlice = mutations[i:]
			}
			err = s.putAppStateMutationMACs(tx, name, version, mutationSlice)
			if err != nil {
				_ = tx.Rollback()
				return err
			}
		}
		err = tx.Commit()
		if err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
		return nil
	} else if len(mutations) > 0 {
		return s.putAppStateMutationMACs(s.db, name, version, mutations)
	}
	return nil
}

func (s *SQLStore) DeleteAppStateMutationMACs(name string, indexMACs [][]byte) (err error) {
	if len(indexMACs) == 0 {
		return
	}
	if s.dialect == "postgres" && PostgresArrayWrapper != nil {
		_, err = s.db.Exec(s.dialectQuery(deleteAppStateMutationMACsQueryPostgres), s.JID, name, PostgresArrayWrapper(indexMACs))
	} else {
		args := make([]interface{}, 2+len(indexMACs))
		args[0] = s.JID
		args[1] = name
		queryParts := make([]string, len(indexMACs))
		for i, item := range indexMACs {
			args[2+i] = item
			if s.dialect == "mysql" || s.dialect == "sqlite" {
				queryParts[i] = "?"
			} else {
				queryParts[i] = fmt.Sprintf("$%d", i+3)
			}
		}
		_, err = s.db.Exec(s.dialectQuery(deleteAppStateMutationMACsQueryGeneric)+"("+strings.Join(queryParts, ",")+")", args...)
	}
	return
}

func (s *SQLStore) GetAppStateMutationMAC(name string, indexMAC []byte) (valueMAC []byte, err error) {
	err = s.db.QueryRow(s.dialectQuery(getAppStateMutationMACQuery), s.JID, name, indexMAC).Scan(&valueMAC)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return
}

const (
	putContactNameQuery = `
		INSERT INTO whatsmeow_contacts (our_jid, their_jid, first_name, full_name) VALUES ($1, $2, $3, $4)
		ON CONFLICT (our_jid, their_jid) DO UPDATE SET first_name=excluded.first_name, full_name=excluded.full_name
	`
	putManyContactNamesQuery = `
		INSERT INTO whatsmeow_contacts (our_jid, their_jid, first_name, full_name)
		VALUES %s
		ON CONFLICT (our_jid, their_jid) DO UPDATE SET first_name=excluded.first_name, full_name=excluded.full_name
	`
	putPushNameQuery = `
		INSERT INTO whatsmeow_contacts (our_jid, their_jid, push_name) VALUES ($1, $2, $3)
		ON CONFLICT (our_jid, their_jid) DO UPDATE SET push_name=excluded.push_name
	`
	putBusinessNameQuery = `
		INSERT INTO whatsmeow_contacts (our_jid, their_jid, business_name) VALUES ($1, $2, $3)
		ON CONFLICT (our_jid, their_jid) DO UPDATE SET business_name=excluded.business_name
	`
	getContactQuery = `
		SELECT first_name, full_name, push_name, business_name FROM whatsmeow_contacts WHERE our_jid=$1 AND their_jid=$2
	`
	getAllContactsQuery = `
		SELECT their_jid, first_name, full_name, push_name, business_name FROM whatsmeow_contacts WHERE our_jid=$1
	`
)

func (s *SQLStore) PutPushName(user types.JID, pushName string) (bool, string, error) {
	s.contactCacheLock.Lock()
	defer s.contactCacheLock.Unlock()

	cached, err := s.getContact(user)
	if err != nil {
		return false, "", err
	}
	if cached.PushName != pushName {
		_, err = s.db.Exec(s.dialectQuery(putPushNameQuery), s.JID, user, pushName)
		if err != nil {
			return false, "", err
		}
		previousName := cached.PushName
		cached.PushName = pushName
		cached.Found = true
		return true, previousName, nil
	}
	return false, "", nil
}

func (s *SQLStore) PutBusinessName(user types.JID, businessName string) (bool, string, error) {
	s.contactCacheLock.Lock()
	defer s.contactCacheLock.Unlock()

	cached, err := s.getContact(user)
	if err != nil {
		return false, "", err
	}
	if cached.BusinessName != businessName {
		_, err = s.db.Exec(s.dialectQuery(putBusinessNameQuery), s.JID, user, businessName)
		if err != nil {
			return false, "", err
		}
		previousName := cached.BusinessName
		cached.BusinessName = businessName
		cached.Found = true
		return true, previousName, nil
	}
	return false, "", nil
}

func (s *SQLStore) PutContactName(user types.JID, firstName, fullName string) error {
	s.contactCacheLock.Lock()
	defer s.contactCacheLock.Unlock()

	cached, err := s.getContact(user)
	if err != nil {
		return err
	}
	if cached.FirstName != firstName || cached.FullName != fullName {
		_, err = s.db.Exec(s.dialectQuery(putContactNameQuery), s.JID, user, firstName, fullName)
		if err != nil {
			return err
		}
		cached.FirstName = firstName
		cached.FullName = fullName
		cached.Found = true
	}
	return nil
}

const contactBatchSize = 300

func (s *SQLStore) putContactNamesBatch(tx execable, contacts []store.ContactEntry) error {
	values := make([]interface{}, 1, 1+len(contacts)*3)
	queryParts := make([]string, 0, len(contacts))
	values[0] = s.JID
	placeholderSyntax := "($1, $%d, $%d, $%d)"
	if s.dialect == "sqlite3" || s.dialect == "mysql" {
		placeholderSyntax = "(?, ?, ?, ?)"
	}
	i := 0
	handledContacts := make(map[types.JID]struct{}, len(contacts))
	for _, contact := range contacts {
		if contact.JID.IsEmpty() {
			s.log.Warnf("Empty contact info in mass insert: %+v", contact)
			continue
		}
		// The whole query will break if there are duplicates, so make sure there aren't any duplicates
		_, alreadyHandled := handledContacts[contact.JID]
		if alreadyHandled {
			s.log.Warnf("Duplicate contact info for %s in mass insert", contact.JID)
			continue
		}
		handledContacts[contact.JID] = struct{}{}
		baseIndex := i*3 + 1
		values = append(values, contact.JID.String(), contact.FirstName, contact.FullName)
		if s.dialect == "sqlite3" || s.dialect == "mysql" {
			queryParts = append(queryParts, placeholderSyntax)
		} else {
			queryParts = append(queryParts, fmt.Sprintf(placeholderSyntax, baseIndex+1, baseIndex+2, baseIndex+3))
		}
		i++
	}
	_, err := tx.Exec(s.dialectQuery(fmt.Sprintf(putManyContactNamesQuery, strings.Join(queryParts, ","))), values...)
	return err
}

func (s *SQLStore) PutAllContactNames(contacts []store.ContactEntry) error {
	if len(contacts) > contactBatchSize {
		tx, err := s.db.Begin()
		if err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}
		for i := 0; i < len(contacts); i += contactBatchSize {
			var contactSlice []store.ContactEntry
			if len(contacts) > i+contactBatchSize {
				contactSlice = contacts[i : i+contactBatchSize]
			} else {
				contactSlice = contacts[i:]
			}
			err = s.putContactNamesBatch(tx, contactSlice)
			if err != nil {
				_ = tx.Rollback()
				return err
			}
		}
		err = tx.Commit()
		if err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	} else if len(contacts) > 0 {
		err := s.putContactNamesBatch(s.db, contacts)
		if err != nil {
			return err
		}
	} else {
		return nil
	}
	s.contactCacheLock.Lock()
	// Just clear the cache, fetching pushnames and business names would be too much effort
	s.contactCache = make(map[types.JID]*types.ContactInfo)
	s.contactCacheLock.Unlock()
	return nil
}

func (s *SQLStore) getContact(user types.JID) (*types.ContactInfo, error) {
	cached, ok := s.contactCache[user]
	if ok {
		return cached, nil
	}

	var first, full, push, business sql.NullString
	err := s.db.QueryRow(s.dialectQuery(getContactQuery), s.JID, user).Scan(&first, &full, &push, &business)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	info := &types.ContactInfo{
		Found:        err == nil,
		FirstName:    first.String,
		FullName:     full.String,
		PushName:     push.String,
		BusinessName: business.String,
	}
	s.contactCache[user] = info
	return info, nil
}

func (s *SQLStore) GetContact(user types.JID) (types.ContactInfo, error) {
	s.contactCacheLock.Lock()
	info, err := s.getContact(user)
	s.contactCacheLock.Unlock()
	if err != nil {
		return types.ContactInfo{}, err
	}
	return *info, nil
}

func (s *SQLStore) GetAllContacts() (map[types.JID]types.ContactInfo, error) {
	s.contactCacheLock.Lock()
	defer s.contactCacheLock.Unlock()
	rows, err := s.db.Query(s.dialectQuery(getAllContactsQuery), s.JID)
	if err != nil {
		return nil, err
	}
	output := make(map[types.JID]types.ContactInfo, len(s.contactCache))
	for rows.Next() {
		var jid types.JID
		var first, full, push, business sql.NullString
		err = rows.Scan(&jid, &first, &full, &push, &business)
		if err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		info := types.ContactInfo{
			Found:        true,
			FirstName:    first.String,
			FullName:     full.String,
			PushName:     push.String,
			BusinessName: business.String,
		}
		output[jid] = info
		s.contactCache[jid] = &info
	}
	return output, nil
}

const (
	putChatSettingQueryPostgres = `
		INSERT INTO whatsmeow_chat_settings (our_jid, chat_jid, %[1]s) VALUES ($1, $2, $3)
		ON CONFLICT (our_jid, chat_jid) DO UPDATE SET %[1]s=excluded.%[1]s
	`
	putChatSettingQueryMySQL = `
		INSERT INTO whatsmeow_chat_settings (our_jid, chat_jid, %[1]s) VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE %[1]s=VALUES(%[1]s)
	`
	putChatSettingQuerySQLite = `
		INSERT INTO whatsmeow_chat_settings (our_jid, chat_jid, %[1]s) VALUES (?, ?, ?)
		ON CONFLICT (our_jid, chat_jid) DO UPDATE SET %[1]s=excluded.%[1]s
	`
	getChatSettingsQuery = `
		SELECT muted_until, pinned, archived FROM whatsmeow_chat_settings WHERE our_jid=$1 AND chat_jid=$2
	`
)

func (s *SQLStore) PutMutedUntil(chat types.JID, mutedUntil time.Time) error {
	var val int64
	if !mutedUntil.IsZero() {
		val = mutedUntil.Unix()
	}

	var query string
	if s.dialect == "mysql" {
		query = fmt.Sprintf(putChatSettingQueryMySQL, "muted_until")
	} else if s.dialect == "sqlite3" {
		query = fmt.Sprintf(putChatSettingQuerySQLite, "muted_until")
	} else {
		query = fmt.Sprintf(putChatSettingQueryPostgres, "muted_until")
	}

	_, err := s.db.Exec(s.dialectQuery(query), s.JID, chat, val)
	return err
}

func (s *SQLStore) PutPinned(chat types.JID, pinned bool) error {
	var query string
	if s.dialect == "mysql" {
		query = fmt.Sprintf(putChatSettingQueryMySQL, "pinned")
	} else if s.dialect == "sqlite3" {
		query = fmt.Sprintf(putChatSettingQuerySQLite, "pinned")
	} else {
		query = fmt.Sprintf(putChatSettingQueryPostgres, "pinned")
	}

	_, err := s.db.Exec(s.dialectQuery(query), s.JID, chat, pinned)
	return err
}

func (s *SQLStore) PutArchived(chat types.JID, archived bool) error {
	var query string
	if s.dialect == "mysql" {
		query = fmt.Sprintf(putChatSettingQueryMySQL, "archived")
	} else if s.dialect == "sqlite3" {
		query = fmt.Sprintf(putChatSettingQuerySQLite, "archived")
	} else {
		query = fmt.Sprintf(putChatSettingQueryPostgres, "archived")
	}

	_, err := s.db.Exec(s.dialectQuery(query), s.JID, chat, archived)
	return err
}

func (s *SQLStore) GetChatSettings(chat types.JID) (settings types.LocalChatSettings, err error) {
	var mutedUntil int64
	err = s.db.QueryRow(s.dialectQuery(getChatSettingsQuery), s.JID, chat).Scan(&mutedUntil, &settings.Pinned, &settings.Archived)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	} else if err != nil {
		return
	} else {
		settings.Found = true
	}
	if mutedUntil != 0 {
		settings.MutedUntil = time.Unix(mutedUntil, 0)
	}
	return
}

const (
	putMsgSecret = `
		INSERT INTO whatsmeow_message_secrets (our_jid, chat_jid, sender_jid, message_id, ` + "`key`" + `)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (our_jid, chat_jid, sender_jid, message_id) DO NOTHING
	`
	getMsgSecret = `
		SELECT key FROM whatsmeow_message_secrets WHERE our_jid=$1 AND chat_jid=$2 AND sender_jid=$3 AND message_id=$4
	`
)

func (s *SQLStore) PutMessageSecrets(inserts []store.MessageSecretInsert) (err error) {
	// Smaller batch size to reduce lock contention
	const msgSecretBatchSize = 50

	// For large batches, process in smaller chunks
	if len(inserts) > msgSecretBatchSize {
		for i := 0; i < len(inserts); i += msgSecretBatchSize {
			end := i + msgSecretBatchSize
			if end > len(inserts) {
				end = len(inserts)
			}

			// Process this smaller batch with retry logic
			batchErr := s.putMessageSecretsWithRetry(inserts[i:end])
			if batchErr != nil {
				return batchErr
			}
		}
		return nil
	}

	// For small batches, use retry logic directly
	return s.putMessageSecretsWithRetry(inserts)
}

func (s *SQLStore) putMessageSecretsWithRetry(inserts []store.MessageSecretInsert) error {
	maxRetries := 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := s.putMessageSecretsInternal(inserts)

		// If successful or not a lock timeout error, return immediately
		if err == nil || (s.dialect == "mysql" && !strings.Contains(err.Error(), "Lock wait timeout exceeded")) {
			return err
		}

		// Log the retry attempt
		s.log.Warnf("Lock timeout storing message secrets (attempt %d/%d), retrying after delay",
			attempt+1, maxRetries)

		// Wait with progressive backoff before retrying
		backoffMs := 100 * (attempt + 1)
		time.Sleep(time.Duration(backoffMs) * time.Millisecond)
	}

	// If we get here, all retries failed
	return fmt.Errorf("failed to store message secrets after %d retry attempts", maxRetries)
}

func (s *SQLStore) putMessageSecretsInternal(inserts []store.MessageSecretInsert) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Use the correct query based on dialect
	var query string
	if s.dialect == "mysql" {
		query = `INSERT INTO whatsmeow_message_secrets (our_jid, chat_jid, sender_jid, message_id, ` + "`key`" + `)
				VALUES (?, ?, ?, ?, ?)
				ON DUPLICATE KEY UPDATE our_jid=our_jid`
	} else {
		query = s.dialectQuery(putMsgSecret)
	}

	for _, insert := range inserts {
		_, err = tx.Exec(query, s.JID, insert.Chat.ToNonAD(), insert.Sender.ToNonAD(), insert.ID, insert.Secret)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert message secret: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (s *SQLStore) PutMessageSecret(chat, sender types.JID, id types.MessageID, secret []byte) (err error) {
	// For single message secrets, also use retry logic for consistency
	if s.dialect == "mysql" {
		// Try with retries for MySQL
		for attempt := 0; attempt < 3; attempt++ {
			query := `INSERT INTO whatsmeow_message_secrets (our_jid, chat_jid, sender_jid, message_id, ` + "`key`" + `)
					VALUES (?, ?, ?, ?, ?)
					ON DUPLICATE KEY UPDATE our_jid=our_jid`
			_, err = s.db.Exec(query, s.JID, chat.ToNonAD(), sender.ToNonAD(), id, secret)

			// If successful or not a lock timeout, break
			if err == nil || !strings.Contains(err.Error(), "Lock wait timeout exceeded") {
				break
			}

			// Wait briefly before retrying
			time.Sleep(time.Duration(50*(attempt+1)) * time.Millisecond)
		}
	} else {
		// Standard path for other databases
		_, err = s.db.Exec(s.dialectQuery(putMsgSecret), s.JID, chat.ToNonAD(), sender.ToNonAD(), id, secret)
	}
	return
}

func (s *SQLStore) GetMessageSecret(chat, sender types.JID, id types.MessageID) (secret []byte, err error) {
	err = s.db.QueryRow(s.dialectQuery(getMsgSecret), s.JID, chat.ToNonAD(), sender.ToNonAD(), id).Scan(&secret)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return
}

const (
	putPrivacyTokens = `
		INSERT INTO whatsmeow_privacy_tokens (our_jid, their_jid, token, timestamp)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (our_jid, their_jid) DO UPDATE SET token=EXCLUDED.token, timestamp=EXCLUDED.timestamp
	`
	getPrivacyToken = `SELECT token, timestamp FROM whatsmeow_privacy_tokens WHERE our_jid=$1 AND their_jid=$2`
)

func (s *SQLStore) PutPrivacyTokens(tokens ...store.PrivacyToken) error {
	if s.dialect == "mysql" {
		// Use a transaction for better error handling
		tx, err := s.db.Begin()
		if err != nil {
			return err
		}

		// Process each token individually for maximum compatibility with all MySQL versions
		for _, token := range tokens {
			query := `
				INSERT INTO whatsmeow_privacy_tokens (our_jid, their_jid, token, timestamp)
				VALUES (?, ?, ?, ?)
				ON DUPLICATE KEY UPDATE token=VALUES(token), timestamp=VALUES(timestamp)
				`

			_, err := tx.Exec(query, s.JID, token.User.ToNonAD().String(), token.Token, token.Timestamp.Unix())
			if err != nil {
				tx.Rollback()
				return err
			}
		}

		return tx.Commit()
	}

	// Standard approach for non-MySQL databases
	args := make([]any, 1+len(tokens)*3)
	placeholders := make([]string, len(tokens))
	args[0] = s.JID
	for i, token := range tokens {
		args[i*3+1] = token.User.ToNonAD().String()
		args[i*3+2] = token.Token
		args[i*3+3] = token.Timestamp.Unix()
		placeholders[i] = fmt.Sprintf("($1, $%d, $%d, $%d)", i*3+2, i*3+3, i*3+4)
	}
	query := strings.ReplaceAll(putPrivacyTokens, "($1, $2, $3, $4)", strings.Join(placeholders, ","))
	_, err := s.db.Exec(s.dialectQuery(query), args...)
	return err
}

func (s *SQLStore) GetPrivacyToken(user types.JID) (*store.PrivacyToken, error) {
	var token store.PrivacyToken
	token.User = user.ToNonAD()
	var ts int64
	err := s.db.QueryRow(s.dialectQuery(getPrivacyToken), s.JID, token.User).Scan(&token.Token, &ts)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	} else {
		token.Timestamp = time.Unix(ts, 0)
		return &token, nil
	}
}
