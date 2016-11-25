package logic

import (
	"strings"

	"github.com/Sirupsen/logrus"
)

type EventFilter struct {
	wxm      *WxManager
	WeChat   string
	Time     string
	Event    string
	Msg      string
	From     string
	FromType string
	Do       string
	DoEvent  []DoEvent

	msgChan chan *ReceiveMsgInfo
	stop    chan struct{}
}

func (self *EventFilter) GetMsgChan() chan *ReceiveMsgInfo {
	return self.msgChan
}

func (self *EventFilter) Init(stop chan struct{}) {
	self.msgChan = make(chan *ReceiveMsgInfo, EVENT_MSG_CHAN_LEN)
	self.stop = stop

	doList := strings.Split(self.Do, ",")
	for _, v := range doList {
		doDetail := strings.Split(v, "^")
		if len(doDetail) == 0 {
			continue
		}
		switch doDetail[0] {
		case DO_EVENT_SENDMSG:
			if len(doDetail) != 4 {
				continue
			}
			msg := &SendMsgInfo{
				WeChat:   self.WeChat,
				ChatType: doDetail[1],
				Name:     doDetail[2],
			}
			msgInfo := strings.Split(doDetail[3], ">")
			if len(msgInfo) != 2 {
				continue
			}
			msg.MsgType = msgInfo[0]
			msg.Msg = msgInfo[1]
			self.DoEvent = append(self.DoEvent, DoEvent{wxm: self.wxm, Type: DO_EVENT_SENDMSG, DoMsg: msg})
		case DO_EVENT_VERIFY_USER:
			self.DoEvent = append(self.DoEvent, DoEvent{wxm: self.wxm, Type: DO_EVENT_VERIFY_USER})
		}
	}
	go self.Run()
}

func (self *EventFilter) Run() {
	logrus.Debugf("filter wechat[%s] Event[%s] Do[%s] start run.", self.WeChat, self.Event, self.Do)
	for {
		select {
		case msg := <-self.msgChan:
			logrus.Debugf("filter msg: %v", msg)
			if self.Event != msg.msg.ReceiveEvent {
				continue
			}
			if self.FromType != "" {
				if self.FromType != msg.msg.FromType {
					continue
				}
			}
			if self.Msg != "" {
				if self.Msg != msg.msg.Msg {
					continue
				}
			}
			if self.From != "" {
				if self.From != msg.msg.FromNickName {
					continue
				}
			}
			for _, v := range self.DoEvent {
				v.Do(msg)
			}
		case <-self.stop:
			return
		}
	}
}
