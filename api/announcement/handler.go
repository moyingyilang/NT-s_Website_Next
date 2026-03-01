package announcement

import (
	"encoding/json"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"go-fiber-web/service"
	"go-fiber-web/utils"
)

// PublishAnnouncement 发布公告
func PublishAnnouncement(c *fiber.Ctx) error {
	type req struct {
		Title   string   `form:"title"`
		Summary string   `form:"summary"`
		Tags    string   `form:"tags"` // JSON字符串 ["tag1","tag2"]
		Visible bool     `form:"visible" default:"true"`
	}
	var r req
	if err := c.BodyParser(&r); err != nil {
		return utils.Fail(c, "参数错误")
	}

	// 解析Tags JSON
	var tags []string
	if r.Tags != "" {
		if err := json.Unmarshal([]byte(r.Tags), &tags); err != nil {
			return utils.Fail(c, "标签格式错误，需为JSON数组")
		}
	}

	ann, err := service.PublishAnnouncement(r.Title, r.Summary, tags, r.Visible)
	if err != nil {
		return utils.Fail(c, err.Error())
	}

	return utils.Success(c, ann)
}

// GetAnnouncements 获取公告列表
func GetAnnouncements(c *fiber.Ctx) error {
	anns, err := service.GetAnnouncements()
	if err != nil {
		return utils.Fail(c, err.Error())
	}

	return utils.Success(c, anns)
}

// GetAnnouncementByID 获取公告详情
func GetAnnouncementByID(c *fiber.Ctx) error {
	annID := c.Query("id")
	if annID == "" {
		return utils.Fail(c, "公告ID不能为空")
	}

	ann, err := service.GetAnnouncementByID(annID)
	if err != nil {
		return utils.Fail(c, err.Error())
	}

	return utils.Success(c, ann)
}

// UpdateAnnouncement 更新公告
func UpdateAnnouncement(c *fiber.Ctx) error {
	annID := c.FormValue("id")
	title := c.FormValue("title")
	summary := c.FormValue("summary")
	tagsStr := c.FormValue("tags")
	visibleStr := c.FormValue("visible")

	// 解析可见性
	visible := true
	if visibleStr != "" {
		v, err := strconv.ParseBool(visibleStr)
		if err != nil {
			return utils.Fail(c, "可见性格式错误")
		}
		visible = v
	}

	// 解析Tags
	var tags []string
	if tagsStr != "" {
		if err := json.Unmarshal([]byte(tagsStr), &tags); err != nil {
			return utils.Fail(c, "标签格式错误")
		}
	}

	ann, err := service.UpdateAnnouncement(annID, title, summary, tags, visible)
	if err != nil {
		return utils.Fail(c, err.Error())
	}

	return utils.Success(c, ann)
}

// DeleteAnnouncement 删除公告
func DeleteAnnouncement(c *fiber.Ctx) error {
	annID := c.FormValue("id")
	if annID == "" {
		return utils.Fail(c, "公告ID不能为空")
	}

	if err := service.DeleteAnnouncement(annID); err != nil {
		return utils.Fail(c, err.Error())
	}

	return utils.Success(c, fiber.Map{"msg": "公告删除成功"})
}
