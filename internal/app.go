package internal

import (
	"Rx-ui/internal/config"
	"Rx-ui/internal/repository"
)

// App 应用容器（依赖注入）
type App struct {
	Config *config.Config
	Repos  *repository.Repositories
}

// NewApp 创建应用实例
func NewApp(cfg *config.Config) (*App, error) {
	app := &App{
		Config: cfg,
	}

	// TODO: 初始化数据库连接
	// TODO: 初始化 Repositories

	return app, nil
}

// Close 关闭应用，释放资源
func (a *App) Close() error {
	// TODO: 关闭数据库连接
	return nil
}
