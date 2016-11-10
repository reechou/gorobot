//web weixin client
package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/reechou/gorobot/config"
	"github.com/reechou/gorobot/cache"
)

const debug = false

func debugPrint(content interface{}) {
	if debug == true {
		fmt.Println(content)
	}
}

type wxweb struct {
	uuid           string
	baseUri        string
	redirectUri    string
	uin            string
	sid            string
	skey           string
	passTicket     string
	deviceId       string
	SyncKey        map[string]interface{}
	synckey        string
	User           map[string]interface{}
	BaseRequest    map[string]interface{}
	syncHost       string
	httpClient     *http.Client
	cookies        []*http.Cookie
	ifTestSyncOK   bool
	ifChangeCookie bool
	SpecialUsers   []string
	lastCheckTs    int64
	
	cfg *config.Config
	memberRedis *cache.RedisCache
	rankRedis *cache.RedisCache

	contact *UserContact
}

func NewWxWeb(cfg *config.Config) *wxweb {
	wx := &wxweb{
		cfg: cfg,
		memberRedis: cache.NewRedisCache(&cfg.MemberRedis),
		rankRedis: cache.NewRedisCache(&cfg.RankRedis),
	}
	
	err := wx.memberRedis.StartAndGC()
	if err != nil {
		panic(err)
	}
	err = wx.rankRedis.StartAndGC()
	if err != nil {
		panic(err)
	}
	
	return wx
}

func (self *wxweb) getUuid(args ...interface{}) bool {
	urlstr := "https://login.weixin.qq.com/jslogin"
	urlstr += "?appid=wx782c26e4c19acffb&fun=new&lang=zh_CN&_=" + self._unixStr()
	data, _ := self._get(urlstr, false)
	re := regexp.MustCompile(`"([\S]+)"`)
	find := re.FindStringSubmatch(data)
	if len(find) > 1 {
		self.uuid = find[1]
		return true
	} else {
		return false
	}
}

func (self *wxweb) _run(desc string, f func(...interface{}) bool, args ...interface{}) {
	start := time.Now().UnixNano()
	logrus.Info(desc)
	var result bool
	if len(args) > 1 {
		result = f(args)
	} else if len(args) == 1 {
		result = f(args[0])
	} else {
		result = f()
	}
	useTime := fmt.Sprintf("%.5f", (float64(time.Now().UnixNano()-start) / 1000000000))
	if result {
		logrus.Infof("\t成功,用时 %s 秒", useTime)
	} else {
		logrus.Errorf("\t失败\n[*] 退出程序")
		os.Exit(1)
	}
}

func (self *wxweb) _post(urlstr string, params map[string]interface{}, jsonFmt bool) (string, error) {
	var err error
	var resp *http.Response
	if jsonFmt == true {
		jsonPost := JsonEncode(params)
		debugPrint(jsonPost)
		requestBody := bytes.NewBuffer([]byte(jsonPost))
		request, err := http.NewRequest("POST", urlstr, requestBody)
		if err != nil {
			return "", err
		}
		request.Header.Set("Content-Type", "application/json;charset=utf-8")
		request.Header.Add("Referer", "https://wx.qq.com/")
		request.Header.Add("User-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/47.0.2526.111 Safari/537.36")
		if self.cookies != nil {
			for _, v := range self.cookies {
				request.AddCookie(v)
			}
		}
		resp, err = self.httpClient.Do(request)
	} else {
		v := url.Values{}
		for key, value := range params {
			v.Add(key, value.(string))
		}
		resp, err = self.httpClient.PostForm(urlstr, v)
	}

	if err != nil || resp == nil {
		fmt.Println(err)
		return "", err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return "", err
	} else {
		defer resp.Body.Close()
	}
	return string(body), nil
}

func (self *wxweb) _get(urlstr string, jsonFmt bool) (string, error) {
	var err error
	res := ""
	request, _ := http.NewRequest("GET", urlstr, nil)
	request.Header.Add("Referer", "https://wx.qq.com/")
	request.Header.Add("User-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/47.0.2526.111 Safari/537.36")
	if self.cookies != nil {
		for _, v := range self.cookies {
			request.AddCookie(v)
		}
	}
	resp, err := self.httpClient.Do(request)
	if err != nil {
		return res, err
	}
	if resp.Cookies() != nil && len(resp.Cookies()) > 0 {
		self.cookies = resp.Cookies()
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return res, err
	}
	return string(body), nil
}

func (self *wxweb) _unixStr() string {
	return strconv.Itoa(int(time.Now().UnixNano() / 1000000))
}

