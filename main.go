package main

import (
    "encoding/json"
    "net/http"
    "os"
    "path/filepath"

    "github.com/gorilla/sessions"
)

// 原PHP常量完全复刻（无任何修改）
const (
	DATA_DIR         = "./data"
	USERS_FILE       = DATA_DIR + "/users.json"
	AVATAR_DIR       = DATA_DIR + "/avatars"
	UPLOAD_DIR       = DATA_DIR + "/upFile"
	FILE_NAME_JSON   = UPLOAD_DIR + "/FileName/FileN.json"
	FRIENDS_PREFIX   = DATA_DIR + "/friends_"
	MESSAGE_DIR      = DATA_DIR + "/"
	MAX_AVATAR_SIZE  = 2 * 1024 * 1024  // 2MB
	MAX_IMAGE_SIZE   = 10 * 1024 * 1024 // 10MB
	MAX_CHUNK_SIZE   = 50 * 1024 * 1024 // 50MB
	MIN_CHUNK_SIZE   = 1 * 1024 * 1024  // 1MB
	MAX_FILE_SIZE    = 500 * 1024 * 1024// 500MB
)

// 允许的压缩包格式（复刻原PHP）
var ALLOWED_EXTENSIONS = []string{"zip", "rar", "7z", "tar", "gz", "bz2", "xz", "tgz"}

// 全局会话存储（复刻原PHP session）
var sessionStore = sessions.NewFilesystemStore("./sessions", []byte("ntc-session-secret-2024"))

func main() {
	// 初始化目录（完全复刻原PHP mkdir逻辑）
	if err := initDirs(); err != nil {
		panic("初始化目录失败: " + err.Error())
	}

	// 注册路由（与原PHP action完全对应，无遗漏）
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/api", handleAPI)
	http.HandleFunc("/upF/", handleUploadFile) // 压缩包上传下载路由

	// 静态文件服务（HTML/JS/CSS完全复用原文件）
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// 启动服务（原PHP默认80端口）
	println("服务启动: http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic("服务启动失败: " + err.Error())
	}
}

// 初始化目录（完全复刻原PHP逻辑，权限0755）
func initDirs() error {
	dirs := []string{
		DATA_DIR, AVATAR_DIR, UPLOAD_DIR,
		filepath.Join(UPLOAD_DIR, "FileName"),
		filepath.Join(UPLOAD_DIR, "temp"),
		"./sessions",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		if !isWritable(dir) {
			return os.ErrPermission
		}
	}

	// 初始化JSON文件（复刻原PHP file_put_contents逻辑）
	for _, file := range []string{USERS_FILE, FILE_NAME_JSON} {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			if err := os.WriteFile(file, []byte("[]"), 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

// 检查目录是否可写（复刻原PHP is_writable）
func isWritable(dir string) bool {
	testFile := filepath.Join(dir, "test_write.tmp")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return false
	}
	return os.Remove(testFile) == nil
}

// 处理HTML入口（完全复刻原PHP默认返回index.html）
func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, "./static/index.html")
}

// 处理API请求（完全复刻原PHP action分发逻辑，无任何遗漏）
func handleAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	action := r.FormValue("action")
	if action == "" {
		writeJSON(w, map[string]any{"success": false, "error": "无效操作"})
		return
	}

	// 所有原PHP action完全复刻，无遗漏
	switch action {
	case "register":
		handleRegister(w, r)
	case "login":
		handleLogin(w, r)
	case "logout":
		handleLogout(w, r)
	case "getUserInfo":
		handleGetUserInfo(w, r)
	case "updateUser":
		handleUpdateUser(w, r)
	case "uploadAvatar":
		handleUploadAvatar(w, r)
	case "searchUser":
		handleSearchUser(w, r)
	case "searchUserInfo":
		handleSearchUserInfo(w, r)
	case "sendFriendRequest":
		handleSendFriendRequest(w, r)
	case "acceptFriendRequest":
		handleAcceptFriendRequest(w, r)
	case "rejectFriendRequest":
		handleRejectFriendRequest(w, r)
	case "getFriendRequests":
		handleGetFriendRequests(w, r)
	case "getFriends":
		handleGetFriends(w, r)
	case "getMessages":
		handleGetMessages(w, r)
	case "sendMessage":
		handleSendMessage(w, r)
	case "uploadImage":
		handleUploadImage(w, r)
	case "getImage":
		handleGetImage(w, r)
	case "deleteFriend":
		handleDeleteFriend(w, r)
	default:
		writeJSON(w, map[string]any{"success": false, "error": "无效操作"})
	}
}

// 统一JSON响应（复刻原PHP json_encode格式）
func writeJSON(w http.ResponseWriter, data map[string]any) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"success":false,"error":"JSON编码失败"}`))
		return
	}
	w.Write(jsonData)
}
