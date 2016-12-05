package config

import (
	"flag"
	"fmt"
	"os"

	"github.com/go-ini/ini"
)

type DBInfo struct {
	User   string
	Pass   string
	Host   string
	DBName string
}

type RedisInfo struct {
	Conninfo string
	DbNum    int
	Key      string
	Password string
}

type Config struct {
	Debug   bool
	Path    string
	Host    string
	Version string

	TestNickName string
	UploadFile   string
	IfInvite     bool
	InviteMsg    string

	WxEventFile string

	MemberRedis  RedisInfo
	RankRedis    RedisInfo
	SessionRedis RedisInfo
}

func NewConfig() *Config {
	c := new(Config)
	initFlag(c)

	if c.Path == "" {
		fmt.Println("wxrobot must run with config file, please check.")
		os.Exit(0)
	}

	cfg, err := ini.Load(c.Path)
	if err != nil {
		fmt.Printf("ini[%s] load error: %v\n", c.Path, err)
		os.Exit(1)
	}
	cfg.BlockMode = false
	err = cfg.MapTo(c)
	if err != nil {
		fmt.Printf("config MapTo error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(c)

	return c
}

func initFlag(c *Config) {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	v := fs.Bool("v", false, "Print version and exit")
	fs.StringVar(&c.Path, "c", "", "wxrobot config file.")

	fs.Parse(os.Args[1:])
	fs.Usage = func() {
		fmt.Println("Usage: " + os.Args[0] + " -c api.ini")
		fmt.Printf("\nglobal flags:\n")
		fs.PrintDefaults()
	}

	if *v {
		fmt.Println("version: 0.0.1")
		os.Exit(0)
	}
}
