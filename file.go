package main

import (
    "bytes"            // 用于 bytes.Repeat
    "crypto/aes"       // 标准库，不是 golang.org/x/crypto/aes
    "crypto/cipher"    // 标准库
    "crypto/md5"
    "crypto/rand"
    "encoding/hex"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "regexp"
    "strconv"
    "strings"
    "time"
)

// ---------------------- 头像上传（完全复刻原PHP handleUploadAvatar）----------------------
func handleUploadAvatar(w http.ResponseWriter, r *http.Request) {
	// 验证登录和CSRF（复刻原PHP逻辑）
	session, err := sessionStore.Get(r, "ntc-session")
	if err != nil || session.Values["user_id"] == nil {
		writeJSON(w, map[string]any{"success": false, "error": "未登录"})
		return
	}
	if !checkCSRF(r, session) {
		writeJSON(w, map[string]any{"success": false, "error": "CSRF令牌无效"})
		return
	}

	// 解析文件（复刻原PHP $_FILES['avatar']）
	file, _, err := r.FormFile("avatar")
	if err != nil {
		writeJSON(w, map[string]any{"success": false, "error": "没有文件"})
		return
	}
	defer file.Close()

	userID := session.Values["user_id"].(string)
	avatarPath, err := saveAvatar(file, userID)
	if err != nil {
		writeJSON(w, map[string]any{"success": false, "error": err.Error()})
		return
	}

	// 更新用户头像（复刻原PHP逻辑）
	users := getUsers()
	for i := range users {
		if users[i].ID == userID {
			users[i].Avatar = avatarPath
			break
		}
	}
	if err := saveUsers(users); err != nil {
		writeJSON(w, map[string]any{"success": false, "error": "保存头像失败"})
		return
	}

	writeJSON(w, map[string]any{
		"success": true,
		"path":    avatarPath,
	})
}

// 保存头像（复刻原PHP handleAvatarUpload）
func saveAvatar(file io.Reader, userID string) (string, error) {
	// 验证文件大小
	fileInfo, err := file.(io.Seeker).Seek(0, io.SeekEnd)
	if err != nil {
		return "", errors.New("获取文件大小失败")
	}
	if fileInfo > MAX_AVATAR_SIZE {
		return "", errors.New("图片不能超过2MB")
	}
	file.(io.Seeker).Seek(0, io.SeekStart)

	// 验证文件类型（复刻原PHP MIME类型校验）
	buf := make([]byte, 512)
	if _, err := file.Read(buf); err != nil {
		return "", errors.New("读取文件失败")
	}
	file.(io.Seeker).Seek(0, io.SeekStart)

	mimeType := http.DetectContentType(buf)
	allowedMime := []string{"image/jpeg", "image/png", "image/gif", "image/webp"}
	if !contains(allowedMime, mimeType) {
		return "", errors.New("只允许上传JPG、PNG、GIF、WEBP格式的图片")
	}

	// 生成文件名（复刻原PHP uniqid）
	ext := getFileExt(mimeType)
	filename := fmt.Sprintf("%s_%d.%s", userID, time.Now().UnixMicro(), ext)
	destPath := filepath.Join(AVATAR_DIR, filename)

	// 保存文件（复刻原PHP move_uploaded_file）
	dst, err := os.Create(destPath)
	if err != nil {
		return "", errors.New("保存失败")
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return "", errors.New("保存失败")
	}

	// 返回相对路径（与原PHP一致）
	return "data/avatars/" + filename, nil
}

