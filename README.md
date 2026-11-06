# pe-ip-tool

在 **WinPE / 离线系统**下，修改另一套 Windows 安装网卡 IP 的工具（读/写目标系统的 `SYSTEM` 注册表蜂巢）。

适用：装机、救援 PE、进不了系统时改静态 IP / 网关 / DNS / 掩码。

## 功能

| 能力 | 说明 |
|------|------|
| 枚举系统 | 自动查找磁盘上的 Windows（需存在 `\Windows\System32\config\SYSTEM`） |
| 读网卡 | 加载目标 `SYSTEM` hive，列出适配器 |
| 改配置 | 查看并修改 IP、掩码、网关、DNS |
| 图形界面 | Windows GUI，操作直观 |

## 下载与使用

1. 到 [Releases](https://github.com/fig3458/pe-ip-tool/releases) 下载 `pe-ip-tool.exe`
2. 在 WinPE 或能访问目标磁盘的本机运行
3. 选择系统 → 选择网卡 → 填写参数 → 应用

## 自行编译

需要 Go 1.21+：

```bat
build.bat
```

产物：`pe-ip-tool.exe`（`CGO_ENABLED=0`，Windows amd64 GUI）。

## 注意

直接改离线注册表，请务必选对系统分区；错误配置可能导致目标系统无法联网。

## 许可证

[MIT](LICENSE)
