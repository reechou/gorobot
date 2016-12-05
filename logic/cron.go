package logic

import (
	"strings"
	
	"github.com/robfig/cron"
	"github.com/Sirupsen/logrus"
	"github.com/reechou/gorobot/wxweb"
)

type EventCron struct {
	wxm     *WxManager
	WeChat  string
	Time    string
	Do      string
	DoEvent []DoEvent
	stop    chan struct{}
}

func (self *EventCron) Init(stop chan struct{}) {
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
			msgInfo := strings.Split(doDetail[3], ">>>")
			if len(msgInfo) != 2 {
				continue
			}
			msg.MsgType = msgInfo[0]
			msg.Msg = msgInfo[1]
			msg.Msg = strings.Replace(msg.Msg, "<br/>", "\n", -1)
			self.DoEvent = append(self.DoEvent, DoEvent{wxm: self.wxm, Type: DO_EVENT_SENDMSG, DoMsg: msg})
		case DO_EVENT_VERIFY_USER:
			self.DoEvent = append(self.DoEvent, DoEvent{wxm: self.wxm, Type: DO_EVENT_VERIFY_USER})
		}
	}
	go self.Run()
}

func (self *EventCron) Run() {
	logrus.Debugf("cron wechat[%s] time[%s] Do[%s] start run.", self.WeChat, self.Time, self.Do)
	self.Time = strings.Replace(self.Time, ",", ", ", -1)
	c := cron.New()
	c.AddFunc(self.Time, self.cronRun)
	c.Start()
	
	select {
	case <-self.stop:
		c.Stop()
		return
	}
}

func (self *EventCron) cronRun() {
	logrus.Debugf("in cron %s %s", self.WeChat, self.Time)
	for _, v := range self.DoEvent {
		v.Do(&ReceiveMsgInfo{msg: &wxweb.ReceiveMsgInfo{}})
	}
}