func (self *wxweb) genQRcode(args ...interface{}) bool {
	urlstr := "https://login.weixin.qq.com/qrcode/" + self.uuid
	urlstr += "?t=webwx"
	urlstr += "&_=" + self._unixStr()
	path := "qrcode.jpg"
	out, err := os.Create(path)
	resp, err := self._get(urlstr, false)
	_, err = io.Copy(out, bytes.NewReader([]byte(resp)))
	if err != nil {
		return false
	} else {
		if runtime.GOOS == "darwin" {
			exec.Command("open", path).Run()
		} else {
			go func() {
				fmt.Println("please open on web broswer ip:8889/qrcode")
				http.HandleFunc("/qrcode", func(w http.ResponseWriter, req *http.Request) {
					http.ServeFile(w, req, "qrcode.jpg")
					return
				})
				http.ListenAndServe(":8889", nil)
			}()
		}
		return true
	}
}

func (self *wxweb) waitForLogin(tip int) bool {
	time.Sleep(time.Duration(tip) * time.Second)
	url := "https://login.weixin.qq.com/cgi-bin/mmwebwx-bin/login"
	url += "?tip=" + strconv.Itoa(tip) + "&uuid=" + self.uuid + "&_=" + self._unixStr()
	data, _ := self._get(url, false)
	re := regexp.MustCompile(`window.code=(\d+);`)
	find := re.FindStringSubmatch(data)
	if len(find) > 1 {
		code := find[1]
		if code == "201" {
			return true
		} else if code == "200" {
			re := regexp.MustCompile(`window.redirect_uri="(\S+?)";`)
			find := re.FindStringSubmatch(data)
			if len(find) > 1 {
				r_uri := find[1] + "&fun=new"
				self.redirectUri = r_uri
				re = regexp.MustCompile(`/`)
				finded := re.FindAllStringIndex(r_uri, -1)
				self.baseUri = r_uri[:finded[len(finded)-1][0]]
				return true
			}
			return false
		} else if code == "408" {
			logrus.Error("[登陆超时]")
		} else {
			logrus.Error("[登陆异常]")
		}
	}
	return false
}

func (self *wxweb) login(args ...interface{}) bool {
	data, _ := self._get(self.redirectUri, false)
	type Result struct {
		Skey       string `xml:"skey"`
		Wxsid      string `xml:"wxsid"`
		Wxuin      string `xml:"wxuin"`
		PassTicket string `xml:"pass_ticket"`
	}
	v := Result{}
	err := xml.Unmarshal([]byte(data), &v)
	if err != nil {
		fmt.Printf("error: %v", err)
		return false
	}
	self.skey = v.Skey
	self.sid = v.Wxsid
	self.uin = v.Wxuin
	self.passTicket = v.PassTicket
	self.BaseRequest = make(map[string]interface{})
	self.BaseRequest["Uin"], _ = strconv.Atoi(v.Wxuin)
	self.BaseRequest["Sid"] = v.Wxsid
	self.BaseRequest["Skey"] = v.Skey
	self.BaseRequest["DeviceID"] = self.deviceId
	return true
}

func (self *wxweb) webwxinit(args ...interface{}) bool {
	url := fmt.Sprintf("%s/webwxinit?passTicket=%s&skey=%s&r=%s", self.baseUri, self.passTicket, self.skey, self._unixStr())
	params := make(map[string]interface{})
	params["BaseRequest"] = self.BaseRequest
	res, err := self._post(url, params, true)
	if err != nil {
		return false
	}
	//log
	ioutil.WriteFile("tmp.txt", []byte(res), 777)
	//log

	data := JsonDecode(res).(map[string]interface{})
	self.User = data["User"].(map[string]interface{})
	self.SyncKey = data["SyncKey"].(map[string]interface{})
	self._setsynckey()

	retCode := data["BaseResponse"].(map[string]interface{})["Ret"].(int)
	if retCode != WX_RET_SUCCESS {
		return false
	}
	chatSet := data["ChatSet"].(string)
	chats := strings.Split(chatSet, ",")
	for _, v := range chats {
		if strings.HasPrefix(v, GROUP_PREFIX) {
			ug := NewUserGroup(0, "", v, self.rankRedis)
			self.contact.Groups[v] = ug
		}
	}
	logrus.Debugf("webwxinit get group num: %d", len(self.contact.Groups))
	//contactList := data["ContactList"].([]interface{})
	//for _, v := range contactList {
	//	contact := v.(map[string]interface{})
	//	userName := contact["UserName"].(string)
	//	contactFlag := contact["ContactFlag"].(int)
	//	nickName := contact["NickName"].(string)
	//	if strings.HasPrefix(userName, GROUP_PREFIX) {
	//		ug := &UserGroup{
	//			ContactFlag: contactFlag,
	//			NickName:    nickName,
	//			UserName:    userName,
	//			MemberList:  make(map[string]*GroupUserInfo),
	//		}
	//		self.contact.Groups[userName] = ug
	//	}
	//}

	return true
}

