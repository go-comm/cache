package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-comm/cache"
)

const createTableSQL = `CREATE TABLE IF NOT EXISTS %s (
	k varchar(127) NOT NULL DEFAULT '',
	v blob,           -- blob types: tinyblob(255B) blob(64KB) mediumblob(16MB) longblob(4GB)
	createdAt bigint NOT NULL DEFAULT 0,
	expiredAt bigint NOT NULL DEFAULT 0,
	PRIMARY KEY (k),
	KEY idx_expiredAt (expiredAt)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
`

func buildSQL(tableName string) sqlSet {
	return sqlSet{
		putSQL: fmt.Sprintf(
			`INSERT INTO %s (k, v, createdAt, expiredAt) VALUES (?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE v=VALUES(v), createdAt=VALUES(createdAt), expiredAt=VALUES(expiredAt)`, tableName),
		getSQL:          fmt.Sprintf(`SELECT v, createdAt, expiredAt FROM %s WHERE k=? LIMIT 1`, tableName),
		delSQL:          fmt.Sprintf(`DELETE FROM %s WHERE k=?`, tableName),
		expiredAtRelSQL: fmt.Sprintf(`UPDATE %s SET expiredAt=createdAt+? WHERE k=?`, tableName),
		expiredAtAbsSQL: fmt.Sprintf(`UPDATE %s SET expiredAt=? WHERE k=?`, tableName),
		expiredScanSQL:  fmt.Sprintf(`SELECT k FROM %s WHERE expiredAt>=0 AND expiredAt<? LIMIT ?`, tableName),
		deleteByKeysSQL: fmt.Sprintf(`DELETE FROM %s WHERE k IN`, tableName),
		clearSQL:        fmt.Sprintf(`DELETE FROM %s`, tableName),
	}
}

type sqlSet struct {
	putSQL, getSQL, delSQL           string
	expiredAtRelSQL, expiredAtAbsSQL string
	expiredScanSQL, deleteByKeysSQL  string
	clearSQL                         string
}

type Option func(*MysqlCache)

func WithLogger(logger func(v ...interface{})) Option {
	return func(c *MysqlCache) { c.logger = logger }
}
func WithNoExpireCheck() Option {
	return func(c *MysqlCache) { c.noCheck = true }
}
func WithCheckInterval(d time.Duration) Option {
	return func(c *MysqlCache) { c.checkInterval = d }
}
func WithBatchSize(n int) Option {
	return func(c *MysqlCache) { c.batchSize = n }
}
func WithAutoCreateTable() Option {
	return func(c *MysqlCache) { c.autoCreate = true }
}

const (
	defaultCheckInterval = 30 * time.Second
	minCheckInterval     = 5 * time.Second
	defaultBatchSize     = 100
)

type MysqlCache struct {
	db                  *sql.DB
	tableName           string
	sql                 sqlSet
	checkInterval       time.Duration
	batchSize           int
	noCheck, autoCreate bool
	logger              func(v ...interface{})
	cancel              context.CancelFunc
	expireHandler       func(k interface{}, v interface{})
}

