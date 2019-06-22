package dingding

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"io/ioutil"
	"net/http"
	"sync"
)

type Robots interface {
	SendText(content string, atMobiles []string, isAtall bool) error
	SendMarkdown(title, markdownText string, atMobiles []string, isAtall bool) error
}

var (
	ErrSendDingRobot = errors.New("send ding ding robot failed. ")
)

// 钉钉机器人推送
type Robot struct {
	WebHook string // 机器人的Hook地址
	lock    sync.Mutex
}

func NewRobot(webHook string) Robots {
	return &Robot{
		WebHook: webHook,
		lock:    sync.Mutex{},
	}
}

// SendText 发送普通类型的message
func (r *Robot) SendText(content string, atMobiles []string, isAtall bool) error { // content: 发送的文本内容。atMobiles:需要@的手机号列表。isAtall: 为true表示@所有人
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.send(&textMsg{
		MsgType: "text",
		Text: textParams{
			Content: content,
		},
		At: AtParams{
			AtMobiles: atMobiles,
			IsAtAll:   isAtall,
		},
	})
}

func (r *Robot) SendMarkdown(title, markdownText string, atMobiles []string, isAtall bool) error { // title: markdown的标题
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.send(&markdownMsg{
		MsgType: "markdown",
		Markdown: markdownParams{
			Title: title,
			Text:  markdownText,
		},
		At: AtParams{
			AtMobiles: atMobiles,
			IsAtAll:   isAtall,
		},
	})
}

// 发送消息到ding ding机器人的接口
func (r *Robot) send(msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	resp, err := http.Post(r.WebHook, "application/json; charset=utf-8", bytes.NewReader(data))
	if err != nil {
		return err
	}
	by, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respMsg := &ResponseMsg{}
	err = json.Unmarshal(by, respMsg)
	if err != nil {
		return err
	}
	if respMsg.Errcode != 0 {
		log.Errorf("Send ding ding robot failed. errcode: %d, errMsg: %s", respMsg.Errcode, respMsg.Errmsg)
		return ErrSendDingRobot
	}
	return nil
}
