package auth

import (
	"github.com/allentom/harukap/commons"
	"github.com/boltdb/bolt"
)

var (
	storeBucket = "tokens"
)

type TokenStoreManager struct {
	DB         *bolt.DB
	Serializer Serializer
	module     *AuthModule
}
type Serializer interface {
	Serialize(data interface{}) ([]byte, error)
	Deserialize(raw []byte) (commons.AuthUser, error)
}

func (m *TokenStoreManager) Init() error {
	db, err := bolt.Open("token.db", 0600, nil)
	if err != nil {
		return err
	}
	m.DB = db
	err = m.DB.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(storeBucket))
		return err
	})
	if err != nil {
		return err
	}
	return nil
}

func (m *TokenStoreManager) GetUserByToken(token string) (commons.AuthUser, error) {
	var auth commons.AuthUser
	var err error
	err = m.DB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(storeBucket))
		v := b.Get([]byte(token))
		auth, err = m.Serializer.Deserialize(v)
		return err
	})
	if auth == nil {
		authUser, err := m.module.ParseToken(token)
		if err != nil {
			return nil, err
		}
		err = m.DB.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(storeBucket))
			data, err := m.Serializer.Serialize(authUser)
			if err != nil {
				return err
			}
			err = b.Put([]byte(token), data)
			return err
		})
		return authUser, err
	}
	return auth, nil
}
