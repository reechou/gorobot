package logic

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/mitchellh/mapstructure"
	"github.com/reechou/gorobot/config"
)

type WxHttpSrv struct {
	cfg     *config.Config
	httpSrv *HttpSrv
	l       *WxLogic
}

type HttpHandler func(rsp http.ResponseWriter, req *http.Request) (interface{}, error)

func NewWxHTTPServer(cfg *config.Config, l *WxLogic) *WxHttpSrv {
	srv := &WxHttpSrv{
		cfg:     cfg,
		l:       l,
		httpSrv: &HttpSrv{Host: cfg.Host, Routers: make(map[string]http.HandlerFunc)},
	}
	srv.registerHandlers()

	return srv
}

func (self *WxHttpSrv) Run() {
	logrus.Infof("wxweb http server starting...")
	self.httpSrv.Run()
}

func (self *WxHttpSrv) registerHandlers() {
	self.httpSrv.Route("/", self.Index)

	self.httpSrv.Route("/startwx", self.httpWrap(self.StartWx))
	self.httpSrv.Route("/startwx2", self.httpWrap(self.StartWxWithArgv))
	self.httpSrv.Route("/qrcode", self.httpWrap(self.Qrcode))
	self.httpSrv.Route("/invite_status", self.httpWrap(self.InviteMemberStatus))
}

func (self *WxHttpSrv) httpWrap(handler HttpHandler) func(rsp http.ResponseWriter, req *http.Request) {
	f := func(rsp http.ResponseWriter, req *http.Request) {
		logURL := req.URL.String()
		start := time.Now()
		defer func() {
			logrus.Debugf("[WxHttpSrv][httpWrap] http: request url[%s] use_time[%v]", logURL, time.Now().Sub(start))
		}()
		obj, err := handler(rsp, req)
		// check err
	HAS_ERR:
		if err != nil {
			logrus.Errorf("[WxHttpSrv][httpWrap] http: request url[%s] error: %v", logURL, err)
			code := 500
			errMsg := err.Error()
			if strings.Contains(errMsg, "Permission denied") || strings.Contains(errMsg, "ACL not found") {
				code = 403
			}
			rsp.WriteHeader(code)
			rsp.Write([]byte(errMsg))
			return
		}

		// return json object
		if obj != nil {
			var buf []byte
			buf, err = json.Marshal(obj)
			if err != nil {
				goto HAS_ERR
			}
			rsp.Header().Set("Content-Type", "application/json")
			rsp.Write(buf)
		}
	}
	return f
}

func (self *WxHttpSrv) Index(rsp http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		rsp.WriteHeader(404)
		return
	}
	rsp.Write([]byte("wx web service."))
}

func (self *WxHttpSrv) decodeBody(req *http.Request, out interface{}, cb func(interface{}) error) error {
	var raw interface{}
	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(&raw); err != nil {
		return err
	}

	if cb != nil {
		if err := cb(raw); err != nil {
			return err
		}
	}

	return mapstructure.Decode(raw, out)
}
