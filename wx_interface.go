package main

import (
	"fmt"
	
	"github.com/Sirupsen/logrus"
)

func (self *wxweb) webwxBatchGetContact(groups map[string]*UserGroup) bool {
	urlstr := fmt.Sprintf("%s/webwxbatchgetcontact?type=ex&lang=zh_CN&pass_ticket=%s&r=%s", self.baseUri, self.passTicket, self._unixStr())
	params := make(map[string]interface{})
	params["BaseRequest"] = self.BaseRequest
	list := make([]map[string]interface{}, 0)
	for _, v := range groups {
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