func New(db *sql.DB, tableName string, opts ...Option) (*MysqlCache, error) {
	if db == nil {
		return nil, errors.New("mysql cache: db is nil")
	}
	if tableName == "" {
		return nil, errors.New("mysql cache: table name is empty")
	}
	c := &MysqlCache{
		db: db, tableName: tableName, sql: buildSQL(tableName),
		checkInterval: defaultCheckInterval, batchSize: defaultBatchSize,
		logger: log.Println,
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.checkInterval < minCheckInterval {
		c.checkInterval = minCheckInterval
	}
	if c.batchSize < 1 {
		c.batchSize = defaultBatchSize
	}
	if c.autoCreate {
		if _, err := db.Exec(fmt.Sprintf(createTableSQL, tableName)); err != nil {
			return nil, fmt.Errorf("mysql cache: auto create table: %w", err)
		}
	}
	if !c.noCheck {
		ctx, cancel := context.WithCancel(context.Background())
		c.cancel = cancel
		go c.expireLoop(ctx)
	}
	return c, nil
}

func (c *MysqlCache) Close() {
	if c.cancel != nil {
		c.cancel()
	}
}

func keyToString(k interface{}) string {
	switch d := k.(type) {
	case string:
		return d
	case []byte:
		return cache.BytesToStr(d)
	default:
		if s, ok := d.(interface{ String() string }); ok {
			return s.String()
		}
		return fmt.Sprintf("%v", d)
	}
}

// sqlValue converts v to a type that database/sql can handle natively.
// Basic types and cache.Valuer are passed through directly — database/sql
// handles driver.Valuer resolution automatically on ExecContext.
// Other types are JSON-encoded.
func sqlValue(v interface{}) (interface{}, error) {
	switch v.(type) {
	case string, []byte, json.RawMessage,
		cache.Valuer,
		bool,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return v, nil
	default:
		return json.Marshal(v)
	}
}

func now() int64 {
	return time.Now().Unix()
}

func entryTTL(expiredAt int64) int64 {
	if expiredAt < 0 {
		return -1
	}
	ttl := expiredAt - now()
	if ttl < 0 {
		return 0
	}
	return ttl
}

func (c *MysqlCache) Get(ctx context.Context, k interface{}) (interface{}, error) {
	v, _, err := c.getInternal(ctx, k)
	return v, err
}

func (c *MysqlCache) GetAndTTL(ctx context.Context, k interface{}) (interface{}, int64, error) {
	return c.getInternal(ctx, k)
}

func (c *MysqlCache) getInternal(ctx context.Context, k interface{}) (interface{}, int64, error) {
	key := keyToString(k)
	var v []byte
	var createdAt, expiredAt int64
	err := c.db.QueryRowContext(ctx, c.sql.getSQL, key).Scan(&v, &createdAt, &expiredAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, 0, cache.ErrNoKey
		}
		return nil, 0, err
	}
	ttl := entryTTL(expiredAt)
	if ttl == 0 {
		return nil, 0, cache.ErrNoKey
	}
	return v, ttl, nil
}

func (c *MysqlCache) TTL(ctx context.Context, k interface{}) (int64, error) {
	_, ttl, err := c.getInternal(ctx, k)
	return ttl, err
}

func (c *MysqlCache) Scan(ctx context.Context, k interface{}, scan cache.Scanner) error {
	v, err := c.Get(ctx, k)
	if err != nil {
		return err
	}
	return scan.Scan(v)
}

func (c *MysqlCache) ScanAndTTL(ctx context.Context, k interface{}, scan cache.Scanner) (int64, error) {
	v, ttl, err := c.getInternal(ctx, k)
	if err != nil {
		return 0, err
	}
	return ttl, scan.Scan(v)
}

func (c *MysqlCache) Put(ctx context.Context, k interface{}, v interface{}) error {
	return c.PutEx(ctx, k, v, -1)
}

func (c *MysqlCache) PutEx(ctx context.Context, k interface{}, v interface{}, sec int64) error {
	key := keyToString(k)
	b, err := sqlValue(v)
	if err != nil {
		return fmt.Errorf("mysql cache: resolve value: %w", err)
	}
	createdAt := now()
	expiredAt := int64(-1)
	if sec >= 0 {
		expiredAt = createdAt + sec
	}
	_, err = c.db.ExecContext(ctx, c.sql.putSQL, key, b, createdAt, expiredAt)
	return err
}

func (c *MysqlCache) Del(ctx context.Context, k interface{}) error {
	key := keyToString(k)
	var val interface{}
	var hasVal bool
	if c.expireHandler != nil {
		var err error
		val, _, err = c.getInternal(ctx, k)
		if err != nil && !errors.Is(err, cache.ErrNoKey) {
			return err
		}
		hasVal = err == nil
	}
	rs, err := c.db.ExecContext(ctx, c.sql.delSQL, key)
	if err != nil {
		return err
	}
	n, _ := rs.RowsAffected()
	if n == 0 {
		return cache.ErrNoKey
	}
	if hasVal && c.expireHandler != nil {
		go c.expireHandler(k, val)
	}
	return nil
}

func (c *MysqlCache) Expire(ctx context.Context, k interface{}, sec int64) error {
	key := keyToString(k)
	var sqlStr string
	var arg int64
	if sec < 0 {
		sqlStr = c.sql.expiredAtAbsSQL
		arg = -1
	} else {
		sqlStr = c.sql.expiredAtRelSQL
		arg = sec
	}
	rs, err := c.db.ExecContext(ctx, sqlStr, arg, key)
	if err != nil {
		return err
	}
	n, _ := rs.RowsAffected()
	if n == 0 {
		return cache.ErrNoKey
	}
	return nil
}

