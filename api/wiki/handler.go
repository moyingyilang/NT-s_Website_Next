package wiki

import (
	"io"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go-fiber-web/service"
	"go-fiber-web/utils"
)

// GetDocList 获取Wiki文档列表
func GetDocList(c *fiber.Ctx) error {
	docs, err := service.GetDocList()
	if err != nil {
		return utils.Fail(c, err.Error())
	}
	return utils.Success(c, docs)
}

// GetDocContent 获取文档内容
func GetDocContent(c *fiber.Ctx) error {
	fileName := c.Query("file_name")
	content, err := service.GetDocContent(fileName)
	if err != nil {
		return utils.Fail(c, err.Error())
	}
	return utils.Success(c, fiber.Map{"content": content})
}

// UploadDoc 上传/编辑Wiki文档
func UploadDoc(c *fiber.Ctx) error {
	fileName := c.FormValue("file_name")
	// 从表单获取内容，或从文件上传获取
	contentStr := c.FormValue("content")
	var content []byte
	if contentStr != "" {
		content = []byte(contentStr)
	} else {
		// 处理文件上传
		file, err := c.FormFile("doc")
		if err != nil {
			return utils.Fail(c, "未获取到文档内容或文件")
		}
		src, err := file.Open()
		if err != nil {
			return utils.Fail(c, "打开文档文件失败")
		}
		defer src.Close()
		content, err = io.ReadAll(src)
		if err != nil {
			return utils.Fail(c, "读取文档文件失败")
		}
		// 用上传文件名
		if fileName == "" {
			fileName = file.Filename
		}
	}

	// 清理文件名
	fileName = strings.ReplaceAll(fileName, "/", "_")
	fileName = strings.ReplaceAll(fileName, "\\", "_")
	if err := service.UploadDoc(fileName, content); err != nil {
		return utils.Fail(c, err.Error())
	}

	return utils.Success(c, fiber.Map{"msg": "文档上传成功"})
}

// DeleteDoc 删除Wiki文档
func DeleteDoc(c *fiber.Ctx) error {
	fileName := c.FormValue("file_name")
	if err := service.DeleteDoc(fileName); err != nil {
		return utils.Fail(c, err.Error())
	}
	return utils.Success(c, fiber.Map{"msg": "文档删除成功"})
}
