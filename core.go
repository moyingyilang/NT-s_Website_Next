package main

import (
    "crypto/md5"       // 用于 md5.New
    "crypto/rand"
    "encoding/base64"
    "encoding/json"
    "errors"
    "fmt"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "time"
)

// 数据结构完全复刻原PHP（字段名、类型一致）
type User struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	EncryptedHash string `json:"encrypted_hash"`
	Nickname      string `json:"nickname"`
	Avatar        string `json:"avatar"`
	Bio           string `json:"bio"`
	VerifyMode    string `json:"verify_mode"`
	Registered    int64  `json:"registered"`
}

type Friend struct {
	ID         string `json:"id"`
	Status     string `json:"status"` // pending/accepted
	Since      int64  `json:"since"`
	SharedKey  string `json:"shared_key"`
}

type Message struct {
	From      string `json:"from"`
	Content   string `json:"content"`
	Type      string `json:"type"` // text/image
	Timestamp int64  `json:"timestamp"`
}

type FileMapItem struct {
	Original string `json:"original"`
	MD5      string `json:"md5"`
	Mime     string `json:"mime"`
}

// ---------------------- 用户相关（完全复刻原PHP逻辑）----------------------
func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, map[string]any{"success": false, "error": "仅支持POST请求"})
		return
	}

	// 解析表单（复刻原PHP $_POST）
	if err := r.ParseMultipartForm(MAX_AVATAR_SIZE); err != nil {
		writeJSON(w, map[string]any{"success": false, "error": "表单解析失败"})
		return
	}

	username := strings.TrimSpace(r.PostForm.Get("username"))
	password := strings.TrimSpace(r.PostForm.Get("password"))
	if username == "" || password == "" {
		writeJSON(w, map[string]any{"success": false, "error": "用户名和密码不能为空"})
		return
	}

	// 检查用户名是否存在（复刻原PHP getUserByUsername）
	users := getUsers()
	for _, u := range users {
		if u.Username == username {
			writeJSON(w, map[string]any{"success": false, "error": "用户名已存在"})
			return
		}
	}

	// 生成10位数字用户ID（完全复刻原PHP generateUserId）
	userID := generateUserId()
	hashedPwd := hashPassword(password)
	encryptedHash := encryptHash(hashedPwd)

	// 初始化用户信息（复刻原PHP newUser）
	newUser := User{
		ID:            userID,
		Username:      username,
		EncryptedHash: encryptedHash,
		Nickname:      username,
		VerifyMode:    "need_verify",
		Registered:    time.Now().Unix(),
		Bio:           "",
		Avatar:        "",
	}

	// 处理头像上传（复刻原PHP handleAvatarUpload）
	if file, _, err := r.FormFile("avatar"); err == nil {
		defer file.Close()
		avatarPath, err := saveAvatar(file, userID)
		if err == nil {
			newUser.Avatar = avatarPath
		}
	}

	// 保存用户（复刻原PHP saveUsers）
	users = append(users, newUser)
	if err := saveUsers(users); err != nil {
		writeJSON(w, map[string]any{"success": false, "error": "保存用户失败"})
		return
	}

	writeJSON(w, map[string]any{
		"success": true,
		"user":    safeUser(newUser),
	})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, map[string]any{"success": false, "error": "仅支持POST请求"})
		return
	}

	// 解析表单（复刻原PHP $_POST）
	usernameOrID := strings.TrimSpace(r.PostForm.Get("username"))
	password := strings.TrimSpace(r.PostForm.Get("password"))
	if usernameOrID == "" || password == "" {
		writeJSON(w, map[string]any{"success": false, "error": "请输入用户名/ID和密码"})
		return
	}

	// 查找用户（复刻原PHP getUserByUsername/getUserById）
	users := getUsers()
	var user *User
	for i := range users {
		if users[i].Username == usernameOrID || users[i].ID == usernameOrID {
			user = &users[i]
			break
		}
	}

	if user == nil {
		writeJSON(w, map[string]any{"success": false, "error": "用户名/ID或密码错误"})
		return
	}

	// 验证密码（复刻原PHP decryptHash + password_verify）
	decryptedHash, err := decryptHash(user.EncryptedHash)
	if err != nil || !verifyPassword(password, decryptedHash) {
		writeJSON(w, map[string]any{"success": false, "error": "用户名/ID或密码错误"})
		return
	}

	// 生成会话和CSRF令牌（复刻原PHP session_regenerate_id + csrf_token）
	session, err := sessionStore.Get(r, "ntc-session")
	if err != nil {
		writeJSON(w, map[string]any{"success": false, "error": "会话创建失败"})
		return
	}

	session.Values["user_id"] = user.ID
	session.Values["csrf_token"] = generateCSRFToken()
	session.Options.MaxAge = 86400 * 7 // 7天有效期
	if err := session.Save(r, w); err != nil {
		writeJSON(w, map[string]any{"success": false, "error": "会话保存失败"})
		return
	}

	writeJSON(w, map[string]any{
		"success":    true,
		"user":       safeUser(*user),
		"csrf_token": session.Values["csrf_token"],
	})
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	// 验证CSRF（复刻原PHP checkCSRF）
	session, err := sessionStore.Get(r, "ntc-session")
	if err != nil || !checkCSRF(r, session) {
		writeJSON(w, map[string]any{"success": false, "error": "CSRF令牌无效"})
		return
	}

	// 销毁会话（复刻原PHP session_destroy）
	session.Options.MaxAge = -1
	if err := session.Save(r, w); err != nil {
		writeJSON(w, map[string]any{"success": false, "error": "登出失败"})
		return
	}

	writeJSON(w, map[string]any{"success": true})
}