func (self *wxweb) _setsynckey() {
	keys := []string{}
	for _, keyVal := range self.SyncKey["List"].([]interface{}) {
		key := strconv.Itoa(int(keyVal.(map[string]interface{})["Key"].(int)))
		value := strconv.Itoa(int(keyVal.(map[string]interface{})["Val"].(int)))
		keys = append(keys, key+"_"+value)
	}
	self.synckey = strings.Join(keys, "|")
	debugPrint(self.synckey)
}

func (self *wxweb) synccheck() (string, string) {
	if self.ifTestSyncOK {
		if !self.ifChangeCookie {
			for _, v := range self.cookies {
				if v.Name == "wxloadtime" {
					v.Value = v.Value + "_expired"
					break
				}
			}
			self.ifChangeCookie = true
		}
	}
	urlstr := fmt.Sprintf("https://%s/cgi-bin/mmwebwx-bin/synccheck", self.syncHost)
	v := url.Values{}
	v.Add("r", self._unixStr())
	v.Add("sid", self.sid)
	v.Add("uin", self.uin)
	v.Add("skey", self.skey)
	v.Add("deviceid", self.deviceId)
	v.Add("synckey", self.synckey)
	v.Add("_", self._unixStr())
	urlstr = urlstr + "?" + v.Encode()
	data, _ := self._get(urlstr, false)
	if data == "" {
		return "9999", "0"
	}
	logrus.Debugf("synccheck result: %s", data)
	re := regexp.MustCompile(`window.synccheck={retcode:"(\d+)",selector:"(\d+)"}`)
	find := re.FindStringSubmatch(data)
	if len(find) > 2 {
		retcode := find[1]
		selector := find[2]
		debugPrint(fmt.Sprintf("retcode:%s,selector,selector%s", find[1], find[2]))
		return retcode, selector
	} else {
		return "9999", "0"
	}
}

func (self *wxweb) testsynccheck(args ...interface{}) bool {
	SyncHost := []string{
		"webpush.weixin.qq.com",
		"webpush2.weixin.qq.com",
		"webpush.wechat.com",
		"webpush1.wechat.com",
		"webpush2.wechat.com",
		"webpush1.wechatapp.com",
		//"webpush.wechatapp.com"
	}
	for _, host := range SyncHost {
		self.syncHost = host
		retcode, _ := self.synccheck()
		if retcode == "0" {
			self.ifTestSyncOK = true
			return true
		}
	}
	return false
}

func (self *wxweb) webwxstatusnotify(args ...interface{}) bool {
	urlstr := fmt.Sprintf("%s/webwxstatusnotify?lang=zh_CN&passTicket=%s", self.baseUri, self.passTicket)
	params := make(map[string]interface{})
	params["BaseRequest"] = self.BaseRequest
	params["Code"] = 3
	params["FromUserName"] = self.User["UserName"]
	params["ToUserName"] = self.User["UserName"]
	params["ClientMsgId"] = int(time.Now().Unix())
	res, err := self._post(urlstr, params, true)
	if err != nil {
		return false
	}
	data := JsonDecode(res).(map[string]interface{})
	retCode := data["BaseResponse"].(map[string]interface{})["Ret"].(int)
	return retCode == 0
}

