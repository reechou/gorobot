package wxweb

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/reechou/gorobot/cache"
)

const (
	MSG_LEN = 100
)

type UserFriend struct {
	Alias       string
	City        string
	ContactFlag int
	NickName    string
	Sex         int
	UserName    string
}

type GroupUserInfo struct {
	DisplayName string
	NickName    string
	UserName    string
}
type MsgInfo struct {
	MsgID    int
	WXMsgId  string
	NickName string
	UserName string
	Content  string
}
type MsgOffset struct {
	SliceStart int
	SliceEnd   int
	MsgIDStart int
	MsgIDEnd   int
}
type UserGroup struct {
	sync.Mutex
	ContactFlag int
	NickName    string
	UserName    string
	MemberList  map[string]*GroupUserInfo

	rankRedis *cache.RedisCache

	offset *MsgOffset
	msgs   []*MsgInfo
	msgId  int
}

func NewUserGroup(contactFlag int, nickName, userName string, rankRedis *cache.RedisCache) *UserGroup {
	logrus.Debugf("新增群组: %s", nickName)
	logrus.Debugf("群组user name: %s", userName)
	return &UserGroup{
		ContactFlag: contactFlag,
		NickName:    nickName,
		UserName:    userName,
		MemberList:  make(map[string]*GroupUserInfo),
		offset: &MsgOffset{
			SliceStart: -1,
			SliceEnd:   -1,
			MsgIDStart: -1,
			MsgIDEnd:   -1,
		},
		msgs:      make([]*MsgInfo, MSG_LEN),
		rankRedis: rankRedis,
	}
}

func (self *UserGroup) AppendInviteMsg(msg *MsgInfo) {
	if self.NickName == "" {
		logrus.Errorf("this group has no nick name.")
		return
	}
	invite := strings.Replace(msg.Content, "\"", "", -1)
	invite = strings.Replace(invite, "邀请", ",", -1)
	invite = strings.Replace(invite, "加入了群聊", "", -1)
	users := strings.Split(invite, ",")
	if len(users) != 2 {
		logrus.Errorf("parse invite content[%s] error.", msg.Content)
		return
	}
	inviteUsers := strings.Split(users[1], "、")
	for _, v := range inviteUsers {
		//has, err := self.rankRedis.HSetNX("invite_"+self.NickName, v, true)
		has, err := self.rankRedis.HSetNX("invite_wx_rank", v, true)
		if err != nil {
			logrus.Errorf("hsetnx invite[%s] error: %v", v, err)
			continue
		}
		if !has {
			logrus.Debugf("has invited[%s] this man.", v)
			continue
		}
		//self.rankRedis.ZIncrby(self.NickName, 1, users[0])
		self.rankRedis.ZIncrby("wx_rank", 1, users[0])
	}
}

func (self *UserGroup) GetInviteRank() string {
	//list := self.rankRedis.ZRevrange(self.NickName, 0, 10)
	list := self.rankRedis.ZRevrange("wx_rank", 0, 10)
	var usersRankInfo string
	var userRank string
	usersRankInfo += "邀请排行榜:\n"
	for i := 0; i < len(list); i++ {
		//if i % 2 == 0 {
		//	userRank += "@"
		//}
		userRank += string(list[i].([]byte))
		if i%2 == 0 {
			userRank += ": "
		} else {
			userRank += "人\n"
			usersRankInfo += userRank
			userRank = ""
		}
	}
	return usersRankInfo
}

func (self *UserGroup) AppendMsg(msg *MsgInfo) {
	self.Lock()
	defer self.Unlock()

	msg.MsgID = self.msgId

	if self.offset.SliceStart == -1 && self.offset.SliceEnd == -1 && self.offset.MsgIDStart == -1 && self.offset.MsgIDEnd == -1 {
		self.msgs[0] = msg
		self.offset.SliceStart = 0
		self.offset.SliceEnd = 1
		self.offset.MsgIDStart = msg.MsgID
		self.offset.MsgIDEnd = msg.MsgID
	} else {
		self.offset.MsgIDEnd = msg.MsgID
		self.msgs[self.offset.SliceEnd] = msg
		if self.offset.SliceEnd-self.offset.SliceStart == -1 ||
			self.offset.SliceEnd-self.offset.SliceStart == (MSG_LEN-1) ||
			self.offset.SliceEnd-self.offset.SliceStart == (1-MSG_LEN) {
			self.offset.SliceStart = (self.offset.SliceStart + 1) % MSG_LEN
			self.offset.MsgIDStart = self.msgs[self.offset.SliceStart].MsgID
		}
		self.offset.SliceEnd = (self.offset.SliceEnd + 1) % MSG_LEN
	}
	//logrus.Debugf("group[%s] add msg[%v] offset[%v]", self.UserName, msg, self.offset)

	self.msgId++
}