func handleGetUserInfo(w http.ResponseWriter, r *http.Request) {
	// 验证登录状态（复刻原PHP $_SESSION['user_id']）
	session, err := sessionStore.Get(r, "ntc-session")
	if err != nil || session.Values["user_id"] == nil {
		writeJSON(w, map[string]any{"loggedIn": false})
		return
	}

	userID := session.Values["user_id"].(string)
	user := getUserById(userID)
	if user == nil {
		// 用户不存在，销毁会话（复刻原PHP session_destroy）
		session.Options.MaxAge = -1
		session.Save(r, w)
		writeJSON(w, map[string]any{"loggedIn": false})
		return
	}

	// 生成CSRF令牌（复刻原PHP逻辑）
	if session.Values["csrf_token"] == nil {
		session.Values["csrf_token"] = generateCSRFToken()
		session.Save(r, w)
	}

	writeJSON(w, map[string]any{
		"loggedIn":   true,
		"user":       safeUser(*user),
		"csrf_token": session.Values["csrf_token"],
	})
}

func handleUpdateUser(w http.ResponseWriter, r *http.Request) {
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

	userID := session.Values["user_id"].(string)
	users := getUsers()
	index := -1
	for i := range users {
		if users[i].ID == userID {
			index = i
			break
		}
	}
	if index == -1 {
		writeJSON(w, map[string]any{"success": false, "error": "用户不存在"})
		return
	}

	user := &users[index]
	oldPassword := strings.TrimSpace(r.PostForm.Get("old_password"))
	newPassword := strings.TrimSpace(r.PostForm.Get("password"))

	// 修改密码（复刻原PHP逻辑：验证旧密码）
	if oldPassword != "" && newPassword != "" {
		decryptedHash, err := decryptHash(user.EncryptedHash)
		if err != nil || !verifyPassword(oldPassword, decryptedHash) {
			writeJSON(w, map[string]any{"success": false, "error": "旧密码错误"})
			return
		}
		// 重新加密密码（复刻原PHP逻辑）
		hashedPwd := hashPassword(newPassword)
		user.EncryptedHash = encryptHash(hashedPwd)
	}

	// 修改昵称（复刻原PHP逻辑）
	if nickname := strings.TrimSpace(r.PostForm.Get("nickname")); nickname != "" {
		user.Nickname = nickname
	}

	// 修改验证方式（复刻原PHP逻辑）
	if verifyMode := r.PostForm.Get("verify_mode"); verifyMode != "" {
		user.VerifyMode = verifyMode
	}

	// 修改简介（复刻原PHP逻辑）
	if bio := strings.TrimSpace(r.PostForm.Get("bio")); bio != "" {
		user.Bio = bio
	}

	// 保存修改（复刻原PHP saveUsers）
	if err := saveUsers(users); err != nil {
		writeJSON(w, map[string]any{"success": false, "error": "保存失败"})
		return
	}

	writeJSON(w, map[string]any{
		"success":    true,
		"user":       safeUser(*user),
		"csrf_token": session.Values["csrf_token"],
	})
}

