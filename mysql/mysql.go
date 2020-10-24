package mysql

import (
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
	v varchar(255) DEFAULT NULL,
	ex bigint(20) DEFAULT NULL,
	ctime bigint(20) DEFAULT NULL,
	PRIMARY KEY (k)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
`
)

func now() int64 {
	return time.Now().Unix()
}

var (
	minCheckInterval = time.Second * 10
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

	c.putsql = fmt.Sprintf("insert into %s (k,v,ex,ctime) values(?,?,?,?) on duplicate key update v=values(v),ex=values(ex),ctime=values(ctime)", tableName)
	c.getsql = fmt.Sprintf("select v,ex,ctime from %s where k=? limit 1", tableName)
	c.setexpiresql = fmt.Sprintf("update from %s ex=? where k=?", tableName)
	c.delsql = fmt.Sprintf("delete from %s where k=?", tableName)
	c.expiresql = fmt.Sprintf("select k,v from %s where k>? and ex+ctime<=? limit 50", tableName)

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
		go c.expireInLoop()
	}
	return c
}

type entry struct {
	k     string
	v     []byte
	ex    int64
	ctime int64
}

func (e *entry) Expired() bool {
	return e.TTL() == 0
}

// TTL -1: never expired, 0: expired, >0: not expired
func (e *entry) TTL() int64 {
	if e.ex < 0 {
		return -1
	}
	ttl := e.ex + e.ctime - now()
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
	setexpiresql  string
	expiresql     string
	delexpiresql  string
	interval      time.Duration
	nocheck       bool
	expireHandler func(interface{})
	logger        func(...interface{})
}

func (c *mysqlCache) expireInLoop() {
	ticker := time.NewTicker(c.interval)
	for {
		<-ticker.C
		c.expires()
	}
}

func (c *mysqlCache) expires() {
	defer func() {
		if err := recover(); err != nil {
			c.logger(err)
		}
	}()
	var k string
	var err error

	for {
		k, err = c.deleteExpires(k)
		if err != nil {
			c.logger(err)
			break
		}
		if k == "" {
			break
		}
	}
}

func (c *mysqlCache) deleteExpires(k string) (next string, err error) {
	var rows *sql.Rows
	rows, err = c.db.Query(c.expiresql, k, now())
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
		if e.k > next {
			next = e.k
		}
	}
	if len(ls) <= 0 {
		return
	}
	var sqlBuilder strings.Builder
	sqlBuilder.WriteString("delete from ")
	sqlBuilder.WriteString(c.tableName)
	sqlBuilder.WriteString(" where k in(")
	for i := len(ls) - 1; i >= 0; i-- {
		sqlBuilder.WriteByte('"')
		sqlBuilder.WriteString(ls[i].k)
		sqlBuilder.WriteByte('"')
	}
	sqlBuilder.WriteByte(')')
	_, err = c.db.Exec(sqlBuilder.String())
	if err != nil {
		return
	}

	h := c.expireHandler
	if h != nil {
		for _, e := range ls {
			h(e.v)
		}
	}
	return
}

func (c *mysqlCache) Get(k []byte) (interface{}, error) {
	v, _, err := c.GetAndTTL(k)
	return v, err
}

func (c *mysqlCache) GetAndTTL(k []byte) (interface{}, int64, error) {
	var err error
	row := c.db.QueryRow(c.getsql, cache.BytesToString(k))
	var e entry
	var v []byte
	err = row.Scan(&v, &e.ex, &e.ctime)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, 0, cache.ErrNoKey
		}
		return nil, 0, err
	}
	if e.Expired() {
		return nil, 0, cache.ErrNoKey
	}
	return v, e.TTL(), nil
}

func (c *mysqlCache) TTL(k []byte) (int64, error) {
	_, ttl, err := c.GetAndTTL(k)
	return ttl, err
}

func (c *mysqlCache) Expire(k []byte, sec int64) error {
	rs, err := c.db.Exec(c.setexpiresql, sec, cache.BytesToString(k))
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

func (c *mysqlCache) Put(k []byte, v interface{}) error {
	return c.PutEx(k, v, -1)
}

func (c *mysqlCache) PutEx(k []byte, v interface{}, sec int64) error {
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
	_, err = c.db.Exec(c.putsql, cache.BytesToString(k), b, sec, now())
	return err
}

func (c *mysqlCache) Del(k []byte) error {
	var err error
	var v interface{}
	h := c.expireHandler
	if h != nil {
		v, _, err = c.GetAndTTL(k)
		if err != nil {
			return err
		}
	}
	_, err = c.db.Exec(c.delsql, cache.BytesToString(k))
	if err != nil {
		return err
	}
	if h != nil {
		h(v)
	}
	return err
}

func (c *mysqlCache) Tx(k []byte, fn func(interface{}) (interface{}, error)) error {
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	err = func() error {
		var err error
		row := tx.QueryRow(c.getsql, cache.BytesToString(k))
		var e entry
		var v []byte
		err = row.Scan(&v, &e.ex, &e.ctime)
		if err != nil {
			if err == sql.ErrNoRows {
				return cache.ErrNoKey
			}
			return err
		}
		if e.Expired() {
			return cache.ErrNoKey
		}
		o, err := fn(v)
		var b []byte
		switch p := o.(type) {
		case []byte:
			b = p
		case json.RawMessage:
			b = p
		case *cache.Event:
			b, err = p.Marshal(nil)
		default:
			return fmt.Errorf("cache: value %v invalid", v)
		}
		_, err = c.db.Exec(c.putsql, cache.BytesToString(k), b, e.ex, e.ctime)
		return err
	}()

	if err != nil {
		return tx.Rollback()
	}
	return tx.Commit()

}

func (c *mysqlCache) ExpireHandler(h func(interface{})) {
	c.expireHandler = h
}