func (self *wxweb) webwxgetcontact(args ...interface{}) bool {
	urlstr := fmt.Sprintf("%s/webwxgetcontact?lang=zh_CN&pass_ticket=%s&seq=0&skey=%s&r=%s", self.baseUri, self.passTicket, self.skey, self._unixStr())
	res, err := self._post(urlstr, nil, true)
	if err != nil {
		logrus.Errorf("webwxgetcontact _post error: %v", err)
		return false
	}

	data := JsonDecode(res).(map[string]interface{})
	if data == nil {
		logrus.Errorf("webwxgetcontact JsonDecode error: %v", err)
		return false
	}
	retCode := data["BaseResponse"].(map[string]interface{})["Ret"].(int)
	if retCode != WX_RET_SUCCESS {
		logrus.Errorf("webwxgetcontact get error retcode[%d]", retCode)
		return false
	}

	memberList := data["MemberList"].([]interface{})
	if memberList == nil {
		logrus.Errorf("webwxgetcontact get memberList error")
		return false
	}
	for _, v := range memberList {
		member := v.(map[string]interface{})
		if member == nil {
			logrus.Errorf("webwxgetcontact get member[%v] error.", v)
			continue
		}
		userName := member["UserName"].(string)
		contactFlag := member["ContactFlag"].(int)
		nickName := member["NickName"].(string)
		if strings.HasPrefix(userName, GROUP_PREFIX) {
			ug := NewUserGroup(contactFlag, nickName, userName, self.rankRedis)
			self.contact.Groups[userName] = ug
		} else {
			alias := member["Alias"].(string)
			city := member["City"].(string)
			sex := member["Sex"].(int)
			uf := &UserFriend{
				Alias:       alias,
				City:        city,
				ContactFlag: contactFlag,
				NickName:    nickName,
				Sex:         sex,
				UserName:    userName,
			}
			self.contact.Friends[userName] = uf
		}
	}
	logrus.Debugf("webwxgetcontact get group num: %d", len(self.contact.Groups))
	
	return true
}

func (self *wxweb) webwxbatchgetcontact(args ...interface{}) bool {
	urlstr := fmt.Sprintf("%s/webwxbatchgetcontact?type=ex&lang=zh_CN&pass_ticket=%s&r=%s", self.baseUri, self.passTicket, self._unixStr())
	params := make(map[string]interface{})
	params["BaseRequest"] = self.BaseRequest
	list := make([]map[string]interface{}, 0)
	for _, v := range self.contact.Groups {
		gInfo := make(map[string]interface{})
		gInfo["EncryChatRoomId"] = ""
		gInfo["UserName"] = v.UserName
		list = append(list, gInfo)
	}
	params["List"] = list
	params["Count"] = len(list)
	res, err := self._post(urlstr, params, true)
	if err != nil {
		logrus.Errorf("webwxbatchgetcontact _post error: %v", err)
		return false
	}

	dataJson := JsonDecode(res)
	if dataJson == nil {
		logrus.Errorf("json decode error.")
		return false
	}
	data := dataJson.(map[string]interface{})
	if data == nil {
		logrus.Errorf("webwxbatchgetcontact translate map error: %v", err)
		return false
	}
	retCode := data["BaseResponse"].(map[string]interface{})["Ret"].(int)
	if retCode != WX_RET_SUCCESS {
		logrus.Errorf("webwxbatchgetcontact get error retcode[%d]", retCode)
		return false
	}

	contactList := data["ContactList"].([]interface{})
	if contactList == nil {
		logrus.Errorf("webwxbatchgetcontact get contactList error")
		return false
	}
	for _, v := range contactList {
		contact := v.(map[string]interface{})
		if contact == nil {
			logrus.Errorf("webwxbatchgetcontact get contact[%v] error", v)
			continue
		}
		groupUserName := contact["UserName"].(string)
		groupContactFlag := contact["ContactFlag"].(int)
		groupNickName := contact["NickName"].(string)
		memberList := contact["MemberList"].([]interface{})
		for _, v2 := range memberList {
			member := v2.(map[string]interface{})
			if member == nil {
				logrus.Errorf("webwxbatchgetcontact get member[%v] error", v2)
				continue
			}
			displayName := member["DisplayName"].(string)
			nickName := member["NickName"].(string)
			userName := member["UserName"].(string)
			gui := &GroupUserInfo{
				DisplayName: displayName,
				NickName:    nickName,
				UserName:    userName,
			}
			gv := self.contact.Groups[groupUserName]
			if gv == nil {
				logrus.Errorf("contact groups have no this username[%s]", groupUserName)
				continue
			}
			gv.MemberList[userName] = gui
			gv.NickName = groupNickName
			gv.ContactFlag = groupContactFlag
		}
	}

	return true
}

