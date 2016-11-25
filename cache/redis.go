package cache

import (
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/reechou/gorobot/config"
)

var (
	DefaultKey = "realwxRedis"
)

type RedisCache struct {
	p         *redis.Pool
	redisInfo *config.RedisInfo
}

func NewRedisCache(redisInfo *config.RedisInfo) *RedisCache {
	return &RedisCache{redisInfo: redisInfo}
}

func (rc *RedisCache) do(commandName string, args ...interface{}) (reply interface{}, err error) {
	c := rc.p.Get()
	defer c.Close()

	return c.Do(commandName, args...)
}

func (rc *RedisCache) ZIncrby(set string, score int, key string) error {
	var err error
	if _, err = rc.do("zincrby", set, score, key); err != nil {
		return err
	}
	if _, err = rc.do("HSET", rc.redisInfo.Key, set, true); err != nil {
		return err
	}
	return err
}

func (rc *RedisCache) ZRevrange(set string, start, end int) []interface{} {
	var err error
	listR, err := rc.do("zrevrange", set, start, end, "withscores")
	if err != nil {
		return nil
	}
	list := listR.([]interface{})
	return list
}

func (rc *RedisCache) HSetNX(set, key string, value interface{}) (bool, error) {
	has, err := rc.do("HSETNX", set, key, value)
	if err != nil {
		return false, err
	}
	if has.(int64) == 0 {
		return false, nil
	}
	return true, nil
}

func (rc *RedisCache) Get(key string) interface{} {
	if v, err := rc.do("GET", key); err == nil {
		return v
	}
	return nil
}

func (rc *RedisCache) Put(key string, val interface{}) error {
	var err error
	if _, err = rc.do("SET", key, val); err != nil {
		return err
	}

	if _, err = rc.do("HSET", rc.redisInfo.Key, key, true); err != nil {
		return err
	}
	return err
}

func (rc *RedisCache) PutNX(key string, val interface{}) error {
	var err error
	if _, err = rc.do("SETNX", key, val); err != nil {
		return err
	}

	if _, err = rc.do("HSET", rc.redisInfo.Key, key, true); err != nil {
		return err
	}
	return err
}

func (rc *RedisCache) PutEX(key string, val interface{}, timeout time.Duration) error {
	var err error
	if _, err = rc.do("SETEX", key, int64(timeout/time.Second), val); err != nil {
		return err
	}

	if _, err = rc.do("HSET", rc.redisInfo.Key, key, true); err != nil {
		return err
	}
	return err
}

func (rc *RedisCache) Delete(key string) error {
	var err error
	if _, err = rc.do("DEL", key); err != nil {
		return err
	}
	_, err = rc.do("HDEL", rc.redisInfo.Key, key)
	return err
}

func (rc *RedisCache) ClearAll() error {
	cachedKeys, err := redis.Strings(rc.do("HKEYS", rc.redisInfo.Key))
	if err != nil {
		return err
	}
	for _, str := range cachedKeys {
		if _, err = rc.do("DEL", str); err != nil {
			return err
		}
	}
	_, err = rc.do("DEL", rc.redisInfo.Key)
	return err
}

func (rc *RedisCache) StartAndGC() error {
	if rc.redisInfo.Key == "" {
		rc.redisInfo.Key = DefaultKey
	}

	rc.connectInit()

	c := rc.p.Get()
	defer c.Close()

	return c.Err()
}

// connect to redis.
func (rc *RedisCache) connectInit() {
	dialFunc := func() (c redis.Conn, err error) {
		c, err = redis.Dial("tcp", rc.redisInfo.Conninfo)
		if err != nil {
			return nil, err
		}

		if rc.redisInfo.Password != "" {
			if _, err := c.Do("AUTH", rc.redisInfo.Password); err != nil {
				c.Close()
				return nil, err
			}
		}

		_, selecterr := c.Do("SELECT", rc.redisInfo.DbNum)
		if selecterr != nil {
			c.Close()
			return nil, selecterr
		}
		return
	}
	// initialize a new pool
	rc.p = &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 180 * time.Second,
		Dial:        dialFunc,
	}
}
