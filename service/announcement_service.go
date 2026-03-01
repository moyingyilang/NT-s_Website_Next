package service

import (
	"errors"
	"time"

	"go-fiber-web/model"
	"gorm.io/gorm"
)

// PublishAnnouncement 发布公告（管理员接口，此处简化为所有人可发，可自行加权限）
func PublishAnnouncement(title, summary string, tags []string, visible bool) (*model.Announcement, error) {
	if title == "" || summary == "" {
		return nil, errors.New("标题和内容不能为空")
	}

	// 格式化发布日期
	date := time.Now().Format("2006-01-02")
	ann := &model.Announcement{
		Title:   title,
		Summary: summary,
		Date:    date,
		Tags:    tags,
		Visible: visible,
	}

	if err := model.DB.Create(ann).Error; err != nil {
		return nil, errors.New("发布公告失败：" + err.Error())
	}

	return ann, nil
}

// GetAnnouncements 获取公开的公告列表（按发布时间倒序）
func GetAnnouncements() ([]*model.Announcement, error) {
	var anns []*model.Announcement
	err := model.DB.Where("visible = ?", true).Order("created_at DESC").Find(&anns).Error
	if err != nil {
		return nil, errors.New("查询公告失败：" + err.Error())
	}
	return anns, nil
}

// GetAnnouncementByID 根据ID获取公告详情
func GetAnnouncementByID(annID string) (*model.Announcement, error) {
	id, err := strconv.Atoi(annID)
	if err != nil {
		return nil, errors.New("公告ID格式错误")
	}

	var ann model.Announcement
	err = model.DB.Where("id = ? AND visible = ?", id, true).First(&ann).Error
	if err == gorm.ErrRecordNotFound {
		return nil, errors.New("公告不存在或未公开")
	}
	if err != nil {
		return nil, errors.New("查询公告失败：" + err.Error())
	}

	return &ann, nil
}

// UpdateAnnouncement 更新公告
func UpdateAnnouncement(annID string, title, summary string, tags []string, visible bool) (*model.Announcement, error) {
	id, err := strconv.Atoi(annID)
	if err != nil {
		return nil, errors.New("公告ID格式错误")
	}

	var ann model.Announcement
	err = model.DB.Where("id = ?", id).First(&ann).Error
	if err == gorm.ErrRecordNotFound {
		return nil, errors.New("公告不存在")
	}
	if err != nil {
		return nil, errors.New("查询公告失败：" + err.Error())
	}

	// 更新字段
	if title != "" {
		ann.Title = title
	}
	if summary != "" {
		ann.Summary = summary
	}
	if len(tags) > 0 {
		ann.Tags = tags
	}
	ann.Visible = visible
	ann.Date = time.Now().Format("2006-01-02") // 重新更新发布日期

	if err := model.DB.Save(&ann).Error; err != nil {
		return nil, errors.New("更新公告失败：" + err.Error())
	}

	return &ann, nil
}

// DeleteAnnouncement 删除公告
func DeleteAnnouncement(annID string) error {
	id, err := strconv.Atoi(annID)
	if err != nil {
		return errors.New("公告ID格式错误")
	}

	if err := model.DB.Delete(&model.Announcement{}, id).Error; err != nil {
		return errors.New("删除公告失败：" + err.Error())
	}

	return nil
}
