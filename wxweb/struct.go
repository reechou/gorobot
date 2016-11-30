package wxweb

type ReceiveMsgInfo struct {
	WeChat        string
	ReceiveEvent  string
	FromType      string
	FromUserName  string
	FromNickName  string
	FromGroupName string
	Msg           string
	Ticket        string // for verify
}