func (self *UserGroup) GetMsgList(msgId int) []*MsgInfo {
	if msgId >= self.offset.MsgIDEnd {
		return nil
	}
	jump := 0
	if msgId > self.offset.MsgIDStart {
		jump = msgId - self.offset.MsgIDStart
	}
	start := self.offset.SliceStart
	for i := 0; i < jump; i++ {
		start = (start + 1) % MSG_LEN
	}
	if start > self.offset.SliceEnd {
		return append(self.msgs[start:], self.msgs[:self.offset.SliceEnd]...)
	} else {
		return self.msgs[start:self.offset.SliceEnd]
	}

	return nil
}

type UserContact struct {
	wx         *WxWeb
	Friends     map[string]*UserFriend
	NickFriends map[string]*UserFriend
	Groups      map[string]*UserGroup
	NickGroups  map[string]*UserGroup
	
	IfInviteMemberSuccess bool
}

func NewUserContact(wx *WxWeb) *UserContact {
	return &UserContact{
		wx:          wx,
		Friends:     make(map[string]*UserFriend),
		NickFriends: make(map[string]*UserFriend),
		Groups:      make(map[string]*UserGroup),
		NickGroups:  make(map[string]*UserGroup),
	}
}

func (self *UserContact) InviteMembers() {
	if self.wx.cfg.IfInvite {
		if self.wx.MyNickName == "小健健" {
			logrus.Debugf("小健健 not exec invite members.")
			return 
		}
		var groupUserName string
		for _, v := range self.Groups {
			if strings.Contains(v.NickName, "网购特卖") {
				groupUserName = v.UserName
				break
			}
		}
		if groupUserName != "" {
			inviteNum := 0
			var memberList []string
			for _, v := range self.Friends {
				_, ok := self.wx.SpecialUsers[v.UserName]
				if ok {
					continue
				}
				if v.NickName == "你好杭州" {
					continue
				}
				memberList = append(memberList, v.UserName)
				if len(memberList) >= 10 {
					data, ok := self.wx.WebwxupdatechatroomInvitemember(groupUserName, memberList)
					if ok {
						dataJson := JsonDecode(data)
						if dataJson != nil {
							dataMap := dataJson.(map[string]interface{})
							retCode := dataMap["BaseResponse"].(map[string]interface{})["Ret"].(int)
							if retCode == -34 {
								logrus.Errorf("wx[%s] invite member get -34 error, maybe sleep some minute", self.wx.MyNickName)
								time.Sleep(30 * time.Minute)
							} else {
								for _, v2 := range memberList {
									self.wx.Webwxsendmsg(self.wx.cfg.InviteMsg, v2)
									time.Sleep(time.Second)
								}
							}
						}
					}
					inviteNum += 10
					// clear
					memberList = nil
					time.Sleep(8 * time.Second)
					if inviteNum >= 200 {
						time.Sleep(2 * time.Minute)
						inviteNum = 0
					}
				}
			}
			if memberList != nil {
				time.Sleep(8 * time.Second)
				self.wx.WebwxupdatechatroomInvitemember(groupUserName, memberList)
				for _, v2 := range memberList {
					self.wx.Webwxsendmsg(self.wx.cfg.InviteMsg, v2)
				}
				// clear
				memberList = nil
			}
		} else {
			logrus.Errorf("check group not found.")
		}
		self.IfInviteMemberSuccess = true
		logrus.Infof("[%s] invite members success.", self.wx.MyNickName)
	}
}

func (self *UserContact) PrintGroupInfo() {
	allGroupNum := 0
	cfNum := 0
	members := make(map[string]int)
	for _, v := range self.Groups {
		fmt.Println("群:", v.NickName)
		if !strings.Contains(v.NickName, "网购特卖") {
			continue
		}
		allGroupNum++
		fmt.Println("\t群:", v.NickName)
		for _, v2 := range v.MemberList {
			_, ok := members[v2.UserName]
			if ok {
				cfNum++
				continue
			}
			members[v2.UserName] = 1
		}
	}
	fmt.Println("[*] REAL-群组数:", allGroupNum)
	fmt.Println("[*] REAL-去重群成员总数:", len(members), cfNum)
}
