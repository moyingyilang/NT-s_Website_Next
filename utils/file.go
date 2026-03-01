package utils

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"go-fiber-web/config"
)

// 生成文件MD5
func FileMD5(file *os.File) string {
	hash := md5.New()
	_, _ = io.Copy(hash, file)
	return hex.EncodeToString(hash.Sum(nil))
}

// 保存头像文件
func SaveAvatar(file *os.File, filename string) (string, error) {
	destPath := filepath.Join(config.AvatarDir, filename)
	out, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		return "", err
	}
	return destPath, nil
}

// 保存聊天图片
func SaveChatImage(file *os.File, md5 string, ext string) (string, error) {
	filename := fmt.Sprintf("%s.%s", md5, ext)
	destPath := filepath.Join(config.ChatImageDir, filename)
	out, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		return "", err
	}
	return destPath, nil
}

// 保存压缩包文件
func SavePackage(file *os.File, md5 string, ext string) (string, error) {
	filename := fmt.Sprintf("%s.%s", md5, ext)
	destPath := filepath.Join(config.PackageDir, filename)
	out, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, file)
	if err != nil {
		return "", err
	}
	return destPath, nil
}
