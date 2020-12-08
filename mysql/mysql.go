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

const (
	createSQL = `CREATE TABLE %s (
	k varchar(127) NOT NULL,
	v blob,
	createAt bigint(20) DEFAULT '0',
	expireAt bigint(20) DEFAULT '0',
	PRIMARY KEY (k),
	KEY idx_expireAt (expireAt)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
`
)

func now() int64 {
	return time.Now().Unix()
}

var (
	minCheckInterval = time.Second * 5
)

type MysqlCacheOption func(*mysqlCache)

func WithLogger(logger func(v ...interface{})) MysqlCacheOption {
	return MysqlCacheOption(func(c *mysqlCache) {
		c.logger = logger
	})
}

func WithNoCheck() MysqlCacheOption {
	return MysqlCacheOption(func(c *mysqlCache) {
		c.nocheck = true
	})
}

func WithCheckInterval(interval time.Duration) MysqlCacheOption {
	return MysqlCacheOption(func(c *mysqlCache) {
		c.interval = interval
	})
}

func NewMysqlCache(db *sql.DB, tableName string, opts ...MysqlCacheOption) cache.Cache {
	if tableName == "" {
		panic(errors.New("cache: table name invalid"))
	}
	c := &mysqlCache{db: db, tableName: tableName}

	c.putsql = fmt.Sprintf("insert into %s (k,v,createAt,expireAt) values(?,?,?,?) on duplicate key update v=values(v),createAt=values(createAt),expireAt=values(expireAt)", tableName)
	c.getsql = fmt.Sprintf("select v,createAt,expireAt from %s where k=? limit 1", tableName)
	c.setexpiresql1 = fmt.Sprintf("update from %s expireAt=createAt+? where k=?", tableName)
	c.setexpiresql2 = fmt.Sprintf("update from %s expireAt=? where k=?", tableName)
	c.delsql = fmt.Sprintf("delete from %s where k=?", tableName)
	c.expiresql = fmt.Sprintf("select k,v from %s where expireAt>=0&&expireAt<? limit 5", tableName)

	for _, opt := range opts {
		opt(c)
	}

	if c.interval < minCheckInterval {
		c.interval = minCheckInterval
	}
	if c.logger == nil {
		c.logger = log.Println
	}
	if !c.nocheck {
		ctx := context.Background()
		ctx, c.cancel = context.WithCancel(ctx)
		go c.expireInLoop(ctx)
	}
	return c
}

type entry struct {
	k        string
	v        []byte
	createAt int64
	expireAt int64
}

func (e *entry) Expired() bool {
	return e.TTL() == 0
}

// TTL -1: never expired, 0: expired, >0: not expired
func (e *entry) TTL() int64 {
	if e.expireAt < 0 {
		return -1
	}
	ttl := e.expireAt - now()
	if ttl < 0 {
		ttl = 0
	}
	return ttl
}

type mysqlCache struct {
	db            *sql.DB
	tableName     string
	getsql        string
	putsql        string
	delsql        string
	setexpiresql1 string
	setexpiresql2 string
	expiresql     string
	delexpiresql  string
	interval      time.Duration
	nocheck       bool
	expireHandler func([]byte, interface{})
	logger        func(...interface{})
	cancel        context.CancelFunc
}

func (c *mysqlCache) expireInLoop(ctx context.Context) {
	ticker := time.NewTicker(c.interval)

LOOP:
	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			break LOOP
		}
		c.expires(ctx)
	}
}

func (c *mysqlCache) expires(ctx context.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.logger(err)
		}
	}()

	for {
		done, err := c.deleteExpires(ctx)
		if err != nil {
			c.logger(err)
			break
		}
		if done {
			break
		}
	}
}

func (c *mysqlCache) deleteExpires(ctx context.Context) (done bool, err error) {
	var rows *sql.Rows
	rows, err = c.db.Query(c.expiresql, now())
	if err != nil {
		return
	}
	defer rows.Close()

	var ls []*entry

	for rows.Next() {
		var e entry
		err = rows.Scan(&e.k, &e.v)
		if err != nil {
			break
		}
		ls = append(ls, &e)
	}
	if err != nil {
		return
	}
	if len(ls) <= 0 {
		done = true
		return
	}
	var sqlBuilder strings.Builder
	sqlBuilder.WriteString("delete from ")
	sqlBuilder.WriteString(c.tableName)
	sqlBuilder.WriteString(" where k in(")
	sqlBuilder.WriteByte('\'')
	sqlBuilder.WriteString(ls[0].k)
	sqlBuilder.WriteByte('\'')
	for i := len(ls) - 1; i >= 1; i-- {
		sqlBuilder.WriteString(",'")
		sqlBuilder.WriteString(ls[i].k)
		sqlBuilder.WriteByte('\'')
	}
	sqlBuilder.WriteByte(')')
	_, err = c.db.Exec(sqlBuilder.String())
	if err != nil {
		return
	}
	if len(ls) < 5 {
		done = true
		return
	}
	h := c.expireHandler
	if h != nil {
		for _, e := range ls {
			h([]byte(e.k), e.v)
		}
	}
	return
}

