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
			if msg.Name == "$from" {
				msg.Name = rMsg.msg.FromNickName
			}
			msg.UserName = rMsg.msg.FromUserName
			msgResult := self.changeString(msg, rMsg)
			self.wxm.SendMsg(msg, msgResult)
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