// ---------------------- 聊天图片上传（完全复刻原PHP handleUploadImage）----------------------
func handleUploadImage(w http.ResponseWriter, r *http.Request) {
	// 验证登录和CSRF（复刻原PHP逻辑）
	session, err := sessionStore.Get(r, "ntc-session")
	if err != nil || session.Values["user_id"] == nil {
		writeJSON(w, map[string]any{"success": false, "error": "未登录"})
		return
	}
	if !checkCSRF(r, session) {
		writeJSON(w, map[string]any{"success": false, "error": "CSRF令牌无效"})
		return
	}

	// 解析文件（复刻原PHP $_FILES['image']）
	file, fileHeader, err := r.FormFile("image")
	if err != nil {
		writeJSON(w, map[string]any{"success": false, "error": "没有文件"})
		return
	}
	defer file.Close()

	// 验证文件大小（复刻原PHP 10MB限制）
	if fileHeader.Size > MAX_IMAGE_SIZE {
		writeJSON(w, map[string]any{"success": false, "error": "图片不能超过10MB"})
		return
	}

	// 验证文件类型（复刻原PHP MIME校验）
	buf := make([]byte, 512)
	if _, err := file.Read(buf); err != nil {
		writeJSON(w, map[string]any{"success": false, "error": "读取文件失败"})
		return
	}
	file.Seek(0, io.SeekStart)

	mimeType := http.DetectContentType(buf)
	allowedMime := []string{"image/jpeg", "image/png", "image/gif", "image/webp"}
	if !contains(allowedMime, mimeType) {
		writeJSON(w, map[string]any{"success": false, "error": "只允许上传图片"})
		return
	}

	// 计算MD5（复刻原PHP md5_file）
	md5Hash := md5.New()
	if _, err := io.Copy(md5Hash, file); err != nil {
		writeJSON(w, map[string]any{"success": false, "error": "计算MD5失败"})
		return
	}
	fileMD5 := hex.EncodeToString(md5Hash.Sum(nil))
	file.Seek(0, io.SeekStart)

	// 保存文件（复刻原PHP逻辑：已存在则跳过）
	destPath := filepath.Join(UPLOAD_DIR, fileMD5)
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		dst, err := os.Create(destPath)
		if err != nil {
			writeJSON(w, map[string]any{"success": false, "error": "保存文件失败"})
			return
		}
		defer dst.Close()
		if _, err := io.Copy(dst, file); err != nil {
			writeJSON(w, map[string]any{"success": false, "error": "保存文件失败"})
			return
		}
	}

	// 更新FILE_NAME_JSON（复刻原PHP逻辑）
	updateFileMap(fileHeader.Filename, fileMD5, mimeType)

	writeJSON(w, map[string]any{
		"success": true,
		"fileId":  fileMD5,
	})
}

// ---------------------- 聊天图片获取（完全复刻原PHP handleGetImage）----------------------
func handleGetImage(w http.ResponseWriter, r *http.Request) {
	// 验证登录（复刻原PHP逻辑）
	session, err := sessionStore.Get(r, "ntc-session")
	if err != nil || session.Values["user_id"] == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// 验证文件ID（复刻原PHP 32位十六进制校验）
	fileID := strings.TrimSpace(r.FormValue("file"))
	if !regexp.MustCompile(`^[a-f0-9]{32}$`).MatchString(fileID) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// 读取文件（复刻原PHP逻辑）
	filePath := filepath.Join(UPLOAD_DIR, fileID)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// 获取MIME类型（复刻原PHP从FILE_NAME_JSON读取）
	mimeType := "image/jpeg"
	fileMap := getFileMap()
	for _, item := range fileMap {
		if item.MD5 == fileID {
			mimeType = item.Mime
			break
		}
	}

	// 返回文件（复刻原PHP readfile）
	w.Header().Set("Content-Type", mimeType)
	http.ServeFile(w, r, filePath)
}

