package logic

import (
	"fmt"
	"sync"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/reechou/gorobot/wxweb"
)

type WxManager struct {
	sync.Mutex
	wxs map[string]*wxweb.WxWeb
}

func NewWxManager() *WxManager {
	wm := &WxManager{
		wxs: make(map[string]*wxweb.WxWeb),
	}
	return wm
}

func (self *WxManager) RegisterWx(wx *wxweb.WxWeb) {
	self.Lock()
	defer self.Unlock()

	nickName, ok := wx.User["NickName"]
	if ok {
		nick := nickName.(string)
		self.wxs[nick] = wx
		logrus.Infof("wx manager register wx[%s] success.", nick)
	}
}

func (self *WxManager) UnregisterWx(wx *wxweb.WxWeb) {
	self.Lock()
	defer self.Unlock()

	nickName, ok := wx.User["NickName"]
	if ok {
		nick := nickName.(string)
		_, ok := self.wxs[nick]
		if ok {
			delete(self.wxs, nick)
			logrus.Infof("wx manager unregister wx[%s] success.", nick)
		}
	}
}

func (self *WxManager) SendMsg(msg *SendMsgInfo, msgStr string) {
	//logrus.Debugf("send msg[%v] msgStr: %s", msg, msgStr)
	wx := self.wxs[msg.WeChat]
	if wx == nil {
		logrus.Errorf("unknown this wechat[%s].", msg.WeChat)
		return
	}
	switch msg.ChatType {
	case CHAT_TYPE_PEOPLE:
		var userName string
		uf := wx.Contact.NickFriends[msg.Name]
		if uf == nil {
			logrus.Errorf("unkown this friend[%s]", msg.Name)
			return
		}
		userName = uf.UserName
		if msg.MsgType == MSG_TYPE_TEXT {
			wx.Webwxsendmsg(msgStr, userName)
		} else if msg.MsgType == MSG_TYPE_IMG {
			mediaId, ok := wx.Webwxuploadmedia(userName, msg.Msg)
			if ok {
				wx.Webwxsendmsgimg(userName, mediaId)
			}
		}
	case CHAT_TYPE_GROUP:
		var userName string
		group := wx.Contact.NickGroups[msg.Name]
		if group == nil {
			logrus.Errorf("unkown this group[%s]", msg.Name)
			return
		}
		userName = group.UserName
		if msg.MsgType == MSG_TYPE_TEXT {
			wx.Webwxsendmsg(msgStr, userName)
		}
	}
}

func (self *WxManager) VerifyUser(msg *wxweb.ReceiveMsgInfo) bool {
	wx := self.wxs[msg.WeChat]
	if wx == nil {
		logrus.Errorf("unknown this wechat[%s].", msg.WeChat)
		return false
	}
	ok := wx.Webwxverifyuser(msg.Ticket, msg.FromUserName)
	if ok {
		logrus.Infof("verigy user[%s] success.", msg.FromNickName)
	} else {
		logrus.Infof("verigy user[%s] error.", msg.FromNickName)
	}
	return ok
}

func (self *WxManager) CheckGroup() {
	
}

func (self *WxManager) StateGroupNum(wechat, g string) string {
	wx := self.wxs[wechat]
	if wx == nil {
		logrus.Errorf("unknown this wechat[%s].", wechat)
		return ""
	}
	g = strings.Replace(g, "\n", "", -1)
	result := "【%s | 群】群总数-%d 去重群成员数-%d 重复成员数-%d"
	allGroupNum := 0
	cfNum := 0
	members := make(map[string]int)
	for _, v := range wx.Contact.Groups {
		if !ExecCheckFunc(g, v.NickName) {
			continue
		}
		allGroupNum++
		for _, v2 := range v.MemberList {
			_, ok := members[v2.UserName]
			if ok {
				cfNum++
				continue
			}
			members[v2.UserName] = 1
		}
	}
	g = ExecGetArgvFunc(g)
	return fmt.Sprintf(result, g, allGroupNum, len(members), cfNum)
}

func (self *WxManager) CheckGroupChat(info *CheckGroupChatInfo) {

}