// Range iterates over all non-expired entries in the cache using cursor-based
// pagination on the primary key (k) to avoid a single full table scan.
// The iteration stops if fn returns an error, and that error is returned.
func (c *MysqlCache) Range(ctx context.Context, fn func(k interface{}, v interface{}) error) error {
	const pageSize = 500
	var lastKey string
	for {
		newKey, hasRow, err := c.rangeScan(ctx, lastKey, pageSize, fn)
		if err != nil {
			return err
		}
		if !hasRow {
			break
		}
		lastKey = newKey
	}
	return nil
}

// rangeScan queries a single page of rows and invokes fn for each non-expired entry.
// It returns the last key of the page, whether any row was found, and any error.
func (c *MysqlCache) rangeScan(ctx context.Context, lastKey string, limit int, fn func(k interface{}, v interface{}) error) (string, bool, error) {
	rows, err := c.db.QueryContext(ctx,
		"SELECT k, v, expiredAt FROM "+c.tableName+" WHERE k > ? ORDER BY k LIMIT ?",
		lastKey, limit)
	if err != nil {
		return lastKey, false, err
	}
	defer rows.Close()

	hasRow := false
	for rows.Next() {
		hasRow = true
		var k string
		var v []byte
		var expiredAt int64
		if err := rows.Scan(&k, &v, &expiredAt); err != nil {
			return lastKey, false, err
		}
		// Always advance the cursor, even for expired entries,
		// to avoid an infinite loop on a page full of expired rows.
		lastKey = k
		if entryTTL(expiredAt) == 0 {
			continue
		}
		if err := fn(k, v); err != nil {
			return lastKey, false, err
		}
	}
	if err := rows.Err(); err != nil {
		return lastKey, false, err
	}
	return lastKey, hasRow, nil
}

func (c *MysqlCache) Clear(ctx context.Context) error {
	_, err := c.db.ExecContext(ctx, c.sql.clearSQL)
	return err
}

func (c *MysqlCache) Tx(ctx context.Context, k interface{}, fn func(*cache.Entry) error) error {
	if fn == nil {
		return errors.New("mysql cache: tx fn is nil")
	}
	key := keyToString(k)
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	getForUpdate := c.sql.getSQL + ` FOR UPDATE`
	var v []byte
	var createdAt, expiredAt int64
	err = tx.QueryRowContext(ctx, getForUpdate, key).Scan(&v, &createdAt, &expiredAt)
	if err != nil {
		tx.Rollback()
		if errors.Is(err, sql.ErrNoRows) {
			return cache.ErrNoKey
		}
		return err
	}
	if entryTTL(expiredAt) == 0 {
		tx.Rollback()
		return cache.ErrNoKey
	}
	e := &cache.Entry{Value: v, CreatedAt: createdAt, ExpiredAt: expiredAt}
	err = fn(e)
	if err != nil {
		tx.Rollback()
		return err
	}
	resolved, err := sqlValue(e.Value)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("mysql cache: tx resolve value: %w", err)
	}
	_, err = tx.ExecContext(ctx, c.sql.putSQL, key, resolved, e.CreatedAt, e.ExpiredAt)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (c *MysqlCache) ExpireHandler(h func(k interface{}, v interface{})) {
	c.expireHandler = h
}

func (c *MysqlCache) expireLoop(ctx context.Context) {
	ticker := time.NewTicker(c.checkInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
		c.cleanupExpired(ctx)
	}
}

func (c *MysqlCache) cleanupExpired(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			c.logger("mysql cache: expire panic:", r)
		}
	}()
	for {
		done, err := c.deleteExpiredBatch(ctx)
		if err != nil {
			c.logger("mysql cache: expire cleanup:", err)
			return
		}
		if done {
			return
		}
	}
}

func (c *MysqlCache) deleteExpiredBatch(ctx context.Context) (bool, error) {
	rows, err := c.db.QueryContext(ctx, c.sql.expiredScanSQL, now(), c.batchSize)
	if err != nil {
		return false, err
	}
	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			rows.Close()
			return false, err
		}
		keys = append(keys, k)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return false, err
	}
	if len(keys) == 0 {
		return true, nil
	}
	var sb strings.Builder
	sb.WriteString(c.sql.deleteByKeysSQL)
	sb.WriteString(" (")
	args := make([]interface{}, len(keys))
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteByte('?')
		args[i] = k
	}
	sb.WriteByte(')')
	_, err = c.db.ExecContext(ctx, sb.String(), args...)
	if err != nil {
		return false, err
	}
	if c.expireHandler != nil {
		for _, k := range keys {
			go c.expireHandler(k, nil)
		}
	}
	return len(keys) < c.batchSize, nil
}
