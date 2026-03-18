package service

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"rxui/internal/model"
	"rxui/internal/repository"
)

var (
	ErrCertNotFound     = errors.New("certificate not found")
	ErrCertExpired      = errors.New("certificate has expired")
	ErrCertInvalid      = errors.New("invalid certificate")
	ErrCertDomainExists = errors.New("certificate for this domain already exists")
)

// CertificateService 证书管理服务（方向2扩展）
type CertificateService struct {
	repo repository.CertificateRepository
}

// NewCertificateService 创建证书服务
func NewCertificateService(repo repository.CertificateRepository) *CertificateService {
	return &CertificateService{repo: repo}
}

// GetAll 获取所有证书
func (s *CertificateService) GetAll() ([]*model.Certificate, error) {
	return s.repo.FindAll()
}

// GetByID 根据ID获取证书
func (s *CertificateService) GetByID(id int) (*model.Certificate, error) {
	cert, err := s.repo.FindByID(id)
	if err != nil {
		return nil, ErrCertNotFound
	}
	return cert, nil
}

// GetByDomain 根据域名获取证书
func (s *CertificateService) GetByDomain(domain string) (*model.Certificate, error) {
	return s.repo.FindByDomain(domain)
}

// GetExpiring 获取即将过期的证书
func (s *CertificateService) GetExpiring(days int) ([]*model.Certificate, error) {
	return s.repo.FindExpiring(days)
}

// Create 创建证书记录
func (s *CertificateService) Create(cert *model.Certificate) error {
	// 检查域名是否已存在
	existing, _ := s.repo.FindByDomain(cert.Domain)
	if existing != nil {
		return ErrCertDomainExists
	}

	if err := s.ensureCertificateFiles(cert); err != nil {
		return err
	}
	applyExpiresAt(cert, s)

	return s.repo.Create(cert)
}

// Update 更新证书
func (s *CertificateService) Update(cert *model.Certificate) error {
	if err := s.ensureCertificateFiles(cert); err != nil {
		return err
	}
	applyExpiresAt(cert, s)

	return s.repo.Update(cert)
}

// Delete 删除证书
func (s *CertificateService) Delete(id int) error {
	return s.repo.Delete(id)
}

// CheckValidity 检查证书有效性
func (s *CertificateService) CheckValidity(cert *model.Certificate) error {
	if cert.IsExpired() {
		return ErrCertExpired
	}
	return nil
}

func sanitizeDomainForFilename(domain string) string {
	d := strings.TrimSpace(domain)
	if d == "" {
		d = "cert"
	}
	re := regexp.MustCompile(`[^a-zA-Z0-9._-]`)
	d = re.ReplaceAllString(d, "_")
	return d
}

func (s *CertificateService) ensureCertificateFiles(cert *model.Certificate) error {
	// 如果已经提供了文件路径，直接使用
	if strings.TrimSpace(cert.CertFile) != "" && strings.TrimSpace(cert.KeyFile) != "" {
		return nil
	}

	// 直接输入模式：将内容落盘为文件，供入站 TLS 复用
	if strings.TrimSpace(cert.CertContent) == "" || strings.TrimSpace(cert.KeyContent) == "" {
		return nil
	}

	certDir := filepath.Join("data", "certs")
	if err := os.MkdirAll(certDir, 0o755); err != nil {
		return fmt.Errorf("创建证书目录失败: %w", err)
	}

	base := fmt.Sprintf("%s-%d", sanitizeDomainForFilename(cert.Domain), time.Now().Unix())
	certPath := filepath.Join(certDir, base+".crt")
	keyPath := filepath.Join(certDir, base+".key")

	if err := os.WriteFile(certPath, []byte(strings.TrimSpace(cert.CertContent)+"\n"), 0o644); err != nil {
		return fmt.Errorf("写入证书文件失败: %w", err)
	}
	if err := os.WriteFile(keyPath, []byte(strings.TrimSpace(cert.KeyContent)+"\n"), 0o600); err != nil {
		return fmt.Errorf("写入私钥文件失败: %w", err)
	}

	cert.CertFile = certPath
	cert.KeyFile = keyPath
	return nil
}

func applyExpiresAt(cert *model.Certificate, s *CertificateService) {
	if strings.TrimSpace(cert.CertFile) != "" {
		expiresAt, err := s.parseCertExpiry(cert.CertFile)
		if err == nil {
			cert.ExpiresAt = expiresAt
			return
		}
	}
	if strings.TrimSpace(cert.CertContent) != "" {
		expiresAt, err := s.parseCertExpiryFromContent(cert.CertContent)
		if err == nil {
			cert.ExpiresAt = expiresAt
		}
	}
}

func (s *CertificateService) parseCertExpiryFromContent(certContent string) (time.Time, error) {
	block, _ := pem.Decode([]byte(certContent))
	if block == nil {
		return time.Time{}, ErrCertInvalid
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}, err
	}
	return cert.NotAfter, nil
}

// parseCertExpiry 解析证书文件获取过期时间
func (s *CertificateService) parseCertExpiry(certFile string) (time.Time, error) {
	data, err := os.ReadFile(certFile)
	if err != nil {
		return time.Time{}, err
	}

	return s.parseCertExpiryFromContent(string(data))
}

// RefreshAll 刷新所有证书的过期时间
func (s *CertificateService) RefreshAll() error {
	certs, err := s.repo.FindAll()
	if err != nil {
		return err
	}

	for _, cert := range certs {
		old := cert.ExpiresAt
		applyExpiresAt(cert, s)
		if !cert.ExpiresAt.Equal(old) && !cert.ExpiresAt.IsZero() {
			s.repo.Update(cert)
		}
	}

	return nil
}
