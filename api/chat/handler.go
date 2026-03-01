package chat

import (
	"path/filepath"
	"strings"

	"github.com/fiber-go/fiber/v2"
	"go-fiber-web/model"
	"go-fiber-web/service"
	"go-fiber-web/utils"
)

// 发送好友请求
func SendFriendRequest(c *fiber.Ctx) error {
	type req struct {
		TargetID string `form:"target_id"`
	}
	var r req
	if err := c.BodyParser(&r); err != nil {
		return utils.Fail(c, "参数错误")
	}

	userID := c.Locals("user_id").(string)
	if err := service.SendFriendRequest(userID, r.TargetID); err != nil {
		return utils.Fail(c, err.Error())
	}

	return utils.Success(c, nil)
}

// 接受好友请求
func AcceptFriendRequest(c *fiber.Ctx) error {
	type req struct {
		RequesterID string `form:"requester_id"`
	}
	var r req
	if err := c.BodyParser(&r); err != nil {
		return utils.Fail(c, "参数错误")
	}

	userID := c.Locals("user_id").(string)
	if err := service.AcceptFriendRequest(userID, r.RequesterID); err != nil {
		return utils.Fail(c, err.Error())
	}

	return utils.Success(c, nil)
}

// 拒绝好友请求
func RejectFriendRequest(c *fiber.Ctx) error {
	type req struct {
		RequesterID string `form:"requester_id"`
	}
	var r req
	if err := c.BodyParser(&r); err != nil {
		return utils.Fail(c, "参数错误")
	}

	userID := c.Locals("user_id").(string)
	if err := service.RejectFriendRequest(userID, r.RequesterID); err != nil {
		return utils.Fail(c, err.Error())
	}

	return utils.Success(c, nil)
}

// 获取好友请求列表
func GetFriendRequests(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	requests, err := service.GetFriendRequests(userID)
	if err != nil {
		return utils.Fail(c, err.Error())
	}

	return utils.Success(c, requests)
}

// 获取好友列表
func GetFriends(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	friends, err := service.GetFriends(userID)
	if err != nil {
		return utils.Fail(c, err.Error())
	}

	return utils.Success(c, friends)
}

// 发送消息
func SendMessage(c *fiber.Ctx) error {
	type req struct {
		FriendID string         `form:"friend_id"`
		Content  string         `form:"content"`
		Type     model.MessageType `form:"type" default:"text"`
	}
	var r req
	if err := c.BodyParser(&r); err != nil {
		return utils.Fail(c, "参数错误")
	}

	userID := c.Locals("user_id").(string)
	if err := service.SendMessage(userID, r.FriendID, r.Content, r.Type); err != nil {
		return utils.Fail(c, err.Error())
	}

	return utils.Success(c, nil)
}

// 获取聊天记录
func GetMessages(c *fiber.Ctx) error {
	friendID := c.Query("friend_id")
	if friendID == "" {
		return utils.Fail(c, "缺少好友ID")
	}

	userID := c.Locals("user_id").(string)
	messages, err := service.GetMessages(userID, friendID)
	if err != nil {
		return utils.Fail(c, err.Error())
	}

	return utils.Success(c, messages)
}

// 删除好友
func DeleteFriend(c *fiber.Ctx) error {
	type req struct {
		FriendID string `form:"friend_id"`
	}
	var r req
	if err := c.BodyParser(&r); err != nil {
		return utils.Fail(c, "参数错误")
	}

	userID := c.Locals("user_id").(string)
	if err := service.DeleteFriend(userID, r.FriendID); err != nil {
		return utils.Fail(c, err.Error())
	}

	return utils.Success(c, nil)
}

// 上传聊天图片
func UploadChatImage(c *fiber.Ctx) error {
	file, err := c.FormFile("image")
	if err != nil {
		return utils.Fail(c, "未获取到图片文件")
	}

	// 校验文件类型
	ext := filepath.Ext(file.Filename)
	allowedExt := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	valid := false
	for _, e := range allowedExt {
		if strings.ToLower(ext) == e {
			valid = true
			break
		}
	}
	if !valid {
		return utils.Fail(c, "只允许上传图片格式")
	}

	// 打开文件
	src, err := file.Open()
	if err != nil {
		return utils.Fail(c, "文件打开失败")
	}
	defer src.Close()

	// 生成MD5
	md5Str := utils.FileMD5(src)
	// 保存文件
	path, err := utils.SaveChatImage(src, md5Str, strings.TrimPrefix(ext, "."))
	if err != nil {
		return utils.Fail(c, "图片保存失败")
	}

	return utils.Success(c, fiber.Map{"image_path": path})
}
