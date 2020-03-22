package database

import (
	"strconv"
	"time"

	"go.etcd.io/bbolt"
)

var (
	timeObj   = []byte("time")
	offsetObj = []byte("offset")
)

type Database bbolt.DB

// create new DB
func NewDB() (*Database, error) {
	db, err := bbolt.Open("tumtum.db", 0644, nil)
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(timeObj)
		return err
	})
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(offsetObj)
		return err
	})
	if err != nil {
		return nil, err
	}

	return (*Database)(db), nil
}

// closes DB
func (s *Database) Close() error {
	return s.get().Close()
}

func (s *Database) get() *bbolt.DB {
	return (*bbolt.DB)(s)
}

// returns time.Time{}
func (s *Database) GetTime(b string) (time.Time, error) {
	var i int64

	err := s.get().Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(timeObj)
		if err != nil {
			return err
		}

		data := b.Get([]byte("time"))
		if len(data) == 0 {
			return nil
		}

		i, err = strconv.ParseInt(string(data), 10, 64)
		return err
	})

	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(i, 0), nil
}

// saves time.Time{} in unix
func (s *Database) SetTime(t time.Time) error {
	return s.get().Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(timeObj)
		if err != nil {
			return err
		}

		s := strconv.FormatInt(t.Unix(), 10)
		return b.Put([]byte("time"), []byte(s))
	})
}

// use offset to paginate through the blog
// returns offset
func (s *Database) GetOffset(b string) (int64, error) {
	var offset int64

	err := s.get().Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(offsetObj)
		if err != nil {
			return err
		}

		data := b.Get([]byte("offset"))
		if len(data) == 0 {
			return nil
		}

		offset, err = strconv.ParseInt(string(data), 10, 64)
		return err
	})

	if err != nil {
		return 0, err
	}

	return offset, nil
}

// sets offset
func (s *Database) SetOffset(o int64) error {
	return s.get().Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(offsetObj)
		if err != nil {
			return err
		}

		s := strconv.FormatInt(o, 10)
		return b.Put([]byte("offset"), []byte(s))
	})
}
