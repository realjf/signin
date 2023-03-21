package signin

import (
	"time"

	"github.com/redis/go-redis/v9"
)

type Option func(signin ISignIn)

func WithDebug() Option {
	return func(signin ISignIn) {
		signin.SetDebug(true)
	}
}

// use something like: i8 u8 i16 u16 i32 u32 i64 u63
// !!! Signed integers support up to 64 bits, while unsigned integers support up to 63 bits
func WithBitFieldType(bitType string) Option {
	return func(signin ISignIn) {
		signin.setBitFieldType(bitType)
	}
}

func WithStartDate(startDate time.Time) Option {
	return func(signin ISignIn) {
		signin.setStartDate(startDate)
	}
}

func WithSignInterval(d time.Duration) Option {
	return func(signin ISignIn) {
		signin.setSignInterval(d)
	}
}

func WithSignInRedisKeyPrefix(prefix string) Option {
	return func(signin ISignIn) {
		signin.setRedisKeyPrefix(prefix)
	}
}

func WithRedisClient(addr, password, username string) Option {
	return func(signin ISignIn) {
		rdb := redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			Username: username,
			DB:       0,
		})
		err := signin.setRedisClient(rdb)
		if err != nil {
			panic(err)
		}
	}
}

type RedisClusterConfig struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
}

func WithRedisCluster(addrs []string, password string) Option {
	return func(signin ISignIn) {
		rdb := redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    addrs,
			Password: password,
		})

		err := signin.setRedisCluster(rdb)
		if err != nil {
			panic(err)
		}
	}
}

func WithRedisURL(url string) Option {
	return func(signin ISignIn) {
		opt, err := redis.ParseURL(url)
		if err != nil {
			panic(err)
		}
		rdb := redis.NewClient(opt)
		err = signin.setRedisClient(rdb)
		if err != nil {
			panic(err)
		}
	}
}
