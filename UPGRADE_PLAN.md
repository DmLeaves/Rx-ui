# x-ui 现代化重构计划

> 目标：Go + Vue3 前后端分离架构，功能解耦，便于扩展

---

## 一、现有功能清单

### 1. 用户系统
| 功能 | API | 位置 |
|-----|-----|-----|
| 登录 | `POST /login` | `controller/index.go` |
| 登出 | `GET /logout` | `controller/index.go` |
| 修改用户 | `POST /xui/setting/updateUser` | `controller/setting.go` |

### 2. 入站规则管理（核心）
| 功能 | API | 位置 |
|-----|-----|-----|
| 列表 | `POST /xui/inbound/list` | `controller/inbound.go` |
| 添加 | `POST /xui/inbound/add` | `controller/inbound.go` |
| 删除 | `POST /xui/inbound/del/:id` | `controller/inbound.go` |
| 更新 | `POST /xui/inbound/update/:id` | `controller/inbound.go` |

**支持的协议：**
- VMess
- VLESS
- Trojan
- Shadowsocks
- Dokodemo-door
- HTTP
- SOCKS

### 3. 系统设置
| 功能 | API | 位置 |
|-----|-----|-----|
| 获取所有设置 | `POST /xui/setting/all` | `controller/setting.go` |
| 更新设置 | `POST /xui/setting/update` | `controller/setting.go` |
| 重启面板 | `POST /xui/setting/restartPanel` | `controller/setting.go` |

### 4. 服务器状态
| 功能 | API | 位置 |
|-----|-----|-----|
| 系统状态 | `POST /server/status` | `controller/server.go` |
| Xray 版本 | `POST /server/getXrayVersion` | `controller/server.go` |
| 安装 Xray | `POST /server/installXray/:version` | `controller/server.go` |

### 5. 定时任务
| 任务 | 文件 |
|-----|-----|
| 流量统计 | `web/job/xray_traffic_job.go` |
| 到期检查 | `web/job/check_inbound_job.go` |
| Xray 存活检测 | `web/job/check_xray_running_job.go` |
| TG 通知 | `web/job/stats_notify_job.go` |

### 6. 数据模型
```
User        - 用户账号
Inbound     - 入站规则（核心）
Setting     - 系统设置（KV 存储）
```

---

## 二、新架构设计

### 目录结构
```
Rx-ui/
├── cmd/
│   └── server/
│       └── main.go              # 入口
│
├── internal/                    # 内部包（不对外暴露）
│   ├── config/                  # 配置管理
│   │   └── config.go
│   │
│   ├── model/                   # 数据模型
│   │   ├── user.go
│   │   ├── inbound.go
│   │   └── setting.go
│   │
│   ├── repository/              # 数据访问层（DAO）
│   │   ├── user_repo.go
│   │   ├── inbound_repo.go
│   │   └── setting_repo.go
│   │
│   ├── service/                 # 业务逻辑层
│   │   ├── auth_service.go
│   │   ├── inbound_service.go
│   │   ├── xray_service.go
│   │   ├── setting_service.go
│   │   └── stats_service.go
│   │
│   ├── handler/                 # HTTP 处理器（Controller）
│   │   ├── auth_handler.go
│   │   ├── inbound_handler.go
│   │   ├── server_handler.go
│   │   └── setting_handler.go
│   │
│   ├── middleware/              # 中间件
│   │   ├── auth.go
│   │   ├── cors.go
│   │   └── logger.go
│   │
│   ├── xray/                    # Xray 内核管理
│   │   ├── process.go           # 进程管理
│   │   ├── config.go            # 配置生成
│   │   └── api.go               # gRPC API 交互
│   │
│   └── scheduler/               # 定时任务
│       ├── scheduler.go
│       └── jobs/
│           ├── traffic_job.go
│           └── expiry_job.go
│
├── pkg/                         # 可复用公共包
│   ├── response/                # 统一响应格式
│   ├── validator/               # 参数校验
│   └── logger/                  # 日志
│
├── api/                         # API 定义（OpenAPI/Swagger）
│   └── openapi.yaml
│
├── web/                         # Vue3 前端（独立项目）
│   ├── package.json
│   ├── vite.config.ts
│   ├── src/
│   │   ├── main.ts
│   │   ├── App.vue
│   │   ├── router/
│   │   ├── stores/              # Pinia 状态管理
│   │   ├── api/                 # API 请求封装
│   │   ├── views/
│   │   │   ├── Login.vue
│   │   │   ├── Dashboard.vue
│   │   │   ├── Inbounds.vue
│   │   │   └── Settings.vue
│   │   └── components/
│   │       ├── InboundForm.vue
│   │       ├── ProtocolConfig/
│   │       │   ├── VMess.vue
│   │       │   ├── VLESS.vue
│   │       │   ├── Trojan.vue
│   │       │   └── ...
│   │       └── common/
│   └── dist/                    # 构建产物（可嵌入后端）
│
├── deploy/
│   ├── Dockerfile
│   ├── docker-compose.yml
│   └── systemd/
│
├── scripts/
│   ├── install.sh
│   └── build.sh
│
├── go.mod
├── go.sum
└── README.md
```