func (self *wxweb) webgetchatroommember(chatroomId string) (map[string]string, error) {
	urlstr := fmt.Sprintf("%s/webwxbatchgetcontact?type=ex&r=%s&passTicket=%s", self.baseUri, self._unixStr(), self.passTicket)
	params := make(map[string]interface{})
	params["BaseRequest"] = self.BaseRequest
	params["Count"] = 1
	params["List"] = []map[string]string{}
	l := []map[string]string{}
	params["List"] = append(l, map[string]string{
		"UserName":   chatroomId,
		"ChatRoomId": "",
	})
	members := []string{}
	stats := make(map[string]string)
	res, err := self._post(urlstr, params, true)
	debugPrint(params)
	if err != nil {
		return stats, err
	}
	data := JsonDecode(res).(map[string]interface{})
	RoomContactList := data["ContactList"].([]interface{})[0].(map[string]interface{})["MemberList"]
	man := 0
	woman := 0
	for _, v := range RoomContactList.([]interface{}) {
		if m, ok := v.([]interface{}); ok {
			for _, s := range m {
				members = append(members, s.(map[string]interface{})["UserName"].(string))
			}
		} else {
			members = append(members, v.(map[string]interface{})["UserName"].(string))
		}
	}
	urlstr = fmt.Sprintf("%s/webwxbatchgetcontact?type=ex&r=%s&passTicket=%s", self.baseUri, self._unixStr(), self.passTicket)
	length := 50
	debugPrint(members)
	mnum := len(members)
	block := int(math.Ceil(float64(mnum) / float64(length)))
	k := 0
	for k < block {
		offset := k * length
		var l int
		if offset+length > mnum {
			l = mnum
		} else {
			l = offset + length
		}
		blockmembers := members[offset:l]
		params := make(map[string]interface{})
		params["BaseRequest"] = self.BaseRequest
		params["Count"] = len(blockmembers)
		blockmemberslist := []map[string]string{}
		for _, g := range blockmembers {
			blockmemberslist = append(blockmemberslist, map[string]string{
				"UserName":        g,
				"EncryChatRoomId": chatroomId,
			})
		}
		params["List"] = blockmemberslist
		debugPrint(urlstr)
		debugPrint(params)
		dic, err := self._post(urlstr, params, true)
		if err == nil {
			debugPrint("flag")
			userlist := JsonDecode(dic).(map[string]interface{})["ContactList"]
			for _, u := range userlist.([]interface{}) {
				if u.(map[string]interface{})["Sex"].(int) == 1 {
					man++
				} else if u.(map[string]interface{})["Sex"].(int) == 2 {
					woman++
				}
			}
		}
		k++
	}
	stats = map[string]string{
		"woman": strconv.Itoa(woman),
		"man":   strconv.Itoa(man),
	}
	return stats, nil
}

func (self *wxweb) webwxsync() interface{} {
	urlstr := fmt.Sprintf("%s/webwxsync?sid=%s&skey=%s&lang=zh_CN&pass_ticket=%s", self.baseUri, self.sid, self.skey, self.passTicket)
	params := make(map[string]interface{})
	params["BaseRequest"] = self.BaseRequest
	params["SyncKey"] = self.SyncKey
	params["rr"] = ^time.Now().Unix()
	res, err := self._post(urlstr, params, true)
	if err != nil {
		logrus.Errorf("webwxsync post error: %v", err)
		return false
	}
	if res == "" {
		logrus.Errorf("webwxsync res == nil")
		return nil
	}
	dataJson := JsonDecode(res)
	if dataJson == nil {
		logrus.Errorf("webwxsync JsonDecode(res[%s]) == nil", res)
		return nil
	}
	data := dataJson.(map[string]interface{})
	retCode := data["BaseResponse"].(map[string]interface{})["Ret"].(int)
	if retCode == 0 {
		self.SyncKey = data["SyncKey"].(map[string]interface{})
		self._setsynckey()
	}
	return data
}

func (self *wxweb) filterMsg(userName, content string) bool {
	if strings.Contains(content, "→手机淘宝→") {
		return true
	}
	if content == "排行榜" {
		logrus.Debugf("filter: %s", content)
		group := self.contact.Groups[userName]
		if group == nil {
			logrus.Errorf("has no this group[%s]", userName)
			return true
		}
		rank := group.GetInviteRank()
		self.webwxsendmsg(rank, userName)
		return true
	}
	return false
}

