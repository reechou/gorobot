package cache

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/reechou/gorobot/config"
)

func TestRedis(t *testing.T) {
	rc := NewRedisCache(&config.RedisInfo{Conninfo: "127.0.0.1:6379", DbNum: 1, Key: "WxRank", Password: ""})
	err := rc.StartAndGC()
	if err != nil {
		fmt.Println(err)
		return
	}
	type User struct {
		UserName string
		NickName string
	}
	u := User{
		UserName: "@ree",
		NickName: "Mr.Ree",
	}
	ub, _ := json.Marshal(&u)
	fmt.Println(string(ub))
	rc.Put("@ree", string(ub))
	info := rc.Get("@ree")
	var u2 User
	err = json.Unmarshal(info.([]byte), &u2)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(u2)
}

func TestRedisHSet(t *testing.T) {
	rc := NewRedisCache(&config.RedisInfo{Conninfo: "127.0.0.1:6379", DbNum: 1, Key: "WxRank", Password: ""})
	err := rc.StartAndGC()
	if err != nil {
		fmt.Println(err)
		return
	}
	has, err := rc.HSetNX("invite_@@ree", "@zhou", true)
	if err != nil {
		fmt.Println(err)
		return
	}
	if has {
		fmt.Println("hsetnx success")
	} else {
		fmt.Println("hsetnx has exist.")
	}
}

func TestRedisZSet(t *testing.T) {
	rc := NewRedisCache(&config.RedisInfo{Conninfo: "127.0.0.1:6379", DbNum: 1, Key: "WxRank", Password: ""})
	err := rc.StartAndGC()
	if err != nil {
		fmt.Println(err)
		return
	}
	rc.ZIncrby("@@ooxx", 10, "@ree")
	rc.ZIncrby("@@ooxx", 6, "@zhou")
	rc.ZIncrby("@@ooxx", 9, "@jin")
	list := rc.ZRevrange("@@ooxx", 0, 3)
	fmt.Println("", len(list))
	for _, v := range list {
		_v := v.([]byte)
		fmt.Println(string(_v))
	}
}

func TestRedisClear(t *testing.T) {
	rc := NewRedisCache(&config.RedisInfo{Conninfo: "127.0.0.1:6379", DbNum: 1, Key: "WxRank", Password: ""})
	err := rc.StartAndGC()
	if err != nil {
		fmt.Println(err)
		return
	}
	err = rc.ClearAll()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("redis clear all success.")
}
