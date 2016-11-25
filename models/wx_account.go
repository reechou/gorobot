package models

import (
	"time"

	"github.com/Sirupsen/logrus"
)

type WxAccount struct {
	ID        int64  `xorm:"pk autoincr"`
	UserName  string `xorm:"not null default '' varchar(128) unique"`
	NickName  string `xorm:"not null default '' varchar(256)"`
	CreatedAt int64  `xorm:"not null default 0 int"`
}

func CreateWxAccount(info *WxAccount) error {
	now := time.Now().Unix()
	info.CreatedAt = now

	_, err := x.Insert(info)
	if err != nil {
		logrus.Errorf("create activity error: %v", err)
		return err
	}
	logrus.Infof("create wx_account[%s - %s] success.", info.UserName, info.NickName)

	return nil
}
