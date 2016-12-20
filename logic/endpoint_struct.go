package logic

const (
	WX_RESPONSE_OK = iota
	WX_RESPONSE_ERR
)

type StartWxRsp struct {
	UUID      string `json:"uuid"`
	QrcodeUrl string `json:"qrcodeUrl"`
}

type WxResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}
