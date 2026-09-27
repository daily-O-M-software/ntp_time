// NTP 定时校时程序
//
// 从配置文件读取 NTP 服务器地址和校对间隔，定时同步本机系统时间。
// 必须使用 Go 1.20.x 编译（Go 1.21 起不再支持 Windows 7）。
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const ntpTimeout = 5 * time.Second

func main() {
	cfgPath := flag.String("c", "", "配置文件路径（默认为程序目录下的 ntp_time.ini）")
	once := flag.Bool("once", false, "校时一次后退出")
	check := flag.Bool("check", false, "仅查询 NTP 并显示偏差，不修改系统时钟")
	flag.Parse()

	path := cfgFile(*cfgPath)
	cfg, created, err := loadConfig(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}
	if created {
		fmt.Println("配置文件不存在，已生成默认配置:", path)
	}

	logger, closeLog, err := newLogger(cfg.Logfile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "初始化日志失败: %v\n", err)
		os.Exit(1)
	}
	if closeLog != nil {
		defer closeLog.Close()
	}

	logger.Printf("NTP 校时程序启动: 间隔 %d 秒, 阈值 %d 秒, 服务器 %s",
		cfg.Interval, cfg.Threshold, strings.Join(cfg.Servers, ","))

	// -check / -once: 单次模式后退出
	if *check || *once {
		syncOnce(logger, cfg, !*check)
		return
	}

	// 常驻模式: 启动即校时一次，之后按配置间隔循环
	syncOnce(logger, cfg, true)
	ticker := time.NewTicker(time.Duration(cfg.Interval) * time.Second)
	defer ticker.Stop()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	for {
		select {
		case <-ticker.C:
			syncOnce(logger, cfg, true)
		case <-sig:
			logger.Println("收到退出信号，程序结束")
			return
		}
	}
}

// syncOnce 执行一次校时：依次尝试各服务器，成功一个即处理。
// allowWrite 为 false 时只查询并显示偏差，不修改系统时钟（-check 模式）。
func syncOnce(logger *log.Logger, cfg *Config, allowWrite bool) {
	for _, server := range cfg.Servers {
		offset, rtt, err := ntpQuery(server, ntpTimeout)
		if err != nil {
			logger.Printf("[%s] 查询失败: %v", server, err)
			continue
		}
		logger.Printf("[%s] 查询成功: 本机时钟偏差 %v, 往返延迟 %v", server, offset, rtt)

		if !allowWrite {
			logger.Println("check 模式: 不修改系统时钟")
			return
		}

		abs := offset
		if abs < 0 {
			abs = -abs
		}
		if cfg.Threshold > 0 && abs < time.Duration(cfg.Threshold)*time.Second {
			logger.Println("偏差在阈值内，无需校准")
			return
		}

		target := time.Now().Add(offset)
		if err := setSystemTime(target); err != nil {
			logger.Printf("[%s] 校时失败: %v", server, err)
			return
		}
		logger.Printf("校时完成，系统时间已调整为 %s", time.Now().Format("2006-01-02 15:04:05.000"))
		return
	}
	logger.Println("所有 NTP 服务器均查询失败")
}

// exeDir 返回程序所在目录，配置和日志默认放在该目录（便携式布局）。
func exeDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

// newLogger 创建同时输出到控制台和日志文件的 logger。
// logfile 为空时仅输出到控制台。
func newLogger(logfile string) (*log.Logger, io.Closer, error) {
	writers := []io.Writer{os.Stdout}
	var closer io.Closer
	if logfile != "" {
		path := logfile
		if !filepath.IsAbs(path) {
			path = filepath.Join(exeDir(), path)
		}
		fresh := false
		if _, err := os.Stat(path); os.IsNotExist(err) {
			fresh = true
		}
		f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return nil, nil, fmt.Errorf("打开日志文件 %s 失败: %v", path, err)
		}
		if fresh {
			// UTF-8 BOM，便于 Windows 记事本正确识别中文
			_, _ = f.Write([]byte{0xEF, 0xBB, 0xBF})
		}
		writers = append(writers, f)
		closer = f
	}
	return log.New(io.MultiWriter(writers...), "", log.LstdFlags), closer, nil
}
