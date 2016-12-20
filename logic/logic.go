package logic

import (
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/reechou/gorobot/cache"
	"github.com/reechou/gorobot/config"
	"github.com/reechou/gorobot/wxweb"
)

type WxLogic struct {
	sync.Mutex

	cfg *config.Config

	wxs      map[string]*wxweb.WxWeb
	wxSrv    *WxHttpSrv
	wxMgr    *WxManager
	eventMgr *EventManager

	memberRedis  *cache.RedisCache
	rankRedis    *cache.RedisCache
	sessionRedis *cache.RedisCache

	stop chan struct{}
	done chan struct{}
}

func NewWxLogic(cfg *config.Config) *WxLogic {
	if cfg.Debug {
		EnableDebug()
	}

	l := &WxLogic{
		cfg:          cfg,
		memberRedis:  cache.NewRedisCache(&cfg.MemberRedis),
		rankRedis:    cache.NewRedisCache(&cfg.RankRedis),
		sessionRedis: cache.NewRedisCache(&cfg.SessionRedis),
		wxs:          make(map[string]*wxweb.WxWeb),
		stop:         make(chan struct{}),
		done:         make(chan struct{}),
	}
	l.wxSrv = NewWxHTTPServer(cfg, l)
	l.wxMgr = NewWxManager()
	l.eventMgr = NewEventManager(l.wxMgr, cfg)

	err := l.memberRedis.StartAndGC()
	if err != nil {
		panic(err)
	}
	err = l.rankRedis.StartAndGC()
	if err != nil {
		panic(err)
	}
	err = l.sessionRedis.StartAndGC()
	if err != nil {
		panic(err)
	}

	go l.runCheck()

	return l
}

func (self *WxLogic) Stop() {
	close(self.stop)
	<-self.done
}

func (self *WxLogic) Run() {
	self.wxSrv.Run()
}

func (self *WxLogic) StartWx() string {
	wx := wxweb.NewWxWeb(self.cfg, self.memberRedis, self.rankRedis, self.sessionRedis, self)
	wx.Start()
	go wx.Run()
	self.Lock()
	self.wxs[wx.UUID()] = wx
	self.Unlock()

	return wx.UUID()
}

func (self *WxLogic) StartWxWithArgv(argv *wxweb.StartWxArgv) *StartWxRsp {
	wx := wxweb.NewWxWebWithArgv(self.cfg, self.memberRedis, self.rankRedis, self.sessionRedis, self, argv)
	wx.Start()
	go wx.Run()
	self.Lock()
	self.wxs[wx.UUID()] = wx
	self.Unlock()

	rsp := &StartWxRsp{
		UUID:      wx.UUID(),
		QrcodeUrl: wx.QRCODE(),
	}

	return rsp
}

func (self *WxLogic) Login(uuid string) {
	logrus.Infof("uuid[%s] login success.", uuid)
	wx, ok := self.wxs[uuid]
	if ok {
		self.wxMgr.RegisterWx(wx)
	} else {
		logrus.Errorf("cannot found wx this uuid[%s]", uuid)
	}
}

func (self *WxLogic) Logout(uuid string) {
	wx, ok := self.wxs[uuid]
	if ok {
		self.wxMgr.UnregisterWx(wx)
		wx.Clear()
		delete(self.wxs, uuid)
		logrus.Infof("logic wx uuid[%s] logout succsss.", uuid)
	} else {
		logrus.Errorf("cannot found wx this uuid[%s]", uuid)
	}
}

func (self *WxLogic) ReceiveMsg(msg *wxweb.ReceiveMsgInfo) {
	self.eventMgr.ReceiveMsg(msg)
}

func (self *WxLogic) runCheck() {
	logrus.Debugf("start run logic check wx status.")
	for {
		select {
		case <-time.After(60 * time.Second):
			self.check()
		case <-self.stop:
			close(self.done)
			return
		}
	}
}

func (self *WxLogic) check() {
	self.Lock()
	defer self.Unlock()

	for k, v := range self.wxs {
		if !v.IfLogin() {
			if time.Now().Unix()-v.StartTime() > WAIT_LOGIN_MAX_TIME {
				v.Stop()
				logrus.Infof("wx uuid[%s] login timeout, stop wx loop", k)
				delete(self.wxs, k)
				continue
			}
		}
	}
}

func EnableDebug() {
	logrus.SetLevel(logrus.DebugLevel)
}
