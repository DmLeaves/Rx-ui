package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	xrayVersion    = "1.8.24"
	xrayInstallDir = "./bin"
)

// getXrayDownloadURL 获取 Xray 下载链接
func getXrayDownloadURL() string {
	osName := runtime.GOOS
	arch := runtime.GOARCH

	// 映射架构名称
	archMap := map[string]string{
		"amd64": "64",
		"386":   "32",
		"arm64": "arm64-v8a",
		"arm":   "arm32-v7a",
	}

	xrayArch, ok := archMap[arch]
	if !ok {
		xrayArch = arch
	}

	// 构建下载 URL
	filename := fmt.Sprintf("Xray-%s-%s.zip", osName, xrayArch)
	return fmt.Sprintf("https://github.com/XTLS/Xray-core/releases/download/v%s/%s", xrayVersion, filename)
}

// getXrayBinName 获取 Xray 二进制文件名
func getXrayBinName() string {
	if runtime.GOOS == "windows" {
		return "xray.exe"
	}
	return "xray"
}

// getXrayPath 获取 Xray 完整路径
func getXrayPath() string {
	return filepath.Join(xrayInstallDir, getXrayBinName())
}

// isXrayInstalled 检查 Xray 是否已安装
func isXrayInstalled() bool {
	path := getXrayPath()
	_, err := os.Stat(path)
	return err == nil
}

// downloadFile 下载文件
func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败: HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// unzip 解压 zip 文件
func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		// 只解压 xray 二进制和 geoip/geosite 文件
		name := filepath.Base(f.Name)
		if !strings.HasPrefix(name, "xray") && !strings.HasPrefix(name, "geo") {
			continue
		}

		fpath := filepath.Join(dest, name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

// installXray 下载并安装 Xray
func installXray() error {
	// 创建安装目录
	if err := os.MkdirAll(xrayInstallDir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}

	// 下载链接
	url := getXrayDownloadURL()
	zipPath := filepath.Join(xrayInstallDir, "xray.zip")

	fmt.Printf("正在下载 Xray v%s...\n", xrayVersion)
	fmt.Printf("下载地址: %s\n", url)

	// 下载
	if err := downloadFile(url, zipPath); err != nil {
		return fmt.Errorf("下载失败: %v", err)
	}

	fmt.Println("正在解压...")

	// 解压
	if err := unzip(zipPath, xrayInstallDir); err != nil {
		return fmt.Errorf("解压失败: %v", err)
	}

	// 删除 zip 文件
	os.Remove(zipPath)

	// 设置可执行权限
	xrayPath := getXrayPath()
	if runtime.GOOS != "windows" {
		if err := os.Chmod(xrayPath, 0755); err != nil {
			return fmt.Errorf("设置权限失败: %v", err)
		}
	}

	fmt.Printf("Xray 已安装到: %s\n", xrayPath)
	return nil
}

// ensureXrayInstalled 确保 Xray 已安装
func ensureXrayInstalled() error {
	if isXrayInstalled() {
		return nil
	}
	return installXray()
}