func (self *wxweb) handleMsg(r interface{}) {
	if r == nil {
		return
	}
	msgSource := r.(map[string]interface{})
	if msgSource == nil {
		return
	}
//MOD_CONTACT_LIST:
	modContactList := msgSource["ModContactList"]
	if modContactList != nil {
		contactList := modContactList.([]interface{})
		if contactList != nil {
			for _, v := range contactList {
				modContact := v.(map[string]interface{})
				userName := modContact["UserName"].(string)
				if strings.HasPrefix(userName, GROUP_PREFIX) {
					groupContactFlag := modContact["ContactFlag"].(int)
					groupNickName := modContact["NickName"].(string)
					group := self.contact.Groups[userName]
					if group == nil {
						group = NewUserGroup(groupContactFlag, groupNickName, userName, self.rankRedis)
					} else {
						group.ContactFlag = groupContactFlag
						group.NickName = groupNickName
						logrus.Debugf("mod contact group: %s nickname: %s", userName, groupNickName)
					}
					memberList := modContact["MemberList"].([]interface{})
					memberListMap := make(map[string]*GroupUserInfo)
					for _, v2 := range memberList {
						member := v2.(map[string]interface{})
						if member == nil {
							logrus.Errorf("handlemsg get member[%v] error", v2)
							continue
						}
						displayName := member["DisplayName"].(string)
						nickName := member["NickName"].(string)
						userName := member["UserName"].(string)
						gui := &GroupUserInfo{
							DisplayName: displayName,
							NickName:    nickName,
							UserName:    userName,
						}
						memberListMap[userName] = gui
					}
					logrus.Debugf("mod contact group: %s oldmemberlen:%d newmemberlen: %d", userName, len(group.MemberList), len(memberListMap))
					group.MemberList = memberListMap
					self.contact.Groups[userName] = group
				}
			}
		}
	}
	
	addMsgList := msgSource["AddMsgList"]
	if addMsgList == nil {
		return
	}
	msgList := addMsgList.([]interface{})
	if msgList == nil {
		return
	}
	//myNickName := self.User["NickName"].(string)
	for _, v := range msgList {
		msg := v.(map[string]interface{})
		if msg == nil {
			continue
		}
		msgType := msg["MsgType"].(int)
		fromUserName := msg["FromUserName"].(string)
		// name = self.getUserRemarkName(msg['FromUserName'])
		content := msg["Content"].(string)
		content = strings.Replace(content, "&lt;", "<", -1)
		content = strings.Replace(content, "&gt;", ">", -1)
		content = strings.Replace(content, " ", " ", 1)
		msgid := msg["MsgId"].(string)
		if msgType == 1 {
			var ans string
			var err error
			if fromUserName[:2] == GROUP_PREFIX {
				contentSlice := strings.Split(content, ":<br/>")
				people := contentSlice[0]
				content = contentSlice[1]
				if self.filterMsg(fromUserName, content) {
					continue
				}
				logrus.Debugf("[*] 你有新的群文本消息，请注意查收")
				group := self.contact.Groups[fromUserName]
				if group == nil {
					logrus.Errorf("cannot found the group[%s]", fromUserName)
					continue
				}
				sendPeople := group.MemberList[people]
				if sendPeople == nil {
					continue
				}
				msg := &MsgInfo{
					WXMsgId: msgid,
					NickName: sendPeople.NickName,
					UserName: sendPeople.UserName,
					Content: content,
				}
				group.AppendMsg(msg)
				
				//if strings.Contains(content, "@"+myNickName) {
				//	realcontent := strings.TrimSpace(strings.Replace(content, "@"+myNickName, "", 1))
				//	debugPrint(realcontent)
				//	if realcontent == "统计人数" {
				//		stat, err := self.webgetchatroommember(fromUserName)
				//		if err == nil {
				//			ans = "据统计群里男生" + stat["man"] + "人，女生" + stat["woman"] + "人 (ó㉨ò)"
				//		}
				//	} else {
				//		ans, err = self.getReplyByApi(realcontent, fromUserName)
				//	}
				//} else if strings.Contains(content, "撩@") {
				//	name := strings.Replace(content, "撩@", "", 1)
				//	name = strings.Replace(name, "\u003cbr/\u003e", "", 1)
				//	ans, err = self.getReplyByApi(LOVEWORDS_QUEST, fromUserName)
				//	if err == nil {
				//		ans = "@" + name + " " + ans
				//	}
				//} else if content == "撩我" {
				//	ans, err = self.getReplyByApi(LOVEWORDS_QUEST, fromUserName)
				//}
			}
			//else {
			//	ans, err = self.getReplyByApi(content, fromUserName)
			//}
			if err != nil {
				debugPrint(err)
			} else if ans != "" {
				go self.webwxsendmsg(ans, fromUserName)
			}
		} else if msgType == 51 {
			logrus.Debug("[*] 成功截获微信初始化消息")
		} else if msgType == 10000 {
			logrus.Debugf("系统消息: %s", content)
			if strings.Contains(content, "邀请") {
				group := self.contact.Groups[fromUserName]
				if group == nil {
					continue
				}
				group.AppendInviteMsg(&MsgInfo{WXMsgId: msgid, Content: content})
			}
		} else if msgType == 37 {
			recommendInfo := msg["RecommendInfo"]
			if recommendInfo == nil {
				logrus.Errorf("recommendInfo == nil")
				return
			}
			rInfo := recommendInfo.(map[string]interface{})
			if rInfo == nil {
				logrus.Errorf("rInfo == nil")
				return
			}
			ticket := rInfo["Ticket"].(string)
			userName := rInfo["UserName"].(string)
			nickName := rInfo["NickName"].(string)
			logrus.Debugf("receive nickName[%s] with friend.", nickName)
			ok := self.webwxverifyuser(ticket, userName)
			if ok {
				self.webwxsendmsgimg(userName, "@crypt_6a815982_15da5f7d8298dcb9832e2df69ea19958740d84d8b84e140bb3257e84e89f2e07bd016094264fa8e8a2ef10cab715a72473ac0760589e27dfcd148beb843ee6165529a2c54f5648982479d9854df19570802bc3860de2bad66a7763d58c82d93662177da207fc46b06515f5f350fc3de4862c13d648cc901286a9b9c38bec2ad5328cbbac7c7545f6684fae8845faf38195a1169477d22014d795e52eea10436d824a3502d3afffd039c96aa032e77590c860540040f8e088f3148b7aeb405a6eed48f8b227cf36e8c822b1c3e5f7223b42581d7bf29eab6a04a1cc2323ad9252c446e80e7e96a13270214ba2b396dc3aa670621a02eced9b2c408a56f44467e80ec56b45d932505e57a702a1dc3dc9e391d0c3f8f26b8721ac07d84d39ee4ddb0da2521f78b28ce445a410e6836387aeb2578fa458182905cd46a32b61146c99adf00c1a41fc394461df7b3a8307a2272454d782ce724a4080b60a0c019aaffb")
				self.webwxsendmsg("xxxxx", userName)
			}
		}
	}
}

