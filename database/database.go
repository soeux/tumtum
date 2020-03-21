package database

import (
	"time"

	"go.etcd.io/bbolt"
)

var (
	time_obj = []byte("time")
)

type Database bbolt.DB

func NewDB() (*Database, error) {
	db, err := bbolt.Open("tumtum.db", 0644, nil)
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(time_obj)
		return err
	})
	if err != nil {
		return nil, err
	}

	return (*Database)(db), nil
}

func (s *Database) Close() error {
	return s.get().Close()
}

func (s *Database) get() *bbolt.DB {
	return (*bbolt.DB)(s)
}

// cookies?????????????
// func (s *Database) GetCookies() (snapshot []byte, err error) {}
// func (s *Databse) SaveCookies(snapshot []byte) error {}

// instead of pagination through ID's, utilise tumblrs &before=timestamp to go thru a blog
// TODO implement these functions
func (s *Database) GetTime(b string) (time.Time, error) {
	return time.Now(), nil
}

func (s *Database) SetTime(b string, t time.Time) error {
	return nil
}
