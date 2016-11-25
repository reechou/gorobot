package logic

type EventTimer struct {
	wxm      *WxManager
	WeChat   string
	Time     string
	Interval int
	Function string
	Do       string
	DoEvent  []DoEvent
}