// ---------------------- 好友相关（完全复刻原PHP逻辑）----------------------
func handleSearchUser(w http.ResponseWriter, r *http.Request) {
	// 验证登录（复刻原PHP逻辑）
	session, err := sessionStore.Get(r, "ntc-session")
	if err != nil || session.Values["user_id"] == nil {
		writeJSON(w, map[string]any{"success": false, "error": "未登录"})
		return
	}

	currentID := session.Values["user_id"].(string)
	userId := strings.TrimSpace(r.FormValue("userId"))
	if !isValidId(userId) {
		writeJSON(w, map[string]any{"success": false, "error": "用户ID格式错误"})
		return
	}

	// 禁止添加自己（复刻原PHP逻辑）
	if userId == currentID {
		writeJSON(w, map[string]any{"success": false, "error": "不能添加自己"})
		return
	}

	// 查找用户（复刻原PHP getUserById）
	user := getUserById(userId)
	if user == nil {
		writeJSON(w, map[string]any{"success": false, "error": "用户不存在"})
		return
	}

	writeJSON(w, map[string]any{
		"success": true,
		"user": map[string]any{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

func handleSearchUserInfo(w http.ResponseWriter, r *http.Request) {
	// 验证登录（复刻原PHP逻辑）
	session, err := sessionStore.Get(r, "ntc-session")
	if err != nil || session.Values["user_id"] == nil {
		writeJSON(w, map[string]any{"success": false, "error": "未登录"})
		return
	}

	userId := strings.TrimSpace(r.FormValue("userId"))
	if !isValidId(userId) {
		writeJSON(w, map[string]any{"success": false, "error": "用户ID格式错误"})
		return
	}

	// 查找用户（复刻原PHP getUserById）
	user := getUserById(userId)
	if user == nil {
		writeJSON(w, map[string]any{"success": false, "error": "用户不存在"})
		return
	}

	// 返回公开信息（复刻原PHP逻辑）
	writeJSON(w, map[string]any{
		"success": true,
		"user": map[string]any{
			"id":         user.ID,
			"username":   user.Username,
			"nickname":   user.Nickname,
			"registered": user.Registered,
			"bio":        user.Bio,
		},
	})
}

func handleSendFriendRequest(w http.ResponseWriter, r *http.Request) {
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

	currentID := session.Values["user_id"].(string)
	targetID := strings.TrimSpace(r.PostForm.Get("targetId"))
	if !isValidId(targetID) {
		writeJSON(w, map[string]any{"success": false, "error": "目标ID格式错误"})
		return
	}
	if targetID == currentID {
		writeJSON(w, map[string]any{"success": false, "error": "不能添加自己"})
		return
	}

	// 检查目标用户是否存在（复刻原PHP逻辑）
	targetUser := getUserById(targetID)
	if targetUser == nil {
		writeJSON(w, map[string]any{"success": false, "error": "目标用户不存在"})
		return
	}

	// 检查是否已是好友（复刻原PHP逻辑）
	myFriends := getFriends(currentID)
	for _, f := range myFriends {
		if f.ID == targetID && f.Status == "accepted" {
			writeJSON(w, map[string]any{"success": false, "error": "已经是好友"})
			return
		}
	}

	// 处理验证模式（复刻原PHP逻辑）
	switch targetUser.VerifyMode {
	case "deny_all":
		writeJSON(w, map[string]any{"success": false, "error": "对方禁止添加好友"})
		return
	case "allow_all":
		// 直接添加为好友（复刻原PHP逻辑）
		sharedKey := generateSharedKey()

		// 更新对方好友列表
		targetFriends := getFriends(targetID)
		targetFriends = filterFriends(targetFriends, currentID)
		targetFriends = append(targetFriends, Friend{
			ID:         currentID,
			Status:     "accepted",
			Since:      time.Now().Unix(),
			SharedKey:  sharedKey,
		})
		if err := saveFriends(targetID, targetFriends); err != nil {
			writeJSON(w, map[string]any{"success": false, "error": "添加失败"})
			return
		}

		// 更新自己好友列表
		myFriends = filterFriends(myFriends, targetID)
		myFriends = append(myFriends, Friend{
			ID:         targetID,
			Status:     "accepted",
			Since:      time.Now().Unix(),
			SharedKey:  sharedKey,
		})
		if err := saveFriends(currentID, myFriends); err != nil {
			writeJSON(w, map[string]any{"success": false, "error": "添加失败"})
			return
		}

		writeJSON(w, map[string]any{"success": true, "message": "添加好友成功"})
		return
	case "need_verify":
		// 发送请求（复刻原PHP逻辑）
		targetFriends := getFriends(targetID)
		// 检查是否已发送请求
		for _, f := range targetFriends {
			if f.ID == currentID && f.Status == "pending" {
				writeJSON(w, map[string]any{"success": false, "error": "请求已发送，请等待"})
				return
			}
		}
		// 添加请求
		targetFriends = append(targetFriends, Friend{
			ID:         currentID,
			Status:     "pending",
			Since:      time.Now().Unix(),
			SharedKey:  "",
		})
		if err := saveFriends(targetID, targetFriends); err != nil {
			writeJSON(w, map[string]any{"success": false, "error": "发送请求失败"})
			return
		}

		writeJSON(w, map[string]any{"success": true, "message": "好友请求已发送"})
		return
	}
}

func handleAcceptFriendRequest(w http.ResponseWriter, r *http.Request) {
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

	currentID := session.Values["user_id"].(string)
	requesterID := strings.TrimSpace(r.PostForm.Get("requesterId"))
	if !isValidId(requesterID) {
		writeJSON(w, map[string]any{"success": false, "error": "请求者ID格式错误"})
		return
	}

	// 检查请求是否存在（复刻原PHP逻辑）
	myFriends := getFriends(currentID)
	foundIndex := -1
	for i, f := range myFriends {
		if f.ID == requesterID && f.Status == "pending" {
			foundIndex = i
			break
		}
	}
	if foundIndex == -1 {
		writeJSON(w, map[string]any{"success": false, "error": "没有找到该请求"})
		return
	}

	// 生成共享密钥（复刻原PHP逻辑）
	sharedKey := generateSharedKey()

	// 更新自己的好友列表（复刻原PHP逻辑）
	myFriends[foundIndex].Status = "accepted"
	myFriends[foundIndex].Since = time.Now().Unix()
	myFriends[foundIndex].SharedKey = sharedKey
	if err := saveFriends(currentID, myFriends); err != nil {
		writeJSON(w, map[string]any{"success": false, "error": "接受失败"})
		return
	}

	// 更新对方的好友列表（复刻原PHP逻辑）
	requesterFriends := getFriends(requesterID)
	requesterFriends = filterFriends(requesterFriends, currentID)
	requesterFriends = append(requesterFriends, Friend{
		ID:         currentID,
		Status:     "accepted",
		Since:      time.Now().Unix(),
		SharedKey:  sharedKey,
	})
	if err := saveFriends(requesterID, requesterFriends); err != nil {
		writeJSON(w, map[string]any{"success": false, "error": "接受失败"})
		return
	}

	writeJSON(w, map[string]any{"success": true})
}

func handleRejectFriendRequest(w http.ResponseWriter, r *http.Request) {
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

	currentID := session.Values["user_id"].(string)
	requesterID := strings.TrimSpace(r.PostForm.Get("requesterId"))
	if !isValidId(requesterID) {
		writeJSON(w, map[string]any{"success": false, "error": "请求者ID格式错误"})
		return
	}

	// 移除请求（复刻原PHP逻辑）
	myFriends := getFriends(currentID)
	myFriends = filterFriendsByStatus(myFriends, requesterID, "pending")
	if err := saveFriends(currentID, myFriends); err != nil {
		writeJSON(w, map[string]any{"success": false, "error": "拒绝失败"})
		return
	}

	writeJSON(w, map[string]any{"success": true})
}

func handleGetFriendRequests(w http.ResponseWriter, r *http.Request) {
	// 验证登录（复刻原PHP逻辑）
	session, err := sessionStore.Get(r, "ntc-session")
	if err != nil || session.Values["user_id"] == nil {
		writeJSON(w, map[string]any{"success": false, "error": "未登录"})
		return
	}

	currentID := session.Values["user_id"].(string)
	myFriends := getFriends(currentID)
	pendingRequests := filterFriendsByStatus(myFriends, "", "pending")

	// 组装请求列表（复刻原PHP逻辑）
	result := []map[string]any{}
	for _, req := range pendingRequests {
		user := getUserById(req.ID)
		if user != nil {
			result = append(result, map[string]any{
				"id":       user.ID,
				"username": user.Username,
				"nickname": user.Nickname,
				"avatar":   user.Avatar,
			})
		}
	}

	writeJSON(w, map[string]any{
		"success": true,
		"requests": result,
	})
}

func handleGetFriends(w http.ResponseWriter, r *http.Request) {
	// 验证登录（复刻原PHP逻辑）
	session, err := sessionStore.Get(r, "ntc-session")
	if err != nil || session.Values["user_id"] == nil {
		writeJSON(w, map[string]any{"success": false, "error": "未登录"})
		return
	}

	currentID := session.Values["user_id"].(string)
	myFriends := getFriends(currentID)
	acceptedFriends := filterFriendsByStatus(myFriends, "", "accepted")

	// 组装好友列表（复刻原PHP逻辑）
	result := []map[string]any{}
	for _, f := range acceptedFriends {
		user := getUserById(f.ID)
		if user != nil {
			result = append(result, map[string]any{
				"id":       user.ID,
				"username": user.Username,
				"nickname": user.Nickname,
				"avatar":   user.Avatar,
			})
		}
	}

	writeJSON(w, map[string]any{
		"success": true,
		"friends": result,
	})
}

func handleDeleteFriend(w http.ResponseWriter, r *http.Request) {
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

	currentID := session.Values["user_id"].(string)
	friendID := strings.TrimSpace(r.PostForm.Get("friendId"))
	if !isValidId(friendID) {
		writeJSON(w, map[string]any{"success": false, "error": "好友ID格式错误"})
		return
	}

	// 从自己好友列表移除（复刻原PHP逻辑）
	myFriends := getFriends(currentID)
	originalCount := len(myFriends)
	myFriends = filterFriendsByStatus(myFriends, friendID, "accepted")
	if len(myFriends) == originalCount {
		writeJSON(w, map[string]any{"success": false, "error": "好友关系不存在"})
		return
	}
	if err := saveFriends(currentID, myFriends); err != nil {
		writeJSON(w, map[string]any{"success": false, "error": "删除失败"})
		return
	}

	// 从对方好友列表移除（复刻原PHP逻辑）
	theirFriends := getFriends(friendID)
	theirFriends = filterFriendsByStatus(theirFriends, currentID, "accepted")
	if err := saveFriends(friendID, theirFriends); err != nil {
		writeJSON(w, map[string]any{"success": false, "error": "删除失败"})
		return
	}

	// 删除聊天记录（复刻原PHP逻辑）
	myMsgFile := filepath.Join(MESSAGE_DIR, currentID, friendID+".json")
	if err := os.RemoveAll(myMsgFile); err != nil && !os.IsNotExist(err) {
		writeJSON(w, map[string]any{"success": false, "error": "删除聊天记录失败"})
		return
	}
	theirMsgFile := filepath.Join(MESSAGE_DIR, friendID, currentID+".json")
	if err := os.RemoveAll(theirMsgFile); err != nil && !os.IsNotExist(err) {
		writeJSON(w, map[string]any{"success": false, "error": "删除聊天记录失败"})
		return
	}

	writeJSON(w, map[string]any{"success": true})
}

// ---------------------- 消息相关（完全复刻原PHP逻辑）----------------------
func handleGetMessages(w http.ResponseWriter, r *http.Request) {
	// 验证登录（复刻原PHP逻辑）
	session, err := sessionStore.Get(r, "ntc-session")
	if err != nil || session.Values["user_id"] == nil {
		writeJSON(w, map[string]any{"success": false, "error": "未登录"})
		return
	}

	currentID := session.Values["user_id"].(string)
	friendID := strings.TrimSpace(r.FormValue("friendId"))
	if !isValidId(friendID) {
		writeJSON(w, map[string]any{"success": false, "error": "好友ID格式错误"})
		return
	}

	// 获取消息（复刻原PHP getMessages逻辑，支持兼容旧版本）
	messages, err := getMessages(currentID, friendID)
	if err != nil {
		writeJSON(w, map[string]any{"success": false, "error": "获取消息失败"})
		return
	}

	// 补充发送者名称（复刻原PHP逻辑）
	usersCache := make(map[string]string)
	for i := range messages {
		if _, ok := usersCache[messages[i].From]; !ok {
			user := getUserById(messages[i].From)
			if user != nil {
				usersCache[messages[i].From] = user.Nickname
			} else {
				usersCache[messages[i].From] = messages[i].From
			}
		}
		messages[i].From = usersCache[messages[i].From]
	}

	writeJSON(w, map[string]any{
		"success": true,
		"messages": messages,
	})
}

func handleSendMessage(w http.ResponseWriter, r *http.Request) {
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

	fromID := session.Values["user_id"].(string)
	toID := strings.TrimSpace(r.PostForm.Get("friendId"))
	content := strings.TrimSpace(r.PostForm.Get("content"))
	msgType := r.PostForm.Get("type")
	if !isValidId(toID) || content == "" || msgType == "" {
		writeJSON(w, map[string]any{"success": false, "error": "参数不足"})
		return
	}

	// 验证好友关系（复刻原PHP逻辑）
	myFriends := getFriends(fromID)
	isFriend := false
	for _, f := range myFriends {
		if f.ID == toID && f.Status == "accepted" {
			isFriend = true
			break
		}
	}
	if !isFriend {
		writeJSON(w, map[string]any{"success": false, "error": "不是好友关系"})
		return
	}

	// 构建消息（复刻原PHP逻辑）
	message := Message{
		From:      fromID,
		Content:   content,
		Type:      msgType,
		Timestamp: time.Now().Unix(),
	}

	// 保存消息到双方目录（复刻原PHP saveMessageToUser）
	if err := saveMessageToUser(fromID, toID, message); err != nil {
		writeJSON(w, map[string]any{"success": false, "error": "发送失败"})
		return
	}
	if err := saveMessageToUser(toID, fromID, message); err != nil {
		writeJSON(w, map[string]any{"success": false, "error": "发送失败"})
		return
	}

	writeJSON(w, map[string]any{"success": true})
}

// ---------------------- 辅助函数（完全复刻原PHP逻辑）----------------------
// 生成10位数字用户ID
func generateUserId() string {
	for {
		id := fmt.Sprintf("%d", randInt(1000000000, 9999999999))
		if getUserById(id) == nil {
			return id
		}
	}
}

// 生成随机整数
func randInt(min, max int64) int64 {
	b := make([]byte, 8)
	rand.Read(b)
	return int64(b[0])%((max - min) + 1) + min
}

// 生成CSRF令牌
func generateCSRFToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// 生成共享密钥（32字节Base64）
func generateSharedKey() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

// 获取用户列表
func getUsers() []User {
	data, err := os.ReadFile(USERS_FILE)
	if err != nil {
		return []User{}
	}
	var users []User
	json.Unmarshal(data, &users)
	return users
}

// 保存用户列表
func saveUsers(users []User) error {
	data, err := json.Marshal(users)
	if err != nil {
		return err
	}
	return os.WriteFile(USERS_FILE, data, 0644)
}

// 根据ID获取用户
func getUserById(id string) *User {
	users := getUsers()
	for i := range users {
		if users[i].ID == id {
			return &users[i]
		}
	}
	return nil
}

// 过滤敏感字段的用户信息
func safeUser(user User) map[string]any {
	return map[string]any{
		"id":         user.ID,
		"username":   user.Username,
		"nickname":   user.Nickname,
		"avatar":     user.Avatar,
		"bio":        user.Bio,
		"verify_mode": user.VerifyMode,
		"registered": user.Registered,
	}
}

// 获取好友列表
func getFriends(userId string) []Friend {
	filePath := FRIENDS_PREFIX + userId + ".json"
	data, err := os.ReadFile(filePath)
	if err != nil {
		return []Friend{}
	}
	var friends []Friend
	json.Unmarshal(data, &friends)
	return friends
}

// 保存好友列表
func saveFriends(userId string, friends []Friend) error {
	filePath := FRIENDS_PREFIX + userId + ".json"
	data, err := json.Marshal(friends)
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

// 过滤好友列表（移除指定用户）
func filterFriends(friends []Friend, userId string) []Friend {
	result := []Friend{}
	for _, f := range friends {
		if f.ID != userId {
			result = append(result, f)
		}
	}
	return result
}

// 过滤好友列表（按状态和用户ID）
func filterFriendsByStatus(friends []Friend, userId, status string) []Friend {
	result := []Friend{}
	for _, f := range friends {
		if (userId == "" || f.ID != userId) && (status == "" || f.Status != status) {
			result = append(result, f)
		}
	}
	return result
}

// 获取消息（复刻原PHP getMessages，支持兼容旧版本）
func getMessages(userId, friendId string) ([]Message, error) {
	user := getUserById(userId)
	if user == nil {
		return nil, errors.New("用户不存在")
	}

	// 创建消息目录（复刻原PHP逻辑）
	msgDir := filepath.Join(MESSAGE_DIR, userId)
	if err := os.MkdirAll(msgDir, 0755); err != nil {
		return nil, err
	}

	msgFile := filepath.Join(msgDir, friendId+".json")
	if _, err := os.Stat(msgFile); os.IsNotExist(err) {
		return []Message{}, nil
	}

	// 读取文件内容
	content, err := os.ReadFile(msgFile)
	if err != nil || len(content) == 0 {
		return []Message{}, nil
	}

	// 1. 尝试新密钥解密（复刻原PHP逻辑）
	tryDecrypt := func() ([]Message, error) {
		key, err := getSessionKey(userId, friendId)
		if err != nil {
			return nil, err
		}
		return decryptUserData(content, key)
	}

	// 2. 尝试旧密钥解密（兼容旧版本）
	tryLegacyDecrypt := func() ([]Message, error) {
		key, err := getLegacyKey(userId)
		if err != nil {
			return nil, err
		}
		return decryptUserData(content, key)
	}

	// 3. 尝试明文JSON（兼容最初版本）
	tryPlainJSON := func() ([]Message, error) {
		var messages []Message
		if err := json.Unmarshal(content, &messages); err != nil {
			return nil, err
		}
		return messages, nil
	}

	// 按优先级尝试解密
	messages, err := tryDecrypt()
	if err == nil {
		return messages, nil
	}
	messages, err = tryLegacyDecrypt()
	if err == nil {
		return messages, nil
	}
	messages, err = tryPlainJSON()
	if err == nil {
		return messages, nil
	}

	return nil, errors.New("解密失败")
}

// 保存消息到用户目录（复刻原PHP saveMessageToUser）
func saveMessageToUser(fromID, toID string, message Message) error {
	messages, err := getMessages(fromID, toID)
	if err != nil {
		return err
	}
	messages = append(messages, message)

	// 加密消息（复刻原PHP encryptUserData）
	key, err := getSessionKey(fromID, toID)
	if err != nil {
		return err
	}
	encrypted, err := encryptUserData(messages, key)
	if err != nil {
		return err
	}

	// 保存加密后的消息
	msgDir := filepath.Join(MESSAGE_DIR, fromID)
	if err := os.MkdirAll(msgDir, 0755); err != nil {
		return err
	}
	msgFile := filepath.Join(msgDir, toID+".json")
	return os.WriteFile(msgFile, encrypted, 0644)
}

// 获取会话密钥（复刻原PHP getSessionKey）
func getSessionKey(userIdA, userIdB string) ([]byte, error) {
	friendsA := getFriends(userIdA)
	for _, f := range friendsA {
		if f.ID == userIdB && f.Status == "accepted" && f.SharedKey != "" {
			return base64.StdEncoding.DecodeString(f.SharedKey)
		}
	}

	// 兼容旧版本（复刻原PHP逻辑）
	userA := getUserById(userIdA)
	userB := getUserById(userIdB)
	if userA == nil || userB == nil {
		return nil, errors.New("用户不存在")
	}

	var seed string
	if userIdA < userIdB {
		seed = fmt.Sprintf("%s%d%s%d", userIdA, userA.Registered, userIdB, userB.Registered)
	} else {
		seed = fmt.Sprintf("%s%d%s%d", userIdB, userB.Registered, userIdA, userA.Registered)
	}

	// SHA256哈希生成密钥
	hash := md5.New()
	hash.Write([]byte(seed))
	return hash.Sum(nil), nil
}

// 获取旧版密钥（复刻原PHP getLegacyKey）
func getLegacyKey(userId string) ([]byte, error) {
	user := getUserById(userId)
	if user == nil {
		return nil, errors.New("用户不存在")
	}
	seed := fmt.Sprintf("%d%s", user.Registered, userId)
	hash := md5.New()
	hash.Write([]byte(seed))
	return hash.Sum(nil), nil
}
