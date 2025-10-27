<div align="center"> 
<img src="./src/icon.png" width="128" height="128" alt="MikaBooM">
<h1>MikaBooM</h1>
</div>

一个跨平台的系统资源监控与调整工具，可以智能地将CPU和内存使用率维持在设定的阈值附近。在寒冷的冬天，让你的设备为你发光发热！

> 注意：Go只支持Windows 7 以上版本，旧版系统请使用[MikaBooM C++](https://github.com/MakotoArai-CN/MikaBooM_CPP)版本，该版本支持Windows 2000/2003/XP/7/8.1/10/11 版本
>
> Windows 7 以上系统以及Linux/MacOS推荐本项目，Go 版本更稳定，跨平台能力更强，程序数据更精确。
>
> **⚠免责申明：**
>
> - 🚫本软件仅用于学习交流，请勿用于非法用途。
> - ⚠高强度运算可能导致硬件过热，寿命缩减，请自行承担风险。

## 功能特性

- ✅ CPU占用率实时监控
- ✅ 内存占用率实时监控
- ✅ 智能负载调整（CPU和内存）
- ✅ 系统托盘实时显示
- ✅ 系统通知支持
- ⭕ 跨平台支持（Windows、Linux、MacOS）
- ✅ 自启动配置
- ✅ YAML配置文件
- ✅ Miku配色命令行界面

## 快速开始

### 安装依赖

```bash
go mod tidy
```

### 编译

```bash
bash build.sh
```

> 目前跨平台无法编译，Linux/Mac尚需适配。
>
> 测试阶段每次编译有效期为2年，过期后需要重新编译。正式版更新时，会加入Github仓库检查更新选项。

## 使用方法

### 基本使用

```bash
# 使用默认配置运行
./MikaBooM

# 指定CPU阈值
./MikaBooM -cpu 80

# 指定内存阈值
./MikaBooM -mem 75

# 指定配置文件
./MikaBooM -c /path/to/config.yaml

# 查看版本信息
./MikaBooM -v

# 查看帮助
./MikaBooM -h
```

### 配置文件

默认配置文件 `config.yaml`：

```yaml
# CPU占用率阈值 (0-100)
cpu_threshold: 70

# 内存占用率阈值 (0-100)
memory_threshold: 70

# 是否自启动
auto_start: true

# 是否显示窗口
show_window: true

# 更新间隔(秒)
update_interval: 2

# 通知设置
notification:
  enabled: true
  cooldown: 60
```

## 系统托盘

程序运行后会在系统托盘显示实时的CPU和内存占用率。右键托盘图标可以：

- 查看当前状态
- 暂停/恢复计算
- 退出程序

## 工作原理

1. **监控阶段**：实时监控系统CPU和内存使用率
2. **判断阶段**：计算除程序自身外的其他程序占用
3. **调整阶段**：
   - 如果其他程序占用低于阈值，启动计算负载，补充到阈值
   - 如果其他程序占用达到阈值，停止计算负载
4. **通知阶段**：状态变化时发送系统通知

## 开发

### 项目结构

```bash
MikaBooM/
├── main.go                 # 主入口
├── config.yaml            # 配置文件
├── go.mod                 # Go模块定义
├── internal/
│   ├── config/           # 配置管理
│   ├── monitor/          # 监控模块
│   ├── worker/           # 负载生成模块
│   ├── tray/             # 系统托盘
│   ├── notify/           # 通知模块
│   ├── sysinfo/          # 系统信息
│   ├── autostart/        # 自启动
│   └── version/          # 版本管理
└── README.md
```

### 依赖库

- `github.com/fatih/color` - 命令行彩色输出
- `github.com/getlantern/systray` - 系统托盘
- `github.com/shirou/gopsutil/v3` - 系统信息获取
- `gopkg.in/yaml.v3` - YAML配置解析
- `github.com/gen2brain/beeep` - 系统通知

## 许可证

本项目遵循[AGPLv3](LICENSE)协议

## issues

[issues](https://github.com/MikaBooM/MikaBooM/issues)

## Todolists

- [ ]支持Android平台
- [ ]支持内存/CPU负载优化（有高负载就要有优化，没问题吧[doge]）
- [ ]支持Linux/Mac

## 版本历史

- v1.0.0
  - 初始版本
  - 支持CPU和内存监控
  - 支持智能负载调整
  - 支持系统托盘
  - 支持跨平台
