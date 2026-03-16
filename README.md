# Rx-ui

> 基于 x-ui 重构的现代化 Xray 面板，目标是 Go + Vue3 前后端分离架构

[![Go Version](https://img.shields.io/badge/Go-1.16+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-GPL--3.0-green.svg)](LICENSE)

## 项目状态

🚧 **开发中** - 正在从 x-ui 迁移并重构

## 功能特性

- 系统状态监控
- 支持多用户多协议，网页可视化操作
- 支持的协议：VMess、VLESS、Trojan、Shadowsocks、Dokodemo-door、SOCKS、HTTP
- 支持配置更多传输配置
- 流量统计，限制流量，限制到期时间
- 可自定义 Xray 配置模板
- 支持 HTTPS 访问面板（自备域名 + SSL 证书）

## 技术栈

### 当前版本
| 组件 | 技术 |
|-----|-----|
| 后端 | Go 1.16 + Gin |
| 数据库 | SQLite + GORM |
| 前端 | HTML + Vue 2 + Ant Design |
| 代理内核 | Xray-core |

### 重构目标
| 组件 | 技术 |
|-----|-----|
| 后端 | Go 1.22+ + Gin |
| 数据库 | SQLite + GORM v2 |
| 前端 | Vue 3 + TypeScript + Vite |
| API | RESTful + Swagger |

## 快速开始

### 环境要求
- Go 1.16+
- 支持的系统：CentOS 7+、Ubuntu 16+、Debian 8+

### 编译运行

```bash
# 克隆项目
git clone https://github.com/YOUR_USERNAME/Rx-ui.git
cd Rx-ui

# 编译
go build -o rx-ui main.go

# 运行
./rx-ui
```

### Docker 部署

```bash
docker build -t rx-ui .
docker run -d --network=host \
  -v $PWD/db/:/etc/rx-ui/ \
  -v $PWD/cert/:/root/cert/ \
  --name rx-ui --restart=unless-stopped \
  rx-ui
```

## 命令行参数

```bash
# 运行面板
./rx-ui run

# 查看版本
./rx-ui -v

# 设置管理
./rx-ui setting -show              # 显示当前设置
./rx-ui setting -reset             # 重置所有设置
./rx-ui setting -port 54321        # 设置面板端口
./rx-ui setting -username admin    # 设置用户名
./rx-ui setting -password 123456   # 设置密码

# 从 v2-ui 迁移
./rx-ui v2-ui -db /etc/v2-ui/v2-ui.db
```

## 项目结构

```
Rx-ui/
├── main.go              # 程序入口
├── config/              # 配置管理
├── database/            # 数据库操作
├── logger/              # 日志
├── xray/                # Xray 内核管理
├── web/
│   ├── web.go           # Web 服务器
│   ├── controller/      # API 路由
│   ├── service/         # 业务逻辑
│   ├── html/            # 前端页面
│   └── assets/          # 静态资源
├── bin/                 # Xray 二进制
└── Dockerfile
```

## 开发计划

- [x] 从 x-ui fork 并清理代码
- [x] 删除 Telegram 通知功能
- [ ] 升级 Go 版本到 1.22+
- [ ] 后端分层重构（Handler → Service → Repository）
- [ ] RESTful API 改造
- [ ] Vue 3 + TypeScript 前端重写
- [ ] Swagger API 文档
- [ ] 多节点管理（扩展功能）

## 致谢

本项目基于 [vaxilu/x-ui](https://github.com/vaxilu/x-ui) 开发，感谢原作者的贡献。

## 许可证

[GPL-3.0](LICENSE)
