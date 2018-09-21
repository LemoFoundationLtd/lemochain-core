package synchronise

import (
	"fmt"
	"testing"
	"time"
)

func Test_Fetcher(t *testing.T) {
	//urlStr := "http://zhuanjistatic.kugou.com/v3/singer_album/get_latest_from_singer_names_pc?singer_names=%E8%B5%B5%E9%B9%8F&singer_ids=3550&platform=1001&version=8254"
	//req, err := http.NewRequest("GET", urlStr, strings.NewReader(""))
	//if err != nil {
	//	t.Error(fmt.Sprintf("request error,err:%v", err))
	//}
	//req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	//req.Header.Add("Connection", "keep-alive")
	//req.Header.Add("User-Agent", "KuGou2012-8254-web_browser_event_handler")
	//
	//client := &http.Client{}
	//resp, err := client.Do(req)
	//if err != nil {
	//	t.Error(fmt.Sprintf("client.Do error,err:%v", err))
	//}
	//defer resp.Body.Close()
	//body, err := ioutil.ReadAll(resp.Body)
	//if err != nil {
	//	t.Error(fmt.Sprintf("read resp.Body error,err:%v", err))
	//}
	//fmt.Println(string(body))
}

func Test_timeAfter(t *testing.T) {
	now := time.Now()
	front := time.Now().Add((-1) * time.Second)
	after := time.Now().Add(1 * time.Second)
	if now.After(front) {
		fmt.Println("now>front")
	}
	if now.After(after) {
		fmt.Println("now>after")
	}
}

func Test_slice(t *testing.T) {
	buf := make([]byte, 0)
	buf = append(buf, byte(0x10))
	modify(buf)
	fmt.Println("", buf)
}

func modify(buf []byte) {
	buf[0] = byte(0x11)
}
