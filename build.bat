@echo off
rem NTP 校时程序构建脚本
rem 注意: 必须使用 Go 1.20.x 编译，Go 1.21 及以上版本生成的程序无法在 Windows 7 运行
setlocal
set GOROOT=D:\alantop_dir\alantop_sde\go120
set PATH=%GOROOT%\bin;%PATH%
set GOOS=windows

cd /d %~dp0

echo [1/2] 构建 32 位版本 (兼容 32/64 位 Windows 7)...
set GOARCH=386
go build -trimpath -ldflags "-s -w" -o ntp_time_x86.exe . || goto :err

echo [2/2] 构建 64 位版本...
set GOARCH=amd64
go build -trimpath -ldflags "-s -w" -o ntp_time_x64.exe . || goto :err

echo 构建完成: ntp_time_x86.exe / ntp_time_x64.exe
goto :eof

:err
echo 构建失败
exit /b 1
