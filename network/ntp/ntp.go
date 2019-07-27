package ntp

import (
	"encoding/binary"
	"errors"
	"github.com/LemoFoundationLtd/lemochain-core/common/log"
	"net"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"sort"
	"time"
)

const (
	host           = "cn.ntp.org.cn:123"    // ntp服务器
	driftThreshold = 500 * time.Millisecond // 允许的时间误差范围
)

var (
	ErrModifyPermission = errors.New("only the time modification for Linux system is supported")
	ErrNotRootUser      = errors.New("user has no permission to modify system time")
)

type durationSlice []time.Duration

func (s durationSlice) Len() int           { return len(s) }
func (s durationSlice) Less(i, j int) bool { return s[i] < s[j] }
func (s durationSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

type Packet struct {
	Settings       uint8  // leap yr indicator, ver number, and mode
	Stratum        uint8  // stratum of local clock
	Poll           int8   // poll exponent
	Precision      int8   // precision exponent
	RootDelay      uint32 // root delay
	RootDispersion uint32 // root dispersion
	ReferenceID    uint32 // reference id
	RefTimeSec     uint32 // reference timestamp sec
	RefTimeFrac    uint32 // reference timestamp fractional
	OrigTimeSec    uint32 // origin time secs
	OrigTimeFrac   uint32 // origin time fractional
	RxTimeSec      uint32 // receive time secs
	RxTimeFrac     uint32 // receive time frac
	TxTimeSec      uint32 // transmit time secs
	TxTimeFrac     uint32 // transmit time frac
}

// TimeProof 同步时间并进行修改系统时间
func TimeProof() error {
	log.Info("Start system time proof.")
	measurements := 10 // 获取ntp服务器上的时间的次数
	diffs := make([]time.Duration, 0, measurements)

	for i := 0; i < measurements+2; i++ {
		// 拨号并获取本地时间和标准时间的差值
		diffTime, err := dialNtpServerAndGetDiffTime(host)
		if err != nil {
			return err
		}
		diffs = append(diffs, diffTime)
	}
	// 计算最终的时间差
	finalDiff := calcDiffTime(durationSlice(diffs), measurements)
	// 如果差值在允许的误差范围之内，则不用修改系统时间
	if finalDiff > -driftThreshold && finalDiff < driftThreshold {
		return nil
	}
	log.Infof("Start modify system time. If modify failed, Please enable network time synchronisation in system settings. time difference: %dns", int64(finalDiff))
	// 修改系统时间
	if err := modifySysTime(int64(finalDiff)); err != nil {
		return err
	}
	return nil
}

// dialNtpServerAndGetDiffTime 拨号ntp服务并获取时间差
func dialNtpServerAndGetDiffTime(host string) (time.Duration, error) {
	conn, err := net.Dial("udp", host)
	if err != nil {
		log.Errorf("UDP dial error: %vPlease restart glemo.", err)
		return 0, err
	}

	send := time.Now()
	if err := conn.SetDeadline(time.Now().Add(15 * time.Second)); err != nil {
		log.Errorf("set deadline error: %s", err)
		return 0, err
	}
	req := &Packet{Settings: 0x1B}

	// 写入请求
	if err := binary.Write(conn, binary.BigEndian, req); err != nil {
		log.Errorf("Write conn error: %s", err)
		return 0, err
	}

	// 读取socket
	resp := &Packet{}
	if err := binary.Read(conn, binary.BigEndian, resp); err != nil {
		log.Errorf("read socket error: %s", err)
		return 0, err
	}
	conn.Close()
	elapsed := time.Since(send) // 网络传输时间
	/*
		Unix 时间是一个开始于 1970 年的纪元（或者说从 1970 年开始的秒数）。
		然而 NTP 使用的是另外一个纪元，从 1900 年开始的秒数。
		因此，从 NTP 服务端获取到的值要正确地转成 Unix 时间必须减掉这 70 年间的秒数 (1970-1900)
	*/
	sec := int64(resp.TxTimeSec)                                                          // 秒数
	frac := (int64(resp.TxTimeFrac) * 1e9) >> 32                                          // 纳秒位
	nanosec := sec*1e9 + frac                                                             // 纳秒时间戳
	tt := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(nanosec)).Local() // 获得从1900年1月1日开始的纳秒时间戳
	// 时间差
	diffTime := send.Sub(tt) + elapsed/2 // 与返回回来的时间差,本地时间 - 标准时间
	return diffTime, nil
}

// calcDiffTime 计算出最终的时间差
func calcDiffTime(diffs durationSlice, measurements int) time.Duration {
	// 排序
	sort.Sort(diffs)
	// 去掉最高位和最低位求平均值
	var finalDiff time.Duration = 0
	sum := diffs[1]
	for i := 2; i < len(diffs)-1; i++ {
		next := sum + diffs[i]
		if sum^next < 0 { // 符号相反，说明溢出了
			finalDiff = diffs[1]
			break
		}
		sum = next
	}

	if finalDiff == time.Duration(0) {
		finalDiff = sum / time.Duration(measurements)
	}
	return finalDiff
}

// modifySysTime 传入参数为本地超过标准时间的时间戳数，可为正数和负数
func modifySysTime(overTimestamp int64) error {
	if runtime.GOOS != "linux" {
		return ErrModifyPermission
	}
	if !IsRoot() { // 非root用户无法修改系统时间
		return ErrNotRootUser
	}

	// 修改系统时间
	standardTimestamp := time.Now().UnixNano() - overTimestamp // 计算出标准时间戳
	// 转换为系统设置时间的字符串类型
	standardTime := time.Unix(0, standardTimestamp).Local().Format("20060102 15:04:05.999999999")
	cmd := exec.Command("date", "-s", standardTime)
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		log.Errorf("modify system time error: %v", err)
		return err
	}
	return nil
}

// IsRoot 是否为root用户
func IsRoot() bool {
	u, err := user.Current()
	if err != nil {
		log.Errorf("Get system user name error: %v", err)
		return false
	}
	if u.Username == "root" { // 不支持windows
		return true
	} else {
		return false
	}
}
