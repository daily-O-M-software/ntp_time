package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"time"
)

// ntpEpochOffset 为 NTP 时间起点(1900-01-01)与 Unix 起点(1970-01-01)之间的秒数。
const ntpEpochOffset = 2208988800

// ntpQuery 向指定 NTP 服务器发起 SNTP 请求。
// offset: 本机时钟相对服务器的偏差（本机时间 + offset = 正确时间）；
// rtt: 网络往返延迟。
func ntpQuery(server string, timeout time.Duration) (offset, rtt time.Duration, err error) {
	conn, err := net.DialTimeout("udp", ntpAddr(server), timeout)
	if err != nil {
		return 0, 0, fmt.Errorf("连接失败: %w", err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return 0, 0, err
	}

	req := make([]byte, 48)
	req[0] = 0x23 // LI=0, VN=4, Mode=3(客户端)
	t0 := time.Now()
	binary.BigEndian.PutUint32(req[40:44], uint32(int64(t0.Unix())+ntpEpochOffset))
	binary.BigEndian.PutUint32(req[44:48], uint32(uint64(t0.Nanosecond())<<32/1e9))
	sentTS := append([]byte(nil), req[40:48]...)
	if _, err := conn.Write(req); err != nil {
		return 0, 0, fmt.Errorf("发送请求失败: %w", err)
	}

	resp := make([]byte, 48)
	n, err := conn.Read(resp)
	t3 := time.Now()
	if err != nil {
		return 0, 0, fmt.Errorf("等待应答超时: %w", err)
	}
	if n < 48 {
		return 0, 0, fmt.Errorf("应答不完整: 仅 %d 字节", n)
	}
	if mode := resp[0] & 0x07; mode != 4 && mode != 5 {
		return 0, 0, fmt.Errorf("非 NTP 服务器应答 (mode=%d)", mode)
	}
	if !bytes.Equal(resp[24:32], sentTS) {
		return 0, 0, fmt.Errorf("应答与请求不匹配")
	}
	if li := resp[0] >> 6; li == 3 {
		return 0, 0, fmt.Errorf("服务器时钟未同步 (leap=%d)", li)
	}
	if resp[1] == 0 { // stratum 0: kiss-of-death
		return 0, 0, fmt.Errorf("服务器拒绝请求 (stratum=0)")
	}

	t1 := ntpToTime(binary.BigEndian.Uint32(resp[32:36]), binary.BigEndian.Uint32(resp[36:40]))
	t2 := ntpToTime(binary.BigEndian.Uint32(resp[40:44]), binary.BigEndian.Uint32(resp[44:48]))
	offset = ((t1.Sub(t0) + t2.Sub(t3)) / 2).Round(time.Millisecond)
	rtt = (t3.Sub(t0) - t2.Sub(t1)).Round(time.Millisecond)
	return offset, rtt, nil
}

// ntpToTime 将 NTP 的 32 位秒数和小数部分转换为 UTC 时间。
func ntpToTime(sec, frac uint32) time.Time {
	unix := int64(sec) - ntpEpochOffset
	if unix < 0 { // 2036 年 NTP 32 位秒数回绕后继续计时
		unix += 1 << 32
	}
	return time.Unix(unix, int64(frac)*1e9>>32).UTC()
}

// ntpAddr 补全 NTP 默认端口 123；地址中已包含端口则原样使用。
func ntpAddr(server string) string {
	if strings.Contains(server, ":") {
		return server
	}
	return net.JoinHostPort(server, "123")
}