func (self *wxweb) getReplyByApi(realcontent string, fromUserName string) (string, error) {
	return getAnswer(realcontent, fromUserName, self.User["NickName"].(string))
}

func (self *wxweb) webwxverifyuser(ticket, userName string) bool {
	urlstr := fmt.Sprintf("%s/webwxverifyuser?r=%s&lang=zh_CN&pass_ticket=%s", self.baseUri, self._unixStr(), self.passTicket)
	params := make(map[string]interface{})
	params["BaseRequest"] = self.BaseRequest
	params["Opcode"] = 3
	params["SceneList"] = []int{33}
	params["SceneListCount"] = 1
	params["VerifyContent"] = ""
	params["VerifyUserList"] = []map[string]interface{}{map[string]interface{}{"Value": userName, "VerifyUserTicket": ticket}}
	params["VerifyUserListSize"] = 1
	params["skey"] = self.skey
	data, err := self._post(urlstr, params, true)
	if err != nil {
		logrus.Errorf("webwxverifyuser error: %v", err)
		return false
	} else {
		logrus.Debugf("webwxverifyuser usrname[%s] success, get data[%s].", userName, data)
		return true
	}
}

func (self *wxweb) webwxsendmsgimg(toUserName, mediaId string) bool {
	urlstr := fmt.Sprintf("%s/webwxsendmsgimg?fun=async&f=json&lang=zh_CN&pass_ticket=%s", self.baseUri, self.passTicket)
	clientMsgId := self._unixStr() + "0" + strconv.Itoa(rand.Int())[3:6]
	params := make(map[string]interface{})
	params["BaseRequest"] = self.BaseRequest
	msg := make(map[string]interface{})
	msg["Type"] = 3
	msg["MediaId"] = mediaId
	msg["FromUserName"] = self.User["UserName"]
	msg["ToUserName"] = toUserName
	msg["LocalID"] = clientMsgId
	msg["ClientMsgId"] = clientMsgId
	params["Msg"] = msg
	data, err := self._post(urlstr, params, true)
	if err != nil {
		logrus.Errorf("wx send mediaId[%s] toUserName[%s] error: %s", mediaId, toUserName, err)
		return false
	} else {
		logrus.Debugf("wx send mediaId[%s] toUserName[%s] get data[%s] success.", mediaId, toUserName, data)
		return true
	}
}

