package service

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go-fiber-web/config"
	"go-fiber-web/model"
	"gorm.io/gorm"
)

const (
	MaxFileSize   = 500 * 1024 * 1024 // 500MB 最大文件限制
	MinChunkSize  = 1 * 1024 * 1024   // 1MB 最小分块
	MaxChunkSize  = 50 * 1024 * 1024  // 50MB 最大分块
	TempChunkDir  = "./storage/temp"  // 临时分块目录
)

// 允许的压缩包格式
var AllowedExtensions = []string{"zip", "rar", "7z", "tar", "gz", "bz2", "xz", "tgz"}
var AllowedMimeTypes = []string{
	"application/zip", "application/x-zip-compressed",
	"application/rar", "application/x-rar-compressed",
	"application/x-7z-compressed", "application/x-tar",
	"application/gzip", "application/x-gzip", "application/x-bzip2",
	"application/x-xz", "application/x-tgz",
}

// 初始化临时目录
func init() {
	_ = os.MkdirAll(TempChunkDir, 0755)
	_ = os.MkdirAll(config.PackageDir, 0755)
}

// 生成唯一文件ID
func generateFileID() string {
	return fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%d_%d", time.Now().UnixNano(), os.Getpid()))))
}

// 校验文件扩展名
func validateFileExt(ext string) bool {
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	for _, e := range AllowedExtensions {
		if ext == e {
			return true
		}
	}
	return false
}

// 校验MIME类型
func validateMimeType(mime string) bool {
	for _, m := range AllowedMimeTypes {
		if mime == m {
			return true
		}
	}
	return false
}

// 检查文件是否已存在（通过MD5）
func CheckFileExistByMD5(md5Str string) (*model.File, error) {
	var file model.File
	err := model.DB.Where("md5 = ?", md5Str).First(&file).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, errors.New("查询文件失败")
	}
	return &file, nil
}

// UploadChunk 上传文件分块
func UploadChunk(userID, fileID string, chunkIndex, totalChunks int, chunkFile *os.File, fileName string, fileSize, chunkSize int64) error {
	// 基础参数校验
	if fileID == "" || chunkIndex < 0 || totalChunks <= 0 || chunkFile == nil || fileName == "" {
		return errors.New("参数错误，不能为空")
	}
	if fileSize > MaxFileSize {
		return fmt.Errorf("文件超过最大限制：%dMB", MaxFileSize/1024/1024)
	}
	if chunkSize < MinChunkSize || chunkSize > MaxChunkSize {
		return fmt.Errorf("分块大小需在%dMB~%dMB之间", MinChunkSize/1024/1024, MaxChunkSize/1024/1024)
	}
	// 校验文件扩展名
	ext := filepath.Ext(fileName)
	if !validateFileExt(ext) {
		return errors.New("不支持的文件类型，仅允许：" + strings.Join(AllowedExtensions, ", "))
	}

	// 保存分块到临时目录
	chunkPath := filepath.Join(TempChunkDir, fmt.Sprintf("%s_chunk_%d.part", fileID, chunkIndex))
	out, err := os.Create(chunkPath)
	if err != nil {
		return errors.New("创建分块文件失败：" + err.Error())
	}
	defer out.Close()

	// 写入分块内容
	_, err = io.Copy(out, chunkFile)
	if err != nil {
		_ = os.Remove(chunkPath) // 写入失败删除临时文件
		return errors.New("写入分块失败：" + err.Error())
	}

	return nil
}

// MergeChunk 合并分块为完整文件
func MergeChunk(userID, fileID, fileName string, totalChunks int) (*model.File, error) {
	// 校验分块是否完整
	for i := 0; i < totalChunks; i++ {
		chunkPath := filepath.Join(TempChunkDir, fmt.Sprintf("%s_chunk_%d.part", fileID, i))
		if _, err := os.Stat(chunkPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("缺少分块 %d，请重新上传", i)
		}
	}

	// 校验文件扩展名
	ext := filepath.Ext(fileName)
	if ext == "" || !validateFileExt(ext) {
		return nil, errors.New("文件扩展名非法")
	}
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))

	// 合并分块 + 计算MD5 + 统计总大小
	hash := md5.New()
	fileIDFinal := generateFileID()
	finalFileName := fmt.Sprintf("%s.%s", fileIDFinal, ext)
	finalPath := filepath.Join(config.PackageDir, finalFileName)

	// 创建最终文件
	out, err := os.Create(finalPath)
	if err != nil {
		return nil, errors.New("创建最终文件失败：" + err.Error())
	}
	defer out.Close()

	var totalSize int64
	// 遍历分块合并
	for i := 0; i < totalChunks; i++ {
		chunkPath := filepath.Join(TempChunkDir, fmt.Sprintf("%s_chunk_%d.part", fileID, i))
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			_ = os.Remove(finalPath) // 失败删除最终文件
			return nil, fmt.Errorf("读取分块 %d 失败：%s", i, err.Error())
		}

		// 计算MD5
		chunkHash, err := io.Copy(hash, chunkFile)
		if err != nil {
			_ = chunkFile.Close()
			_ = os.Remove(finalPath)
			return nil, errors.New("计算文件MD5失败：" + err.Error())
		}
		totalSize += chunkHash

		// 回到分块开头，写入最终文件
		_, _ = chunkFile.Seek(0, io.SeekStart)
		_, err = io.Copy(out, chunkFile)
		_ = chunkFile.Close()
		if err != nil {
			_ = os.Remove(finalPath)
			return nil, fmt.Errorf("合并分块 %d 失败：%s", i, err.Error())
		}

		// 删除临时分块
		_ = os.Remove(chunkPath)
	}

	// 生成MD5字符串
	md5Str := hex.EncodeToString(hash.Sum(nil))
	// 检查文件是否已存在（避免重复上传）
	existFile, err := CheckFileExistByMD5(md5Str)
	if err != nil {
		_ = os.Remove(finalPath)
		return nil, err
	}
	if existFile != nil {
		_ = os.Remove(finalPath) // 删除重复文件
		return existFile, nil
	}

	// 创建文件记录到数据库
	file := &model.File{
		ID:           fileIDFinal,
		OriginalName: fileName,
		Path:         finalPath,
		Size:         totalSize,
		Type:         model.FilePackage,
		MD5:          md5Str,
		UserID:       userID,
	}
	if err := model.DB.Create(file).Error; err != nil {
		_ = os.Remove(finalPath)
		return nil, errors.New("保存文件记录失败：" + err.Error())
	}

	return file, nil
}

// GetFileByID 根据文件ID获取文件信息
func GetFileByID(fileID string) (*model.File, error) {
	var file model.File
	err := model.DB.Where("id = ?", fileID).First(&file).Error
	if err == gorm.ErrRecordNotFound {
		return nil, errors.New("文件不存在")
	}
	if err != nil {
		return nil, errors.New("查询文件失败：" + err.Error())
	}
	return &file, nil
}

// GetUserFiles 获取当前用户上传的所有文件
func GetUserFiles(userID string) ([]*model.File, error) {
	var files []*model.File
	err := model.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&files).Error
	if err != nil {
		return nil, errors.New("查询文件列表失败：" + err.Error())
	}
	return files, nil
}