---

## 三、技术选型

### 后端
| 组件 | 旧版 | 新版 | 理由 |
|-----|-----|-----|-----|
| Go 版本 | 1.16 | 1.22+ | 泛型、性能优化 |
| Web 框架 | Gin | Gin / Echo | 保持 Gin 即可 |
| ORM | GORM | GORM v2 | 继续用，升级版本 |
| 配置 | 硬编码 | Viper | 支持多格式、热更新 |
| 日志 | go-logging | Zap / Slog | 高性能结构化日志 |
| 定时任务 | robfig/cron | robfig/cron/v3 | 继续用，升级版本 |
| API 文档 | 无 | Swagger | 自动生成文档 |

### 前端
| 组件 | 旧版 | 新版 |
|-----|-----|-----|
| 框架 | Vue 2（内嵌 HTML） | Vue 3 + TypeScript |
| 构建 | 无 | Vite |
| UI 库 | Ant Design Vue 1.x | Ant Design Vue 4.x / Naive UI |
| 状态管理 | 无 | Pinia |
| HTTP | 原生 fetch | Axios |
| 路由 | 无（多页面） | Vue Router |

---

## 四、分层职责

```
┌─────────────────────────────────────────────────────────┐
│                      Vue3 Frontend                       │
├─────────────────────────────────────────────────────────┤
│                     HTTP API (JSON)                      │
├─────────────────────────────────────────────────────────┤
│  Handler (接收请求、参数校验、调用 Service、返回响应)     │
├─────────────────────────────────────────────────────────┤
│  Service (业务逻辑、事务、多 Repo 协调)                   │
├─────────────────────────────────────────────────────────┤
│  Repository (数据库 CRUD、查询封装)                      │
├─────────────────────────────────────────────────────────┤
│  Model (数据结构定义)                                    │
├─────────────────────────────────────────────────────────┤
│                   SQLite / PostgreSQL                    │
└─────────────────────────────────────────────────────────┘

        ↕ (独立模块)
        
┌─────────────────────────────────────────────────────────┐
│                    Xray Manager                          │
│  - 进程管理（启动/停止/重启）                            │
│  - 配置生成（JSON 拼装）                                 │
│  - 流量统计（gRPC API）                                  │
└─────────────────────────────────────────────────────────┘
```

---

## 五、API 规范（RESTful 改造）

### 旧版 → 新版 对照

| 功能 | 旧 API | 新 API |
|-----|--------|--------|
| 登录 | `POST /login` | `POST /api/v1/auth/login` |
| 登出 | `GET /logout` | `POST /api/v1/auth/logout` |
| 入站列表 | `POST /xui/inbound/list` | `GET /api/v1/inbounds` |
| 添加入站 | `POST /xui/inbound/add` | `POST /api/v1/inbounds` |
| 更新入站 | `POST /xui/inbound/update/:id` | `PUT /api/v1/inbounds/:id` |
| 删除入站 | `POST /xui/inbound/del/:id` | `DELETE /api/v1/inbounds/:id` |
| 系统状态 | `POST /server/status` | `GET /api/v1/system/status` |
| 获取设置 | `POST /xui/setting/all` | `GET /api/v1/settings` |
| 更新设置 | `POST /xui/setting/update` | `PUT /api/v1/settings` |

