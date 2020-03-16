package database

import (
    "go.etcd.io/bbolt"
)

// possibily relevant IDs to keep track of
var (
    highestID = []byte("highest_id")
    lowestID = []byte("lowest_id")
    currentID = []byte("current_id")
    IDs = [][]byte{highestID, lowestID, currentID}
)

type Database bbolt.DB

func newDB() (*Database, error) {
    db, err := bbolt.Open("tumtum.db", 0644, nil)
    if err != nil {
        return nil, err
    }

    for _, ID := range IDs {
        err = db.Update(func(tx *bbolt.Tx) error {
            _, err := tx.CreateBucketIfNotExists(ID)
            return err
        })
        if err != nil {
            return nil, err
        }
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


// TODO figure out how to implement this lol
// func (s *Database) setIDS(blogname string, allIDs []int64) error {
//     return nil
// }

// i don't feel like implementing the database just yet
// func (s *Database) GetHighestID(blogName string) (int64, error) {
//     return int64(11), nil
// }
// func (s *Database) SetHighestID(blogName string, int64 highestID) error {
//     return int64(11), nil
// }
//
// func (s *Database) GetLowestID(blogName string) (int64, error) {
//     return int64(11), nil
// }
// func (s *Database) SetLowestID(blogName string, int64 highestID) error {
//     return int64(11), nil
// }
//
// func (s *Database) GetCurrentID(blogName string) (int64, error) {
//     return int64(11), nil
// }
// func (s *Database) SetCurrentID(blogName string, int64 highestID) error {
//     return int64(11), nil
// }
