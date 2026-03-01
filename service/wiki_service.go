package service

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"go-fiber-web/config"
)

// WikiDoc 定义Wiki文档结构
type WikiDoc struct {
	FileName string `json:"file_name"` // 文件名
	Title    string `json:"title"`    // 文档标题（去扩展名）
	Path     string `json:"path"`     // 文档路径
	Size     int64  `json:"size"`     // 文档大小
}

// 初始化Wiki文档目录
func init() {
	_ = os.MkdirAll(config.WikiDocDir, 0755)
}

// GetDocList 获取Wiki文档列表
func GetDocList() ([]*WikiDoc, error) {
	var docs []*WikiDoc
	// 遍历Wiki文档目录
	err := filepath.Walk(config.WikiDocDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 跳过目录，只处理md文件
		if info.IsDir() {
			return nil
		}
		ext := filepath.Ext(info.Name())
		if strings.ToLower(ext) != ".md" {
			return nil
		}
		// 构造文档信息
		title := strings.TrimSuffix(info.Name(), ext)
		docs = append(docs, &WikiDoc{
			FileName: info.Name(),
			Title:    title,
			Path:     path,
			Size:     info.Size(),
		})
		return nil
	})
	if err != nil {
		return nil, errors.New("获取文档列表失败：" + err.Error())
	}
	return docs, nil
}

// GetDocContent 获取Wiki文档内容（Markdown）
func GetDocContent(fileName string) (string, error) {
	if fileName == "" {
		return "", errors.New("文档名不能为空")
	}
	// 校验文件扩展名
	ext := filepath.Ext(fileName)
	if strings.ToLower(ext) != ".md" {
		fileName += ".md"
	}
	// 文档完整路径
	docPath := filepath.Join(config.WikiDocDir, fileName)
	// 读取文件内容
	content, err := os.ReadFile(docPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errors.New("文档不存在")
		}
		return "", errors.New("读取文档失败：" + err.Error())
	}
	return string(content), nil
}

// UploadDoc 上传Wiki文档（覆盖已有）
func UploadDoc(fileName string, content []byte) error {
	if fileName == "" || len(content) == 0 {
		return errors.New("文档名和内容不能为空")
	}
	ext := filepath.Ext(fileName)
	if strings.ToLower(ext) != ".md" {
		fileName += ".md"
	}
	docPath := filepath.Join(config.WikiDocDir, fileName)
	// 写入文件
	err := os.WriteFile(docPath, content, 0644)
	if err != nil {
		return errors.New("上传文档失败：" + err.Error())
	}
	return nil
}

// DeleteDoc 删除Wiki文档
func DeleteDoc(fileName string) error {
	if fileName == "" {
		return errors.New("文档名不能为空")
	}
	ext := filepath.Ext(fileName)
	if strings.ToLower(ext) != ".md" {
		fileName += ".md"
	}
	docPath := filepath.Join(config.WikiDocDir, fileName)
	// 删除文件
	err := os.Remove(docPath)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("文档不存在")
		}
		return errors.New("删除文档失败：" + err.Error())
	}
	return nil
}
