package main

import (
	"github.com/reechou/gorobot/config"
	"github.com/reechou/gorobot/logic"
)

func main() {
	logic.NewWxLogic(config.NewConfig()).Run()
}
