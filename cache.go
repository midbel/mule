package mule

import (
	"encoding/json"
	"io"
	"time"

	bolt "go.etcd.io/bbolt"
)

type Cache interface {
	Get(string, time.Duration) ([]byte, error)
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

func (b boltCache) Get(key string, ttl time.Duration) ([]byte, error) {
	var data []byte
	return data, b.DB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("data"))
		if b == nil {
			return errReusable
		}
		c := struct {
			When time.Time
			Data []byte
		}{}
		if err := json.Unmarshal(b.Get([]byte(key)), &c); err != nil {
			return errReusable
		}
		if time.Since(c.When) >= ttl {
			b.Delete([]byte(key))
			return errReusable
		}
		data = c.Data
		return nil
	})
}

func (b boltCache) Put(key string, data []byte) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("data"))
		if b == nil {
			return errReusable
		}
		c := struct {
			When time.Time
			Data []byte
		}{
			When: time.Now(),
			Data: data,
		}
		data, err := json.Marshal(c)
		if err == nil {
			err = b.Put([]byte(key), data)
		}
		return err
	})
}
