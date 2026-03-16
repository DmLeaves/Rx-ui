# Rx-ui 开发路线图

## 总体策略

**渐进式重构**：保持系统始终可运行，每个检查点都是可部署的稳定版本。

```
Phase 1 → Phase 2 → Phase 3 → Phase 4
 基础升级    后端重构    前端重写    扩展功能
 (2天)      (3天)      (4天)      (持续)
```

---

## Phase 1：基础升级（预计 2 天）

### 目标
升级技术栈基础，确保项目能用现代工具链编译运行。

### 任务清单
- [ ] 1.1 Go 版本升级到 1.22+
- [ ] 1.2 更新 go.mod 所有依赖到最新稳定版
- [ ] 1.3 修复升级后的编译错误
- [ ] 1.4 添加 Makefile（统一构建命令）
- [ ] 1.5 添加 .editorconfig（统一代码风格）
- [ ] 1.6 配置 golangci-lint（代码质量检查）

### ✅ 检查点 1：基础升级完成
```bash
# 验收标准
go build -o rx-ui .           # 编译成功
./rx-ui run                    # 能正常启动
golangci-lint run             # 无严重警告
```

---

## Phase 2：后端重构（预计 3 天）

### 目标
实现分层架构，API RESTful 化，为前后端分离做准备。

### 2.1 分层架构搭建（Day 1）
- [ ] 创建 `internal/` 目录结构
- [ ] 抽象 Repository 接口
- [ ] 迁移 Model 层
- [ ] 实现依赖注入容器

### ✅ 检查点 2.1：分层骨架完成
```bash
# 目录结构正确
ls internal/{model,repository,service,handler}
# 编译通过
go build ./...
```

### 2.2 核心模块迁移（Day 2）
- [ ] 迁移 InboundService（入站管理）
- [ ] 迁移 XrayService（Xray 控制）
- [ ] 迁移 SettingService（系统设置）
- [ ] 迁移 UserService（用户认证）

### ✅ 检查点 2.2：核心服务迁移完成
```bash
# 所有原有功能可用
curl -X POST http://localhost:54321/login
curl http://localhost:54321/xui/inbound/list
```

### 2.3 API RESTful 改造（Day 3）
- [ ] 新建 `/api/v1/` 路由组
- [ ] 实现 RESTful 端点（GET/POST/PUT/DELETE）
- [ ] 统一响应格式 `{code, message, data}`
- [ ] 添加 Swagger 注解
- [ ] 生成 API 文档

### ✅ 检查点 2.3：API 层完成
```bash
# 新 API 可用
curl http://localhost:54321/api/v1/inbounds
curl http://localhost:54321/api/v1/system/status
# Swagger 文档可访问
curl http://localhost:54321/swagger/index.html
```

---

## Phase 3：前端重写（预计 4 天）

### 目标
Vue3 + TypeScript + Vite 现代前端，与后端完全分离。

### 3.1 项目初始化（Day 1）
- [ ] 初始化 Vite + Vue3 + TypeScript
- [ ] 配置 ESLint + Prettier
- [ ] 安装 UI 库（Naive UI / Ant Design Vue 4）
- [ ] 安装 Pinia + Vue Router + Axios
- [ ] 搭建基础布局（Header + Sidebar + Content）

### ✅ 检查点 3.1：前端骨架完成
```bash
cd web && npm run dev
# 能看到基础布局页面
```

### 3.2 核心页面开发（Day 2-3）
- [ ] 登录页 + 认证逻辑
- [ ] 仪表盘（系统状态展示）
- [ ] 入站管理页（列表 + CRUD）
- [ ] 入站表单（协议配置组件化）
- [ ] 设置页

### ✅ 检查点 3.2：核心功能可用
```bash
# 能完成完整操作流程
登录 → 查看状态 → 添加入站 → 编辑入站 → 删除入站 → 修改设置
```

### 3.3 集成与优化（Day 4）
- [ ] API 请求错误处理
- [ ] 加载状态 + 骨架屏
- [ ] 响应式适配（移动端）
- [ ] 构建产物嵌入后端
- [ ] 生产环境配置

### ✅ 检查点 3.3：前端集成完成
```bash
cd web && npm run build
# 访问 http://localhost:54321 看到新 UI
# 所有功能正常工作
```

---

## Phase 4：扩展功能（持续迭代）

### 可选功能池
| 优先级 | 功能 | 复杂度 |
|-------|------|--------|
| P0 | 多用户权限 | 中 |
| P0 | 订阅链接生成 | 低 |
| P1 | 多节点管理 | 高 |
| P1 | API Token 认证 | 低 |
| P2 | Prometheus 监控 | 中 |
| P2 | 流量图表可视化 | 中 |
| P3 | 国际化 i18n | 低 |

### ✅ 检查点 4.x：按需验收
每个功能独立验收，不阻塞主线。

---

## 开发规范

### Git 分支策略
```
main          # 稳定版本
├── dev       # 开发分支
├── feat/*    # 功能分支
└── fix/*     # 修复分支
```

### Commit 规范
```
feat: 新功能
fix: 修复
refactor: 重构
docs: 文档
chore: 杂项
```

### 代码审查要点
1. 编译通过，无 lint 错误
2. 有必要的注释
3. 接口有错误处理
4. 敏感信息不硬编码

---

## 检查点汇总

| # | 检查点 | 验收标准 | 预计时间 |
|---|--------|---------|---------|
| 1 | 基础升级完成 | Go 1.22 编译通过，lint 通过 | Day 2 |
| 2.1 | 分层骨架完成 | internal 目录结构正确 | Day 3 |
| 2.2 | 核心服务迁移 | 原有 API 全部可用 | Day 4 |
| 2.3 | API 层完成 | RESTful API + Swagger | Day 5 |
| 3.1 | 前端骨架完成 | Vite 开发服务器可访问 | Day 6 |
| 3.2 | 核心功能可用 | 完整 CRUD 流程跑通 | Day 8 |
| 3.3 | 前端集成完成 | 生产构建嵌入后端 | Day 9 |

---

## 风险与应对

| 风险 | 应对方案 |
|-----|---------|
| Xray API 变更 | 锁定 xray-core 版本，后续单独升级 |
| 前端协议配置复杂 | 先实现 VMess/VLESS，其他协议渐进添加 |
| 数据库兼容 | 保持 SQLite 表结构不变，用迁移脚本 |
| 老用户升级 | 提供 v1 → v2 迁移文档 |

---

_最后更新：2026-03-16_
