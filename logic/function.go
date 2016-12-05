package logic

import "fmt"

type FunctionEvent struct {
	wxm      *WxManager
	Function string
	Argv     interface{}
}

func (self *FunctionEvent) function() {
	switch self.Function {
	case FUNC_EVENT_CHECK_GROUP_CHAT:
		argv, ok := self.Argv.(*CheckGroupChatInfo)
		if ok {
			fmt.Println(argv)
		}
	}
}