### 统一响应格式
```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

---

## 六、迁移步骤（分阶段）

### Phase 1：后端重构（2-3 天）
- [ ] 初始化新项目结构
- [ ] 迁移 Model 层
- [ ] 实现 Repository 层
- [ ] 实现 Service 层
- [ ] 实现 Handler 层 + 路由
- [ ] 迁移 Xray 管理模块
- [ ] 迁移定时任务
- [ ] 添加 Swagger 文档

### Phase 2：前端重写（3-4 天）
- [ ] 初始化 Vue3 + Vite 项目
- [ ] 搭建布局框架
- [ ] 实现登录页
- [ ] 实现仪表盘（系统状态）
- [ ] 实现入站管理页（核心）
- [ ] 实现设置页
- [ ] 协议配置组件化

### Phase 3：集成与部署（1 天）
- [ ] 前端构建嵌入后端
- [ ] Docker 镜像构建
- [ ] 安装脚本更新
- [ ] 文档编写

### Phase 4：扩展功能（可选）
- [ ] 多用户权限
- [ ] 多节点管理
- [ ] 订阅链接生成
- [ ] API Token 认证
- [ ] 更多协议支持

---

## 七、解耦设计要点

### 1. 接口抽象
```go
// 定义接口，便于测试和替换实现
type InboundRepository interface {
    FindAll() ([]*model.Inbound, error)
    FindByID(id int) (*model.Inbound, error)
    Create(inbound *model.Inbound) error
    Update(inbound *model.Inbound) error
    Delete(id int) error
}
```

### 2. 依赖注入
```go
// Service 依赖 Repository 接口，不依赖具体实现
type InboundService struct {
    repo   InboundRepository
    xray   XrayManager
    logger *zap.Logger
}

func NewInboundService(repo InboundRepository, xray XrayManager, logger *zap.Logger) *InboundService {
    return &InboundService{repo: repo, xray: xray, logger: logger}
}
```

### 3. 事件驱动（可选）
```go
// 解耦"入站变更"和"Xray 重载"
eventBus.Publish("inbound.created", inbound)
eventBus.Publish("inbound.updated", inbound)
eventBus.Publish("inbound.deleted", id)

// Xray 模块订阅事件
eventBus.Subscribe("inbound.*", func(e Event) {
    xrayService.Reload()
})
```

### 4. 插件化协议支持
```go
// 协议配置生成器接口
type ProtocolConfigGenerator interface {
    Protocol() string
    GenerateSettings(config map[string]any) (string, error)
    ValidateSettings(config map[string]any) error
}

// 注册机制
func init() {
    RegisterProtocol(&VMessGenerator{})
    RegisterProtocol(&VLESSGenerator{})
    RegisterProtocol(&TrojanGenerator{})
}
```

---

## 八、风险与注意事项

1. **数据库兼容**：新版要能读取旧版 SQLite 数据，或提供迁移工具
2. **Xray 版本**：确保兼容最新 Xray-core
3. **Session 迁移**：旧用户升级后需要重新登录
4. **配置迁移**：旧版 config 文件要能平滑迁移

---

## 九、后续可扩展方向

| 方向 | 说明 |
|-----|-----|
| 多节点 | 中心化管理多台服务器 |
| 用户系统 | 注册、套餐、到期提醒 |
| API 开放 | Token 认证，供第三方调用 |
| 订阅系统 | 生成 Clash/V2Ray 订阅链接 |
| 监控告警 | Prometheus + Grafana |
| 国际化 | i18n 多语言 |

---

_文档生成时间：2026-03-16_
