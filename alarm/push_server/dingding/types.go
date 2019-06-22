package dingding

type ResponseMsg struct {
	Errcode int    `json:"errcode"`
	Errmsg  string `json:"errmsg"`
}

// text参数
type textParams struct {
	Content string `json:"content"`
}

// @参数
type AtParams struct {
	AtMobiles []string `json:"atMobiles"` // 需要@的电话号码
	IsAtAll   bool     `json:"isAtAll"`   // 是否需要@全部成员
}

// 普通文本
type textMsg struct {
	MsgType string     `json:"msgtype"`
	Text    textParams `json:"text"`
	At      AtParams   `json:"at"`
}

type markdownParams struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

// markdown文本
type markdownMsg struct {
	MsgType  string         `json:"msgtype"`
	Markdown markdownParams `json:"markdown"`
	At       AtParams       `json:"at"`
}
