package service

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"time"

	"Rx-ui/internal/model"
	"Rx-ui/internal/repository"
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

	// 解析证书获取过期时间
	if cert.CertFile != "" {
		expiresAt, err := s.parseCertExpiry(cert.CertFile)
		if err == nil {
			cert.ExpiresAt = expiresAt
		}
	}

	return s.repo.Create(cert)
}

// Update 更新证书
func (s *CertificateService) Update(cert *model.Certificate) error {
	// 重新解析过期时间
	if cert.CertFile != "" {
		expiresAt, err := s.parseCertExpiry(cert.CertFile)
		if err == nil {
			cert.ExpiresAt = expiresAt
		}
	}

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

// parseCertExpiry 解析证书文件获取过期时间
func (s *CertificateService) parseCertExpiry(certFile string) (time.Time, error) {
	data, err := os.ReadFile(certFile)
	if err != nil {
		return time.Time{}, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return time.Time{}, ErrCertInvalid
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}, err
	}

	return cert.NotAfter, nil
}

// RefreshAll 刷新所有证书的过期时间
func (s *CertificateService) RefreshAll() error {
	certs, err := s.repo.FindAll()
	if err != nil {
		return err
	}

	for _, cert := range certs {
		if cert.CertFile != "" {
			expiresAt, err := s.parseCertExpiry(cert.CertFile)
			if err == nil && !expiresAt.Equal(cert.ExpiresAt) {
				cert.ExpiresAt = expiresAt
				s.repo.Update(cert)
			}
		}
	}

	return nil
}
