package logic

import (
	"github.com/reechou/gorobot/wxweb"
	"golang.org/x/net/context"
)

type ReceiveMsgInfo struct {
	msg    *wxweb.ReceiveMsgInfo
	ctx    context.Context
	cancel context.CancelFunc
}

type SendMsgInfo struct {
	WeChat   string
	ChatType string
	Name     string
	UserName string
	MsgType  string
	Msg      string
}

type CheckGroupChatInfo struct {
	Group            string
	LastChatInterval int
}