func (self *wxweb) webwxsendmsg(message string, toUseName string) bool {
	urlstr := fmt.Sprintf("%s/webwxsendmsg?pass_ticket=%s", self.baseUri, self.passTicket)
	clientMsgId := self._unixStr() + "0" + strconv.Itoa(rand.Int())[3:6]
	params := make(map[string]interface{})
	params["BaseRequest"] = self.BaseRequest
	msg := make(map[string]interface{})
	msg["Type"] = 1
	msg["Content"] = message
	msg["FromUserName"] = self.User["UserName"]
	msg["ToUserName"] = toUseName
	msg["LocalID"] = clientMsgId
	msg["ClientMsgId"] = clientMsgId
	params["Msg"] = msg
	data, err := self._post(urlstr, params, true)
	if err != nil {
		logrus.Errorf("wx send msg[%s] toUserName[%s] error: %s", message, toUseName, err)
		return false
	} else {
		logrus.Debugf("wx send msg[%s] toUserName[%s] get data[%s] success.", message, toUseName, data)
		return true
	}
}

func (self *wxweb) _init() {
	logrus.SetLevel(logrus.DebugLevel)
	
	gCookieJar, _ := cookiejar.New(nil)
	httpclient := http.Client{
		CheckRedirect: nil,
		Jar:           gCookieJar,
	}
	self.httpClient = &httpclient
	rand.Seed(time.Now().Unix())
	str := strconv.Itoa(rand.Int())
	self.deviceId = "e" + str[2:17]
	self.contact = NewUserContact()
}

func (self *wxweb) test() {

}

func (self *wxweb) start() {
	logrus.Info("[*] 微信网页版 ... 开动")
	self._init()
	self._run("[*] 正在获取 uuid ... ", self.getUuid)
	self._run("[*] 正在获取 二维码 ... ", self.genQRcode)
	logrus.Infof("[*] 请使用微信扫描二维码以登录 ... ")
	for {
		if self.waitForLogin(1) == false {
			continue
		}
		logrus.Infof("[*] 请在手机上点击确认以登录 ... ")
		if self.waitForLogin(0) == false {
			continue
		}
		break
	}
	self._run("[*] 正在登录 ... ", self.login)
	self._run("[*] 微信初始化 ... ", self.webwxinit)
	self._run("[*] 开启状态通知 ... ", self.webwxstatusnotify)
	self._run("[*] 进行同步线路测试 ... ", self.testsynccheck)
	self._run("[*] 获取好友列表 ... ", self.webwxgetcontact)
	self._run("[*] 获取群列表 ... ", self.webwxbatchgetcontact)
	self.contact.PrintGroupInfo()
	for {
		self.lastCheckTs = time.Now().Unix()
		retcode, selector := self.synccheck()
		if retcode == "1100" {
			logrus.Info("[*] 你在手机上登出了微信, 88")
			break
		} else if retcode == "1101" {
			logrus.Info("[*] 你在其他地方登录了 WEB 版微信, 88")
			break
		} else if retcode == "0" {
			if selector == "2" {
				r := self.webwxsync()
				if r == nil {
					time.Sleep(1 * time.Second)
					continue
				}
				switch r.(type) {
				case bool:
				default:
					self.handleMsg(r)
				}
			} else if selector == "0" {
				time.Sleep(1 * time.Second)
			} else if selector == "6" || selector == "4" {
				self.webwxsync()
				time.Sleep(1 * time.Second)
			} else if selector == "7" {
				self.webwxsync()
				time.Sleep(1 * time.Second)
			} else if selector == "3" {
				self.webwxsync()
				time.Sleep(1 * time.Second)
			} else {
				self.webwxsync()
				time.Sleep(1 * time.Second)
			}
		}
		if (time.Now().Unix() - self.lastCheckTs) <= 3 {
			time.Sleep(time.Duration(time.Now().Unix() - self.lastCheckTs) * time.Second)
		}
	}
}

func forgeheadget(urlstr string) string {

	client := &http.Client{}

	reqest, err := http.NewRequest("GET", urlstr, nil)

	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(0)
	}

	reqest.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	reqest.Header.Add("Accept-Encoding", "gzip, deflate, sdch")
	reqest.Header.Add("Accept-Language", "zh-CN,zh;q=0.8")
	reqest.Header.Add("Connection", "keep-alive")
	reqest.Header.Add("Host", "login.weixin.qq.com")
	reqest.Header.Add("Referer", "https://wx.qq.com/")
	reqest.Header.Add("Upgrade-Insecure-Requests", "1")
	reqest.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/47.0.2526.111 Safari/537.36")
	response, err := client.Do(reqest)
	defer response.Body.Close()

	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(0)
	}
	body, _ := ioutil.ReadAll(response.Body)
	return string(body)
}
