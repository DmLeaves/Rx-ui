package service

import (
	"errors"

	"Rx-ui/internal/model"
	"Rx-ui/internal/repository"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrInvalidCredential = errors.New("invalid username or password")
	ErrUserExists        = errors.New("user already exists")
)

// UserService 用户认证服务
type UserService struct {
	repo repository.UserRepository
}

// NewUserService 创建用户服务
func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

// Authenticate 验证用户登录
func (s *UserService) Authenticate(username, password string) (*model.User, error) {
	user, err := s.repo.FindByUsername(username)
	if err != nil {
		return nil, ErrInvalidCredential
	}

	// TODO: 使用 bcrypt 验证密码
	if user.Password != password {
		return nil, ErrInvalidCredential
	}

	return user, nil
}

// GetFirstUser 获取第一个用户（用于初始化检查）
func (s *UserService) GetFirstUser() (*model.User, error) {
	return s.repo.GetFirst()
}

// UpdateUser 更新用户信息
func (s *UserService) UpdateUser(id int, username, password string) error {
	user, err := s.repo.FindByID(id)
	if err != nil {
		return ErrUserNotFound
	}

	if username != "" {
		user.Username = username
	}
	if password != "" {
		// TODO: 使用 bcrypt 加密密码
		user.Password = password
	}

	return s.repo.Update(user)
}

// UpdateFirstUser 更新第一个用户（管理员）
func (s *UserService) UpdateFirstUser(username, password string) error {
	user, err := s.repo.GetFirst()
	if err != nil {
		return ErrUserNotFound
	}

	if username != "" {
		user.Username = username
	}
	if password != "" {
		user.Password = password
	}

	return s.repo.Update(user)
}

// CreateUser 创建新用户
func (s *UserService) CreateUser(username, password string) (*model.User, error) {
	// 检查用户是否已存在
	existing, _ := s.repo.FindByUsername(username)
	if existing != nil {
		return nil, ErrUserExists
	}

	user := &model.User{
		Username: username,
		Password: password, // TODO: bcrypt
	}

	if err := s.repo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}
