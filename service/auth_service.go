package service

import (
	"errors"
	"math/rand"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"go-fiber-web/model"
	"go-fiber-web/utils"
)

// 生成10位唯一用户ID
func generateUserID() string {
	rand.Seed(time.Now().UnixNano())
	for {
		id := fmt.Sprintf("%010d", rand.Int63n(9000000000)+1000000000)
		var user model.User
		if err := model.DB.Where("id = ?", id).First(&user).Error; err == gorm.ErrRecordNotFound {
			return id
		}
	}
}

// 注册业务逻辑
func Register(username, password string) (*model.User, error) {
	// 检查用户名是否已存在
	var exist model.User
	if err := model.DB.Where("username = ?", username).First(&exist).Error; err != gorm.ErrRecordNotFound {
		return nil, errors.New("用户名已存在")
	}

	// 密码哈希
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("密码加密失败")
	}

	// 创建用户
	userID := generateUserID()
	user := &model.User{
		ID:         userID,
		Username:   username,
		Password:   string(hash),
		Nickname:   username,
		Bio:        "",
		VerifyMode: "need_verify",
		Registered: time.Now().Unix(),
	}

	if err := model.DB.Create(user).Error; err != nil {
		return nil, errors.New("用户创建失败")
	}

	return user, nil
}

// 登录业务逻辑
func Login(username, password string) (string, *model.User, error) {
	// 支持用户名或ID登录
	var user model.User
	if err := model.DB.Where("username = ?", username).First(&user).Error; err == gorm.ErrRecordNotFound {
		if err := model.DB.Where("id = ?", username).First(&user).Error; err == gorm.ErrRecordNotFound {
			return "", nil, errors.New("用户名/ID或密码错误")
		}
	}

	// 校验密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", nil, errors.New("用户名/ID或密码错误")
	}

	// 生成JWT Token
	token, err := utils.GenerateToken(user.ID)
	if err != nil {
		return "", nil, errors.New("登录失败，请重试")
	}

	return token, user, nil
}

// 通过ID获取用户信息
func GetUserByID(userID string) (*model.User, error) {
	var user model.User
	if err := model.DB.Where("id = ?", userID).First(&user).Error; err == gorm.ErrRecordNotFound {
		return nil, errors.New("用户不存在")
	}
	return &user, nil
}

// 更新用户信息（昵称/简介/验证方式）
func UpdateUserInfo(userID string, nickname, bio, verifyMode string) (*model.User, error) {
	var user model.User
	if err := model.DB.Where("id = ?", userID).First(&user).Error; err == gorm.ErrRecordNotFound {
		return nil, errors.New("用户不存在")
	}

	if nickname != "" {
		user.Nickname = nickname
	}
	if bio != "" {
		user.Bio = bio
	}
	if verifyMode != "" {
		user.VerifyMode = verifyMode
	}

	if err := model.DB.Save(&user).Error; err != nil {
		return nil, errors.New("信息更新失败")
	}
	return &user, nil
}

// 修改密码
func UpdatePassword(userID, oldPwd, newPwd string) error {
	var user model.User
	if err := model.DB.Where("id = ?", userID).First(&user).Error; err == gorm.ErrRecordNotFound {
		return errors.New("用户不存在")
	}

	// 校验旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPwd)); err != nil {
		return errors.New("旧密码错误")
	}

	// 新密码哈希
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPwd), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("密码加密失败")
	}

	user.Password = string(newHash)
	if err := model.DB.Save(&user).Error; err != nil {
		return errors.New("密码修改失败")
	}
	return nil
}

// 更新头像
func UpdateAvatar(userID, avatarPath string) error {
	var user model.User
	if err := model.DB.Where("id = ?", userID).First(&user).Error; err == gorm.ErrRecordNotFound {
		return errors.New("用户不存在")
	}

	user.Avatar = avatarPath
	if err := model.DB.Save(&user).Error; err != nil {
		return errors.New("头像更新失败")
	}
	return nil
}
