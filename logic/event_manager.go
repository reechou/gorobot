package logic

import (
	"bufio"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/reechou/gorobot/config"
	"github.com/reechou/gorobot/wxweb"
	"golang.org/x/net/context"
)

type EventManager struct {
	wxm *WxManager
	cfg *config.Config

	msgChan chan *ReceiveMsgInfo
	filters map[string][]*EventFilter

	stop chan struct{}
}

func NewEventManager(wxm *WxManager, cfg *config.Config) *EventManager {
	em := &EventManager{
		wxm:     wxm,
		cfg:     cfg,
		msgChan: make(chan *ReceiveMsgInfo, EVENT_MSG_CHAN_LEN),
		filters: make(map[string][]*EventFilter),
		stop:    make(chan struct{}),
	}
	em.loadFile()
	go em.Run()

	return em
}

func (self *EventManager) Stop() {
	close(self.stop)
}

func (self *EventManager) loadFile() {
	logrus.Debugf("start load event file")
	f, err := os.Open(self.cfg.WxEventFile)
	if err != nil {
		logrus.Errorf("open file[%s] error: %v", self.cfg.WxEventFile, err)
		return
	}
	buf := bufio.NewReader(f)
	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return
			}
			logrus.Errorf("load event file[%s] error: %v", self.cfg.WxEventFile, err)
			return
		}
		argv := strings.Split(line, " ")
		if len(argv) == 0 {
			continue
		}
		switch argv[0] {
		case "filter":
			if len(argv) != 8 {
				logrus.Errorf("filter argv error: %s", line)
				continue
			}
			f := &EventFilter{
				wxm:      self.wxm,
				WeChat:   argv[1],
				Time:     argv[2],
				Event:    argv[3],
				Msg:      argv[4],
				From:     argv[5],
				FromType: argv[6],
				Do:       argv[7],
			}
			if f.Msg == EMPTY {
				f.Msg = ""
			}
			if f.From == EMPTY {
				f.From = ""
			}
			if f.FromType == EMPTY {
				f.FromType = ""
			}
			f.Init(self.stop)
			fv, ok := self.filters[f.WeChat]
			fv = append(fv, f)
			if !ok {
				self.filters[f.WeChat] = fv
			}
		case "timer":
		case "cron":
		}
	}
	logrus.Infof("load event file[%s] success.", self.cfg.WxEventFile)
	return
}

func (self *EventManager) Run() {
	logrus.Debugf("event manager start run.")
	logrus.Debugf("filters: %v", self.filters)
	for {
		select {
		case msg := <-self.msgChan:
			fs, ok := self.filters[msg.msg.WeChat]
			if ok {
				for _, v := range fs {
					//logrus.Debugf("find filter: %v", *v)
					select {
					case v.GetMsgChan() <- msg:
					case <-msg.ctx.Done():
						logrus.Errorf("receive msg into filter msg channal error: %v", msg.ctx.Err())
						continue
					}
				}
			}
			msg.cancel()
		case <-self.stop:
			return
		}
	}
}

func (self *EventManager) ReceiveMsg(msg *wxweb.ReceiveMsgInfo) {
	//logrus.Debugf("event manager reveive msg: %v", msg)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	select {
	case self.msgChan <- &ReceiveMsgInfo{msg: msg, ctx: ctx, cancel: cancel}:
	case <-ctx.Done():
		logrus.Errorf("receive msg into msg channal error: %v", ctx.Err())
		return
	case <-self.stop:
		return
	}
}
