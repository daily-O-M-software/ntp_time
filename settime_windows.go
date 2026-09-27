//go:build windows

package main

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

var (
	modkernel32               = syscall.NewLazyDLL("kernel32.dll")
	procSetSystemTime         = modkernel32.NewProc("SetSystemTime")
	modadvapi32               = syscall.NewLazyDLL("advapi32.dll")
	procOpenProcessToken      = modadvapi32.NewProc("OpenProcessToken")
	procLookupPrivilegeValue  = modadvapi32.NewProc("LookupPrivilegeValueW")
	procAdjustTokenPrivileges = modadvapi32.NewProc("AdjustTokenPrivileges")
)

const (
	tokenAdjustPrivileges = 0x0020
	tokenQuery            = 0x0008
	sePrivilegeEnabled    = 0x0002
	seSystemtimePrivilege = "SeSystemtimePrivilege"
	errNotAllAssigned     = syscall.Errno(1300) // ERROR_NOT_ALL_ASSIGNED
)

type luid struct {
	LowPart  uint32
	HighPart int32
}

type luidAndAttributes struct {
	Luid       luid
	Attributes uint32
}

type tokenPrivileges struct {
	PrivilegeCount uint32
	Privileges     [1]luidAndAttributes
}

type systemtime struct {
	Year         uint16
	Month        uint16
	DayOfWeek    uint16
	Day          uint16
	Hour         uint16
	Minute       uint16
	Second       uint16
	Milliseconds uint16
}

// enableSystemtimePrivilege 启用当前进程令牌中的 SeSystemtimePrivilege 特权。
// 不用 syscall.OpenCurrentProcessToken，因为它只申请 TOKEN_QUERY 权限，
// 无法通过 AdjustTokenPrivileges 修改特权状态。
func enableSystemtimePrivilege() error {
	pseudo, err := syscall.GetCurrentProcess()
	if err != nil {
		return fmt.Errorf("GetCurrentProcess 失败: %v", err)
	}
	var token syscall.Handle
	r1, _, callErr := procOpenProcessToken.Call(
		uintptr(pseudo),
		tokenAdjustPrivileges|tokenQuery,
		uintptr(unsafe.Pointer(&token)),
	)
	if r1 == 0 {
		return fmt.Errorf("OpenProcessToken 失败: %v", callErr)
	}
	defer syscall.CloseHandle(token)

	name, err := syscall.UTF16PtrFromString(seSystemtimePrivilege)
	if err != nil {
		return err
	}
	var privLuid luid
	r1, _, callErr = procLookupPrivilegeValue.Call(
		0,
		uintptr(unsafe.Pointer(name)),
		uintptr(unsafe.Pointer(&privLuid)),
	)
	if r1 == 0 {
		return fmt.Errorf("LookupPrivilegeValue 失败: %v", callErr)
	}

	tp := tokenPrivileges{
		PrivilegeCount: 1,
		Privileges: [1]luidAndAttributes{{
			Luid:       privLuid,
			Attributes: sePrivilegeEnabled,
		}},
	}
	r1, _, callErr = procAdjustTokenPrivileges.Call(
		uintptr(token),
		0,
		uintptr(unsafe.Pointer(&tp)),
		0,
		0,
		0,
	)
	if r1 == 0 {
		return fmt.Errorf("AdjustTokenPrivileges 失败: %v", callErr)
	}
	// 调用成功不代表特权已授予，还需检查 ERROR_NOT_ALL_ASSIGNED
	if errno, ok := callErr.(syscall.Errno); ok && errno == errNotAllAssigned {
		return fmt.Errorf("当前账户没有修改系统时间的特权，请以管理员身份运行")
	}
	return nil
}

// setSystemTime 将系统时钟设置为 t（SetSystemTime 使用 UTC 语义，与 NTP 返回值直接对应）。
func setSystemTime(t time.Time) error {
	if err := enableSystemtimePrivilege(); err != nil {
		return err
	}
	u := t.UTC()
	st := systemtime{
		Year:         uint16(u.Year()),
		Month:        uint16(u.Month()),
		DayOfWeek:    uint16(u.Weekday()),
		Day:          uint16(u.Day()),
		Hour:         uint16(u.Hour()),
		Minute:       uint16(u.Minute()),
		Second:       uint16(u.Second()),
		Milliseconds: uint16(u.Nanosecond() / 1e6),
	}
	r1, _, callErr := procSetSystemTime.Call(uintptr(unsafe.Pointer(&st)))
	if r1 == 0 {
		if errno, ok := callErr.(syscall.Errno); ok && errno == syscall.ERROR_ACCESS_DENIED {
			return fmt.Errorf("没有修改系统时间的权限，请以管理员身份运行")
		}
		return fmt.Errorf("SetSystemTime 失败: %v", callErr)
	}
	return nil
}
