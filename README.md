# chait

一个基于 Cobra 的 Golang 命令行工具，用于管理配置数据。

## 功能特点

- 配置数据保存在 `~/.config/chait/config.json` 中
- 支持获取、设置、列出和重置配置
- 使用 Viper 进行配置管理，支持嵌套配置项

## 安装

```bash
go install github.com/plucury/chait@latest
```

## 使用方法

### 基本命令

```bash
# 显示帮助信息
chait --help

# 获取配置值
chait get [key]

# 设置配置值
chait set [key] [value]

# 列出所有配置
chait list

# 重置配置为默认值
chait reset
```

### 示例

```bash
# 设置调试模式
chait set settings.debug true

# 获取版本信息
chait get version

# 列出所有配置
chait list
```

## 开发

### 构建

```bash
go build -o chait
```

### 添加新命令

```bash
go run main.go [command] [args]
```
