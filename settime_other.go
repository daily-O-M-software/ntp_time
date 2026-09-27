//go:build !windows

package main

import (
	"errors"
	"time"
)

// setSystemTime 非 Windows 平台的占位实现，仅保证程序可编译。
func setSystemTime(t time.Time) error {
	return errors.New("当前平台不支持设置系统时间")
}
