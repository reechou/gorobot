package wxweb

type ReceiveMsgInfo struct {
	WeChat       string
	ReceiveEvent string
	FromType     string
	FromUserName string
	FromNickName string
	Msg          string
	Ticket       string // for verify
}
