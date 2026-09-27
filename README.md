# NTP 定时校时工具（ntptime）

go的安装路径 D:\alantop_dir\alantop_sde\go120\bin\go.exe

从 `ntp_time.ini` 读取 NTP 服务器地址和校对间隔，定时向 NTP 服务器查询并校准本机系统时间。
使用 Go 1.20.14 编译，支持 Windows 7 / 8 / 10 / 11（含 32 位系统），零第三方依赖。

## 文件说明

| 文件 | 说明 |
| --- | --- |
| `ntp_time_x86.exe` | 32 位版，可运行于 32/64 位 Windows（Win7 优先用这个） |
| `ntp_time_x64.exe` | 64 位版，用于 64 位 Windows |
| `ntp_time.ini` | 配置文件（首次运行若不存在会自动生成默认配置） |
| `ntp_sync.log` | 运行日志（UTF-8 编码，自动生成） |

## 快速使用

1. 编辑 `ntp_time.ini`，设置 NTP 服务器地址和校对间隔
2. **右键"以管理员身份运行"** `ntp_time_x64.exe`（或 32 位系统用 `ntp_time_x86.exe`）

> 修改系统时间需要管理员权限。没有权限时程序会继续运行并在日志中提示，不会自动弹窗提权。

## 命令行参数

| 参数 | 说明 |
| --- | --- |
| `-c 路径` | 指定配置文件路径，默认为程序目录下的 `ntp_time.ini` |
| `-once` | 立即校时一次后退出（适合配合 Windows 计划任务） |
| `-check` | 只查询 NTP 并显示偏差，**不修改系统时钟**（用于测试） |

示例：

```bat
ntp_time_x64.exe -check
```

## 配置项说明（ntp_time.ini）

```ini
# NTP 服务器地址，多个用英文逗号分隔，按顺序尝试，失败的自动跳过
server = ntp.aliyun.com,time.windows.com
# 定时校对间隔（秒），最小 30
interval = 300
# 偏差阈值（秒）：偏差超过该值才校准系统时间；0 表示每次都校准
threshold = 1
# 日志文件名（位于程序目录），留空则只输出到控制台
logfile = ntp_sync.log
```

## 开机自动校时

方式一：计划任务（开机即校时一次，无需登录界面）：

```bat
schtasks /create /tn "NTP校时" /tr "C:\path\ntp_time_x64.exe -once" /sc onstart /ru SYSTEM /rl HIGHEST
```

方式二：随登录启动常驻（定时循环校时）——把 `ntp_time_x64.exe` 的快捷方式放入
`shell:startup` 文件夹，或放一个指向它的启动脚本。

## 构建

双击 `build.bat`，或手动执行（GOROOT 指向本机 Go 1.20.14）：

```bat
set GOROOT=D:\alantop_dir\alantop_sde\go120
set PATH=%GOROOT%\bin;%PATH%
set GOOS=windows
set GOARCH=386 & go build -trimpath -ldflags "-s -w" -o ntp_time_x86.exe .
set GOARCH=amd64 & go build -trimpath -ldflags "-s -w" -o ntp_time_x64.exe .
```

> **注意**：Go 1.21 及以上版本编译的程序无法在 Windows 7 运行，请勿用新版 Go 构建。
> 本程序未使用任何第三方库，因此构建过程不需要联网拉取依赖。