// ---------------------- 压缩包上传/下载（完全复刻原PHP upF目录逻辑）----------------------
func handleUploadFile(w http.ResponseWriter, r *http.Request) {
	// 验证登录（复刻原PHP session验证）
	session, err := sessionStore.Get(r, "ntc-session")
	if err != nil || session.Values["user_id"] == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	switch r.Method {
	case "GET":
		handleFileDownload(w, r) // 下载/列表展示
	case "POST":
		handleFileChunkUpload(w, r) // 分块上传
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// 压缩包分块上传（完全复刻原PHP分块上传逻辑）
func handleFileChunkUpload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 解析表单（复刻原PHP $_POST/$_FILES）
	if err := r.ParseMultipartForm(MAX_CHUNK_SIZE); err != nil {
		writeJSON(w, map[string]any{"success": false, "error": "表单解析失败"})
		return
	}

	// 获取上传参数（复刻原PHP参数名）
	fileName := strings.TrimSpace(r.PostForm.Get("fileName"))
	chunkIndex := strings.TrimSpace(r.PostForm.Get("chunkIndex"))
	totalChunks := strings.TrimSpace(r.PostForm.Get("totalChunks"))
	fileMD5 := strings.TrimSpace(r.PostForm.Get("fileMD5"))
	chunkFile, _, err := r.FormFile("chunkFile")
	if err != nil || fileName == "" || chunkIndex == "" || totalChunks == "" || fileMD5 == "" {
		writeJSON(w, map[string]any{"success": false, "error": "参数不足"})
		return
	}
	defer chunkFile.Close()

	// 转换参数类型（复刻原PHP类型转换）
	chunkIdx, err := strconv.Atoi(chunkIndex)
	totalChunk, err := strconv.Atoi(totalChunks)
	if err != nil || chunkIdx < 0 || totalChunk <= 0 || chunkIdx >= totalChunk {
		writeJSON(w, map[string]any{"success": false, "error": "参数格式错误"})
		return
	}

	// 验证文件扩展名（复刻原PHP ALLOWED_EXTENSIONS）
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == "" || !contains(ALLOWED_EXTENSIONS, ext[1:]) {
		writeJSON(w, map[string]any{"success": false, "error": "不允许的文件格式"})
		return
	}

	// 验证文件MD5格式（复刻原PHP校验）
	if !regexp.MustCompile(`^[a-f0-9]{32}$`).MatchString(fileMD5) {
		writeJSON(w, map[string]any{"success": false, "error": "MD5格式错误"})
		return
	}

	// 临时目录（复刻原PHP temp目录逻辑）
	tempDir := filepath.Join(UPLOAD_DIR, "temp", fileMD5)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		writeJSON(w, map[string]any{"success": false, "error": "创建临时目录失败"})
		return
	}

	// 保存分块（复刻原PHP分块命名逻辑）
	chunkPath := filepath.Join(tempDir, fmt.Sprintf("chunk_%d", chunkIdx))
	dst, err := os.Create(chunkPath)
	if err != nil {
		writeJSON(w, map[string]any{"success": false, "error": "保存分块失败"})
		return
	}
	defer dst.Close()

	// 验证分块大小（复刻原PHP MIN_CHUNK_SIZE/MAX_CHUNK_SIZE）
	chunkSize, err := io.Copy(dst, chunkFile)
	if err != nil {
		writeJSON(w, map[string]any{"success": false, "error": "保存分块失败"})
		return
	}
	if chunkSize < MIN_CHUNK_SIZE && chunkIdx != totalChunk-1 {
		writeJSON(w, map[string]any{"success": false, "error": "分块过小（最小1MB）"})
		return
	}

	// 检查是否所有分块上传完成（复刻原PHP逻辑）
	chunkFiles, err := filepath.Glob(filepath.Join(tempDir, "chunk_*"))
	if err != nil {
		writeJSON(w, map[string]any{"success": false, "error": "检查分块失败"})
		return
	}
	if len(chunkFiles) != totalChunk {
		writeJSON(w, map[string]any{"success": true, "finished": false, "chunkIndex": chunkIdx})
		return
	}

	// 合并分块（复刻原PHP合并逻辑）
	finalPath := filepath.Join(UPLOAD_DIR, fileMD5+"_"+fileName)
	finalFile, err := os.Create(finalPath)
	if err != nil {
		writeJSON(w, map[string]any{"success": false, "error": "创建最终文件失败"})
		return
	}
	defer finalFile.Close()

	for i := 0; i < totalChunk; i++ {
		chunkPath := filepath.Join(tempDir, fmt.Sprintf("chunk_%d", i))
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			writeJSON(w, map[string]any{"success": false, "error": "读取分块失败"})
			return
		}
		if _, err := io.Copy(finalFile, chunkFile); err != nil {
			chunkFile.Close()
			writeJSON(w, map[string]any{"success": false, "error": "合并分块失败"})
			return
		}
		chunkFile.Close()
		os.Remove(chunkPath) // 删除临时分块
	}
	os.RemoveAll(tempDir) // 删除临时目录

	// 验证最终文件MD5（复刻原PHP校验）
	finalMD5, err := calculateFileMD5(finalPath)
	if err != nil || finalMD5 != fileMD5 {
		os.Remove(finalPath)
		writeJSON(w, map[string]any{"success": false, "error": "MD5校验失败"})
		return
	}

	// 更新文件映射（复刻原PHP逻辑）
	updateFileMap(fileName, fileMD5, getMIMEType(fileName))

	writeJSON(w, map[string]any{
		"success": true,
		"finished": true,
		"filePath": finalPath,
	})
}

// 压缩包下载/列表（完全复刻原PHP逻辑）
func handleFileDownload(w http.ResponseWriter, r *http.Request) {
	// 解析参数（复刻原PHP逻辑）
	fileMD5 := strings.TrimSpace(r.FormValue("md5"))
	action := strings.TrimSpace(r.FormValue("action"))

	if action == "download" && fileMD5 != "" {
		// 下载文件（复刻原PHP下载逻辑）
		fileMap := getFileMap()
		var targetFile string
		for _, item := range fileMap {
			if item.MD5 == fileMD5 {
				targetFile = filepath.Join(UPLOAD_DIR, fileMD5+"_"+item.Original)
				break
			}
		}
		if targetFile == "" || !fileExists(targetFile) {
			http.NotFound(w, r)
			return
		}

		// 响应下载头（复刻原PHP逻辑）
		fileName := filepath.Base(targetFile)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
		http.ServeFile(w, r, targetFile)
		return
	}

	// 展示文件列表（复刻原PHP页面逻辑）
	fileMap := getFileMap()
	html := generateFileListHTML(fileMap)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// ---------------------- 加密/解密（完全复刻原PHP aes-256-cbc逻辑）----------------------
// 加密消息（复刻原PHP encryptUserData）
func encryptUserData(data []Message, key []byte) ([]byte, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	// AES-256-CBC 加密（复刻原PHP openssl_encrypt）
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// 生成16字节IV（复刻原PHP random_bytes）
	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return nil, err
	}

	// PKCS7填充（复刻原PHP默认填充）
	paddedData := pkcs7Pad(jsonData, aes.BlockSize)

	// CBC模式加密
	mode := cipher.NewCBCEncrypter(block, iv)
	ciphertext := make([]byte, len(paddedData))
	mode.CryptBlocks(ciphertext, paddedData)

	// 返回 IV+密文（复刻原PHP格式）
	return append(iv, ciphertext...), nil
}

