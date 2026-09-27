package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	defaultServers   = "ntp.aliyun.com,time.windows.com"
	defaultInterval  = 300
	defaultThreshold = 1
	defaultLogfile   = "ntp_sync.log"
	minInterval      = 30
	// cfgName 默认配置文件名，与主程序同名（仅扩展名为 .ini）。
	cfgName = "ntp_time.ini"
)

// Config 校时程序配置。
type Config struct {
	Servers   []string // NTP 服务器列表，按顺序尝试
	Interval  int      // 定时校对间隔（秒）
	Threshold int      // 校准阈值（秒），偏差超过才写时钟；0 表示每次都校准
	Logfile   string   // 日志文件名（相对程序目录），空表示只输出控制台
}

// cfgFile 返回配置文件路径：未指定时用程序目录下的 ntp_time.ini。
func cfgFile(p string) string {
	if p != "" {
		return p
	}
	return filepath.Join(exeDir(), cfgName)
}

// loadConfig 读取并解析配置文件。文件不存在时自动生成默认配置，
// created 表示本次运行是否生成了新配置文件。
func loadConfig(path string) (cfg *Config, created bool, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, false, err
		}
		cfg = defaultConfig()
		if werr := writeDefaultConfig(path); werr != nil {
			return nil, false, fmt.Errorf("生成默认配置 %s 失败: %v", path, werr)
		}
		return cfg, true, nil
	}
	return parseConfig(string(data)), false, nil
}

func defaultConfig() *Config {
	return &Config{
		Servers:   splitServers(defaultServers),
		Interval:  defaultInterval,
		Threshold: defaultThreshold,
		Logfile:   defaultLogfile,
	}
}

func splitServers(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// parseConfig 解析 key=value 格式的配置文本，非法值回退默认值。
func parseConfig(text string) *Config {
	cfg := defaultConfig()
	text = strings.TrimPrefix(text, "\ufeff")
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		eq := strings.Index(line, "=")
		if eq < 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(line[:eq]))
		val := strings.TrimSpace(line[eq+1:])
		switch key {
		case "server":
			if servers := splitServers(val); len(servers) > 0 {
				cfg.Servers = servers
			}
		case "interval":
			if n, e := strconv.Atoi(val); e == nil && n > 0 {
				if n < minInterval {
					n = minInterval
				}
				cfg.Interval = n
			}
		case "threshold":
			if n, e := strconv.Atoi(val); e == nil && n >= 0 {
				cfg.Threshold = n
			}
		case "logfile":
			cfg.Logfile = val
		}
	}
	return cfg
}

const defaultConfigText = `# NTP 校时程序配置文件
# NTP 服务器地址，多个用英文逗号分隔，按顺序尝试，失败的自动跳过
server = ntp.aliyun.com,time.windows.com
# 定时校对间隔（秒），最小 30
interval = 300
# 偏差阈值（秒）：偏差超过该值才校准系统时间；0 表示每次都校准
threshold = 1
# 日志文件名（位于程序目录），留空则只输出到控制台
logfile = ntp_sync.log
`

func writeDefaultConfig(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	// UTF-8 BOM，便于 Windows 7 记事本正确识别中文注释
	if _, err := f.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return err
	}
	_, err = f.WriteString(defaultConfigText)
	return err
}
