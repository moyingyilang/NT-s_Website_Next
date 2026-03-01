package service

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"go-fiber-web/model"
)

// 发送好友请求
func SendFriendRequest(fromID, toID string) error {
	// 检查目标用户是否存在
	var toUser model.User
	if err := model.DB.Where("id = ?", toID).First(&toUser).Error; err == gorm.ErrRecordNotFound {
		return errors.New("目标用户不存在")
	}

	// 检查是否已为好友
	var exist model.Friend
	err := model.DB.Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)", fromID, toID, toID, fromID).First(&exist).Error
	if err != gorm.ErrRecordNotFound {
		if exist.Status == model.StatusAccepted {
			return errors.New("已经是好友")
		}
		return errors.New("请求已发送，请等待")
	}

	// 检查目标用户验证方式
	if toUser.VerifyMode == "deny_all" {
		return errors.New("对方禁止添加好友")
	}

	// 直接通过/待审核
	status := model.StatusPending
	if toUser.VerifyMode == "allow_all" {
		status = model.StatusAccepted
	}

	// 创建好友关系（双向）
	now := time.Now()
	model.DB.Create(&model.Friend{
		UserID:   fromID,
		FriendID: toID,
		Status:   status,
		CreatedAt: now,
	})
	model.DB.Create(&model.Friend{
		UserID:   toID,
		FriendID: fromID,
		Status:   status,
		CreatedAt: now,
	})

	return nil
}

// 接受好友请求
func AcceptFriendRequest(userID, requesterID string) error {
	// 检查请求是否存在
	var friend model.Friend
	err := model.DB.Where("user_id = ? AND friend_id = ? AND status = ?", userID, requesterID, model.StatusPending).First(&friend).Error
	if err == gorm.ErrRecordNotFound {
		return errors.New("没有找到该请求")
	}

	// 更新状态为已通过
	friend.Status = model.StatusAccepted
	friend.CreatedAt = time.Now()
	model.DB.Save(&friend)

	// 更新对方的好友状态
	model.DB.Model(&model.Friend{}).
		Where("user_id = ? AND friend_id = ?", requesterID, userID).
		Updates(map[string]interface{}{
			"status":     model.StatusAccepted,
			"created_at": time.Now(),
		})

	return nil
}

// 拒绝好友请求
func RejectFriendRequest(userID, requesterID string) error {
	// 删除自己的好友记录
	model.DB.Where("user_id = ? AND friend_id = ? AND status = ?", userID, requesterID, model.StatusPending).Delete(&model.Friend{})
	// 删除对方的好友记录
	model.DB.Where("user_id = ? AND friend_id = ? AND status = ?", requesterID, userID, model.StatusPending).Delete(&model.Friend{})
	return nil
}

// 获取好友请求列表
func GetFriendRequests(userID string) ([]*model.User, error) {
	var friends []model.Friend
	model.DB.Where("user_id = ? AND status = ?", userID, model.StatusPending).Find(&friends)

	var requests []*model.User
	for _, f := range friends {
		user, err := GetUserByID(f.FriendID)
		if err == nil {
			requests = append(requests, user)
		}
	}
	return requests, nil
}

// 获取好友列表
func GetFriends(userID string) ([]*model.User, error) {
	var friends []model.Friend
	model.DB.Where("user_id = ? AND status = ?", userID, model.StatusAccepted).Find(&friends)

	var friendList []*model.User
	for _, f := range friends {
		user, err := GetUserByID(f.FriendID)
		if err == nil {
			friendList = append(friendList, user)
		}
	}
	return friendList, nil
}

// 发送消息
func SendMessage(fromID, toID, content string, msgType model.MessageType) error {
	// 检查是否为好友
	var friend model.Friend
	err := model.DB.Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)", fromID, toID, toID, fromID).First(&friend).Error
	if err == gorm.ErrRecordNotFound || friend.Status != model.StatusAccepted {
		return errors.New("不是好友关系")
	}

	// 保存消息
	message := &model.Message{
		FromID:    fromID,
		ToID:      toID,
		Content:   content,
		Type:      msgType,
		Timestamp: time.Now().Unix(),
	}

	if err := model.DB.Create(message).Error; err != nil {
		return errors.New("消息发送失败")
	}
	return nil
}

// 获取聊天记录
func GetMessages(userID, friendID string) ([]*model.Message, error) {
	var messages []*model.Message
	model.DB.Where("(from_id = ? AND to_id = ?) OR (from_id = ? AND to_id = ?)", userID, friendID, friendID, userID).
		Order("timestamp ASC").Find(&messages)
	return messages, nil
}

// 删除好友
func DeleteFriend(userID, friendID string) error {
	// 删除好友关系
	model.DB.Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)", userID, friendID, friendID, userID).Delete(&model.Friend{})
	// 删除聊天记录
	model.DB.Where("(from_id = ? AND to_id = ?) OR (from_id = ? AND to_id = ?)", userID, friendID, friendID, userID).Delete(&model.Message{})
	return nil
}