func (c *mysqlCache) Get(ctx context.Context, k []byte) (interface{}, error) {
	v, _, err := c.GetAndTTL(ctx, k)
	return v, err
}

func (c *mysqlCache) GetAndTTL(ctx context.Context, k []byte) (interface{}, int64, error) {
	var err error
	row := c.db.QueryRowContext(ctx, c.getsql, cache.BytesToString(k))
	var e entry
	err = row.Scan(&e.v, &e.createAt, &e.expireAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, 0, cache.ErrNoKey
		}
		return nil, 0, err
	}
	ttl := e.TTL()
	if ttl == 0 {
		return nil, 0, cache.ErrNoKey
	}
	return e.v, ttl, nil
}

func (c *mysqlCache) TTL(ctx context.Context, k []byte) (int64, error) {
	_, ttl, err := c.GetAndTTL(ctx, k)
	return ttl, err
}

func (c *mysqlCache) Expire(ctx context.Context, k []byte, sec int64) error {
	var sqlStr = c.setexpiresql1
	if sec < 0 {
		sqlStr = c.setexpiresql2
		sec = -1
	}
	rs, err := c.db.ExecContext(ctx, sqlStr, sec, cache.BytesToString(k))
	if err != nil {
		return err
	}
	a, err := rs.RowsAffected()
	if err != nil {
		return err
	}
	if a <= 0 {
		return cache.ErrNoKey
	}
	return err
}

func (c *mysqlCache) Put(ctx context.Context, k []byte, v interface{}) error {
	return c.PutEx(ctx, k, v, -1)
}

func (c *mysqlCache) PutEx(ctx context.Context, k []byte, v interface{}, sec int64) error {
	var b []byte
	var err error
	switch p := v.(type) {
	case []byte:
		b = p
	case json.RawMessage:
		b = p
	case *cache.Event:
		b, err = p.Marshal(nil)
	default:
		return fmt.Errorf("cache: value %v invalid", v)
	}
	createAt := now()
	expireAt := createAt + sec
	if sec < 0 {
		expireAt = -1
	}
	_, err = c.db.ExecContext(ctx, c.putsql, cache.BytesToString(k), b, createAt, expireAt)
	return err
}

func (c *mysqlCache) Del(ctx context.Context, k []byte) error {
	var err error
	var v interface{}
	h := c.expireHandler
	if h != nil {
		v, _, err = c.GetAndTTL(ctx, k)
		if err != nil {
			return err
		}
	}
	_, err = c.db.ExecContext(ctx, c.delsql, cache.BytesToString(k))
	if err != nil {
		return err
	}
	if h != nil {
		h(k, v)
	}
	return err
}

func (c *mysqlCache) Tx(ctx context.Context, k []byte, fn func(interface{}) (interface{}, error)) error {
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	err = func() error {
		var err error
		row := tx.QueryRowContext(ctx, c.getsql, cache.BytesToString(k))
		var e entry
		err = row.Scan(&e.v, &e.createAt, &e.expireAt)
		if err != nil {
			if err == sql.ErrNoRows {
				return cache.ErrNoKey
			}
			return err
		}
		if e.Expired() {
			return cache.ErrNoKey
		}
		o, err := fn(e.v)
		var b []byte
		switch p := o.(type) {
		case []byte:
			b = p
		case json.RawMessage:
			b = p
		case *cache.Event:
			b, err = p.Marshal(nil)
		default:
			return fmt.Errorf("cache: value %v invalid", e.v)
		}
		_, err = c.db.ExecContext(ctx, c.putsql, cache.BytesToString(k), b, e.createAt, e.expireAt)
		return err
	}()

	if err != nil {
		return tx.Rollback()
	}
	return tx.Commit()
}

func (c *mysqlCache) ExpireHandler(h func([]byte, interface{})) {
	c.expireHandler = h
}
