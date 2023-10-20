package mule

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"time"

	bolt "go.etcd.io/bbolt"
)

type Entry struct {
	When time.Time
	Data []byte
	Sum  string
}

type Cache interface {
	Get(string, time.Duration) (Entry, error)
	Put(string, []byte) error
	io.Closer
}

type boltCache struct {
	*bolt.DB
}

func Bolt() (Cache, error) {
	db, err := bolt.Open(".bolt.db", 0600, nil)
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("data"))
		return err
	})
	if err != nil {
		return nil, err
	}
	return boltCache{
		DB: db,
	}, nil
}

func (b boltCache) Get(key string, ttl time.Duration) (Entry, error) {
	var e Entry
	return e, b.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("data"))
		if b == nil {
			return errReusable
		}
		if err := json.Unmarshal(b.Get([]byte(key)), &e); err != nil {
			return errReusable
		}
		if time.Since(e.When) >= ttl {
			b.Delete([]byte(key))
			return errReusable
		}
		return nil
	})
}

func (b boltCache) Put(key string, data []byte) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("data"))
		if b == nil {
			return errReusable
		}
		e := Entry {
			When: time.Now(),
			Data: data,
			Sum: fmt.Sprintf("%x", md5.Sum(data)),
		}
		data, err := json.Marshal(e)
		if err == nil {
			err = b.Put([]byte(key), data)
		}
		return err
	})
}
