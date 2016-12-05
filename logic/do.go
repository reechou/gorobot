package logic

import (
	"strings"

	"github.com/Sirupsen/logrus"
)

type DoEvent struct {
	wxm   *WxManager
	Type  string
	DoMsg interface{}
}

func (self *DoEvent) Do(rMsg *ReceiveMsgInfo) {
	switch self.Type {
	case DO_EVENT_SENDMSG:
		msg, ok := self.DoMsg.(*SendMsgInfo)
		if ok {
			msgCopy := &SendMsgInfo{
				WeChat:   msg.WeChat,
				ChatType: msg.ChatType,
				Name:     msg.Name,
				UserName: msg.UserName,
				MsgType:  msg.MsgType,
				Msg:      msg.Msg,
			}
			if msgCopy.Name == "$from" {
				msgCopy.Name = rMsg.msg.FromNickName
			}
			msgCopy.UserName = rMsg.msg.FromUserName
			msgResult := self.changeString(msgCopy, rMsg)
			self.wxm.SendMsg(msgCopy, msgResult)
		} else {
			logrus.Errorf("translate to SendMsgInfo error.")
		}
	case DO_EVENT_VERIFY_USER:
		self.wxm.VerifyUser(rMsg.msg)
	}
}

func (self *DoEvent) changeString(sm *SendMsgInfo, rm *ReceiveMsgInfo) string {
	result := sm.Msg
	if strings.Contains(sm.Msg, FROMGROUP) {
		result = strings.Replace(result, FROMGROUP, rm.msg.FromGroupName, -1)
	}
	if strings.Contains(sm.Msg, FROMUSER) {
		result = strings.Replace(result, FROMUSER, rm.msg.FromNickName, -1)
	}
	if strings.Contains(sm.Msg, FROMMSG) {
		result = strings.Replace(result, FROMMSG, rm.msg.Msg, -1)
	}

	return result
}
