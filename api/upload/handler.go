package upload

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go-fiber-web/service"
	"go-fiber-web/utils"
)

// UploadChunk 上传分块接口
func UploadChunk(c *fiber.Ctx) error {
	// 获取表单参数
	userID := c.Locals("user_id").(string)
	fileID := c.FormValue("file_id")
	chunkIndexStr := c.FormValue("chunk_index")
	totalChunksStr := c.FormValue("total_chunks")
	fileName := c.FormValue("file_name")
	fileSizeStr := c.FormValue("file_size")
	chunkSizeStr := c.FormValue("chunk_size")

	// 解析数字参数
	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil {
		return utils.Fail(c, "分块索引格式错误")
	}
	totalChunks, err := strconv.Atoi(totalChunksStr)
	if err != nil {
		return utils.Fail(c, "总分块数格式错误")
	}
	fileSize, err := strconv.ParseInt(fileSizeStr, 10, 64)
	if err != nil {
		return utils.Fail(c, "文件大小格式错误")
	}
	chunkSize, err := strconv.ParseInt(chunkSizeStr, 10, 64)
	if err != nil {
		return utils.Fail(c, "分块大小格式错误")
	}

	// 获取上传文件
	file, err := c.FormFile("chunk")
	if err != nil {
		return utils.Fail(c, "未获取到分块文件")
	}
	src, err := file.Open()
	if err != nil {
		return utils.Fail(c, "打开分块文件失败")
	}
	defer src.Close()

	// 调用业务层
	err = service.UploadChunk(userID, fileID, chunkIndex, totalChunks, src.(*os.File), fileName, fileSize, chunkSize)
	if err != nil {
		return utils.Fail(c, err.Error())
	}

	return utils.Success(c, fiber.Map{"msg": "分块上传成功", "chunk_index": chunkIndex})
}

// MergeChunk 合并分块接口
func MergeChunk(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	fileID := c.FormValue("file_id")
	fileName := c.FormValue("file_name")
	totalChunksStr := c.FormValue("total_chunks")

	totalChunks, err := strconv.Atoi(totalChunksStr)
	if err != nil {
		return utils.Fail(c, "总分块数格式错误")
	}

	// 调用业务层
	file, err := service.MergeChunk(userID, fileID, fileName, totalChunks)
	if err != nil {
		return utils.Fail(c, err.Error())
	}

	return utils.Success(c, file)
}

// SearchByMD5 根据MD5查询文件是否存在
func SearchByMD5(c *fiber.Ctx) error {
	md5Str := c.Query("md5")
	if md5Str == "" {
		return utils.Fail(c, "MD5不能为空")
	}

	file, err := service.CheckFileExistByMD5(md5Str)
	if err != nil {
		return utils.Fail(c, err.Error())
	}

	return utils.Success(c, fiber.Map{"exist": file != nil, "file": file})
}

// DownloadFile 文件下载接口
func DownloadFile(c *fiber.Ctx) error {
	fileID := c.Query("file_id")
	if fileID == "" {
		return utils.Fail(c, "文件ID不能为空")
	}

	// 获取文件信息
	file, err := service.GetFileByID(fileID)
	if err != nil {
		return utils.Fail(c, err.Error())
	}

	// 设置下载响应头
	c.Set("Content-Disposition", "attachment; filename="+fiber.URLEncode(file.OriginalName))
	c.Set("Content-Type", "application/octet-stream")
	c.Set("Content-Length", strconv.FormatInt(file.Size, 10))

	// 返回文件流
	return c.SendFile(file.Path, false)
}

// ListUserFiles 获取当前用户的文件列表
func ListUserFiles(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	files, err := service.GetUserFiles(userID)
	if err != nil {
		return utils.Fail(c, err.Error())
	}

	return utils.Success(c, files)
}

// DeleteFile 删除用户上传的文件
func DeleteFile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	fileID := c.FormValue("file_id")
	if fileID == "" {
		return utils.Fail(c, "文件ID不能为空")
	}

	// 获取文件信息
	file, err := service.GetFileByID(fileID)
	if err != nil {
		return utils.Fail(c, err.Error())
	}
	// 校验文件归属
	if file.UserID != userID {
		return utils.Fail(c, "无权限删除该文件")
	}

	// 删除文件物理文件+数据库记录
	if err := os.Remove(file.Path); err != nil && !os.IsNotExist(err) {
		return utils.Fail(c, "删除物理文件失败："+err.Error())
	}
	if err := model.DB.Delete(&model.File{}, file.ID).Error; err != nil {
		return utils.Fail(c, "删除文件记录失败："+err.Error())
	}

	return utils.Success(c, fiber.Map{"msg": "文件删除成功"})
}
