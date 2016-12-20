package logic

import (
	"encoding/json"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/reechou/gorobot/wxweb"
)

func (self *WxHttpSrv) StartWx(rsp http.ResponseWriter, req *http.Request) (interface{}, error) {
	response := WxResponse{Code: WX_RESPONSE_OK}

	uuid := self.l.StartWx()
	response.Data = uuid

	return response, nil
}

func (self *WxHttpSrv) StartWxWithArgv(rsp http.ResponseWriter, req *http.Request) (interface{}, error) {
	request := &wxweb.StartWxArgv{}
	if err := json.NewDecoder(req.Body).Decode(request); err != nil {
		logrus.Errorf("StartWxWithArgv json decode error: %v", err)
		return nil, err
	}

	response := WxResponse{Code: WX_RESPONSE_OK}

	startRsp := self.l.StartWxWithArgv(request)
	response.Data = startRsp

	return response, nil
}

func (self *WxHttpSrv) Qrcode(rsp http.ResponseWriter, req *http.Request) (interface{}, error) {
	req.ParseForm()

	response := WxResponse{Code: WX_RESPONSE_OK}

	if len(req.Form["uuid"]) == 0 {
		response.Code = WX_RESPONSE_ERR
		response.Msg = "req cannot found uuid."
	} else {
		uuid := req.Form["uuid"][0]
		_, ok := self.l.wxs[uuid]
		if !ok {
			response.Code = WX_RESPONSE_ERR
			response.Msg = "cannot found this uuid."
		} else {
			http.ServeFile(rsp, req, uuid+".jpg")
		}
	}

	return response, nil
}

func (self *WxHttpSrv) InviteMemberStatus(rsp http.ResponseWriter, req *http.Request) (interface{}, error) {
	req.ParseForm()

	response := WxResponse{Code: WX_RESPONSE_OK}

	if len(req.Form["uuid"]) == 0 {
		response.Code = WX_RESPONSE_ERR
		response.Msg = "req cannot found uuid."
	} else {
		uuid := req.Form["uuid"][0]
		wx, ok := self.l.wxs[uuid]
		if !ok {
			response.Code = WX_RESPONSE_ERR
			response.Msg = "cannot found this uuid."
		} else {
			response.Data = wx.Contact.IfInviteMemberSuccess
		}
	}

	return response, nil
}
