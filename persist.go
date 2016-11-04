package mqtt

import "hilldan/db/redis"

// Persister manages sessions, retained packets, support persistance, delete
// the data must be json format
type Persister interface {
	Save(key, field string, data []byte) error
	Read(key, field string) (data []byte, err error)
	Delete(key, field string) error
	LoadAll(key string) (datas map[string][]byte, err error)
}

type redisPersist struct {
	db *redis.Client
}

func NewRedisPersist(redis *redis.Client) *redisPersist {
	return &redisPersist{
		db: redis,
	}
}

func (rp *redisPersist) Save(key, field string, data []byte) error {
	_, err := rp.db.Hset(key, field, data)
	return err
}
func (rp *redisPersist) Read(key, field string) (data []byte, err error) {
	data, err = rp.db.Hget(key, field)
	return
}
func (rp *redisPersist) Delete(key, field string) error {
	_, err := rp.db.Hdel(key, field)
	return err
}
func (rp *redisPersist) LoadAll(key string) (datas map[string][]byte, err error) {
	datas = make(map[string][]byte)
	err = rp.db.Hgetall(key, datas)
	return
}
