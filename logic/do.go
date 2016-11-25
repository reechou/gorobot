package logic

import (
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
			if msg.Name == "$from" {
				msg.Name = rMsg.msg.FromNickName
			}
			msg.UserName = rMsg.msg.FromUserName
			self.wxm.SendMsg(msg)
		} else {
			logrus.Errorf("translate to SendMsgInfo error.")
		}
	case DO_EVENT_VERIFY_USER:
		self.wxm.VerifyUser(rMsg.msg)
	}
}