// 解密消息（复刻原PHP decryptUserData）
func decryptUserData(encrypted []byte, key []byte) ([]Message, error) {
	// 解析IV和密文（复刻原PHP格式）
	if len(encrypted) < aes.BlockSize {
		return nil, errors.New("数据过短")
	}
	iv := encrypted[:aes.BlockSize]
	ciphertext := encrypted[aes.BlockSize:]

	// AES-256-CBC 解密
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	// 去除PKCS7填充
	unpaddedData, err := pkcs7Unpad(plaintext)
	if err != nil {
		return nil, err
	}

	// 解析JSON
	var messages []Message
	if err := json.Unmarshal(unpaddedData, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

// ---------------------- 文件操作辅助函数（完全复刻原PHP逻辑）----------------------
// 获取文件扩展名
func getFileExt(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	default:
		return "bin"
	}
}

// 获取文件MIME类型（复刻原PHP逻辑）
func getMIMEType(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".zip":
		return "application/zip"
	case ".rar":
		return "application/x-rar-compressed"
	case ".7z":
		return "application/x-7z-compressed"
	case ".tar":
		return "application/x-tar"
	case ".gz":
		return "application/gzip"
	case ".bz2":
		return "application/x-bzip2"
	case ".xz":
		return "application/x-xz"
	case ".tgz":
		return "application/x-gzip"
	default:
		return "application/octet-stream"
	}
}

// 更新文件映射（复刻原PHP FILE_NAME_JSON操作）
func updateFileMap(originalName, md5, mime string) {
	fileMap := getFileMap()
	fileMap = append(fileMap, FileMapItem{
		Original: originalName,
		MD5:      md5,
		Mime:     mime,
	})
	data, _ := json.Marshal(fileMap)
	os.WriteFile(FILE_NAME_JSON, data, 0644)
}

// 获取文件映射（复刻原PHP FILE_NAME_JSON读取）
func getFileMap() []FileMapItem {
	data, err := os.ReadFile(FILE_NAME_JSON)
	if err != nil {
		return []FileMapItem{}
	}
	var fileMap []FileMapItem
	json.Unmarshal(data, &fileMap)
	return fileMap
}

// 计算文件MD5（复刻原PHP md5_file）
func calculateFileMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	md5Hash := md5.New()
	if _, err := io.Copy(md5Hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(md5Hash.Sum(nil)), nil
}

// 检查文件是否存在（复刻原PHP file_exists）
func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

// 生成文件列表HTML（复刻原PHP upF页面）
func generateFileListHTML(fileMap []FileMapItem) string {
	html := `<!DOCTYPE html>
<html lang="zh">
<head>
    <meta charset="UTF-8">
    <title>压缩包管理</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; font-family: 'Microsoft YaHei'; }
        body { background: #f5f7fa; padding: 20px; }
        .container { max-width: 1200px; margin: 0 auto; background: #fff; padding: 30px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { margin-bottom: 30px; color: #333; }
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 12px; text-align: left; border-bottom: 1px solid #eee; }
        th { background: #f9f9f9; font-weight: bold; }
        a { color: #07c160; text-decoration: none; }
        a:hover { text-decoration: underline; }
        .upload-btn { display: inline-block; padding: 10px 20px; background: #07c160; color: #fff; border-radius: 4px; text-decoration: none; margin-bottom: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>压缩包列表</h1>
        <a href="/" class="upload-btn">返回主页</a>
        <table>
            <tr>
                <th>文件名</th>
                <th>MD5</th>
                <th>操作</th>
            </tr>`

	for _, item := range fileMap {
		html += fmt.Sprintf(`
            <tr>
                <td>%s</td>
                <td>%s</td>
                <td><a href="?action=download&md5=%s">下载</a></td>
            </tr>`, item.Original, item.MD5, item.MD5)
	}

	html += `
        </table>
    </div>
</body>
</html>`
	return html
}

// PKCS7填充（复刻原PHP openssl_encrypt默认填充）
func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

// PKCS7去填充（复刻原PHP openssl_decrypt默认填充）
func pkcs7Unpad(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("无效数据")
	}
	padding := int(data[length-1])
	if padding > length {
		return nil, errors.New("填充错误")
	}
	return data[:length-padding], nil
}

// 检查切片是否包含指定元素
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
