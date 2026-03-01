package main

import (
	"os"
	"path/filepath"
	"strconv"
    "time"
    "fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// 统一响应结构体
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func success(c *fiber.Ctx, data interface{}) error {
	return c.JSON(Response{200, "成功", data})
}

func fail(c *fiber.Ctx, msg string) error {
	return c.JSON(Response{400, msg, nil})
}

// JWT中间件
func jwtMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenStr := c.Cookies("ntc_token") // 从Cookie获取token，更安全
		if tokenStr == "" {
			return fail(c, "未登录")
		}

		claims, err := ParseToken(tokenStr)
		if err != nil {
			return fail(c, "登录过期")
		}

		c.Locals("user_id", claims.UserID)
		return c.Next()
	}
}

func main() {
	// 初始化DB
	InitDB()

	// 创建Fiber实例
	app := fiber.New()
	app.Use(recover.New(), cors.New())

	// 静态文件（前端页面）
	app.Static("/", "./web")
	// 暴露存储目录（图片/文件访问）
	app.Static("/storage", "./storage")

	// 公开接口
	public := app.Group("/api/public")

	// 登录/注册
	public.Post("/register", func(c *fiber.Ctx) error {
		username := c.FormValue("username")
		password := c.FormValue("password")
		if username == "" || password == "" {
			return fail(c, "用户名和密码不能为空")
		}
		user, err := Register(username, password)
		if err != nil {
			return fail(c, err.Error())
		}
		token, _ := GenerateToken(user.ID)
		// 设置Cookie（有效期24小时）
		c.Cookie(&fiber.Cookie{
			Name:     "ntc_token",
			Value:    token,
			Expires:  time.Now().Add(time.Second * JWTExpire),
			HTTPOnly: true,
			SameSite: "Lax",
		})
		return success(c, user)
	})

	public.Post("/login", func(c *fiber.Ctx) error {
		username := c.FormValue("username")
		password := c.FormValue("password")
		token, user, err := Login(username, password)
		if err != nil {
			return fail(c, err.Error())
		}
		// 设置Cookie
		c.Cookie(&fiber.Cookie{
			Name:     "ntc_token",
			Value:    token,
			Expires:  time.Now().Add(time.Second * JWTExpire),
			HTTPOnly: true,
			SameSite: "Lax",
		})
		return success(c, user)
	})

	public.Post("/logout", func(c *fiber.Ctx) error {
		// 清除Cookie
		c.Cookie(&fiber.Cookie{
			Name:    "ntc_token",
			Value:   "",
			Expires: time.Now().Add(-time.Hour),
		})
		return success(c, "退出成功")
	})

	// 公告接口
	public.Get("/announcements", func(c *fiber.Ctx) error {
		anns, err := GetAnnouncements()
		if err != nil {
			return fail(c, err.Error())
		}
		return success(c, anns)
	})

	// Wiki接口（公开访问）
	public.Get("/wiki/docs", func(c *fiber.Ctx) error {
		docs, err := GetWikiDocs()
		if err != nil {
			return fail(c, err.Error())
		}
		return success(c, docs)
	})

	public.Get("/wiki/doc", func(c *fiber.Ctx) error {
		docName := c.Query("name")
		content, err := GetWikiDocContent(docName)
		if err != nil {
			return fail(c, err.Error())
		}
		return success(c, content)
	})

	// 需登录接口
	auth := app.Group("/api", jwtMiddleware())

	// 用户信息
	auth.Get("/user/info", func(c *fiber.Ctx) error {
		uid := c.Locals("user_id").(string)
		user, err := GetUserByID(uid)
		if err != nil {
			return fail(c, err.Error())
		}
		return success(c, user)
	})

	// 好友相关
	auth.Post("/friend/request", func(c *fiber.Ctx) error {
		uid := c.Locals("user_id").(string)
		targetID := c.FormValue("target_id")
		if err := SendFriendRequest(uid, targetID); err != nil {
			return fail(c, err.Error())
		}
		return success(c, "申请已发送")
	})

	auth.Post("/friend/handle", func(c *fiber.Ctx) error {
		uid := c.Locals("user_id").(string)
		fromID := c.FormValue("from_id")
		accept := c.FormValue("accept") == "true"
		if err := HandleFriendRequest(uid, fromID, accept); err != nil {
			return fail(c, err.Error())
		}
		return success(c, "处理成功")
	})

	auth.Get("/friends", func(c *fiber.Ctx) error {
		uid := c.Locals("user_id").(string)
		friends, err := GetFriendList(uid)
		if err != nil {
			return fail(c, err.Error())
		}
		return success(c, friends)
	})

	auth.Get("/friend/requests", func(c *fiber.Ctx) error {
		uid := c.Locals("user_id").(string)
		reqs, err := GetFriendRequests(uid)
		if err != nil {
			return fail(c, err.Error())
		}
		return success(c, reqs)
	})

	// 聊天相关
	auth.Post("/message/send", func(c *fiber.Ctx) error {
		uid := c.Locals("user_id").(string)
		toUID := c.FormValue("to_uid")
		content := c.FormValue("content")
		typ := c.FormValue("type", MessageTypeText)
		if toUID == "" || content == "" {
			return fail(c, "参数不能为空")
		}
		if err := SaveMessage(uid, toUID, content, typ); err != nil {
			return fail(c, err.Error())
		}
		return success(c, "发送成功")
	})

	auth.Get("/messages", func(c *fiber.Ctx) error {
		uid := c.Locals("user_id").(string)
		friendID := c.Query("friend_id")
		limit, _ := strconv.Atoi(c.Query("limit", "20"))
		msgs, err := GetMessages(uid, friendID, limit)
		if err != nil {
			return fail(c, err.Error())
		}
		return success(c, msgs)
	})

	// 文件上传相关
	auth.Post("/upload/file", func(c *fiber.Ctx) error {
		uid := c.Locals("user_id").(string)
		file, err := c.FormFile("file")
		if err != nil {
			return fail(c, "未选择文件")
		}
		src, err := file.Open()
		if err != nil {
			return fail(c, "打开文件失败")
		}
		defer src.Close()

		fileModel, err := UploadFile(uid, src.(*os.File), file.Filename)
		if err != nil {
			return fail(c, err.Error())
		}
		return success(c, fiber.Map{
			"url": "/storage/" + filepath.Base(fileModel.Path),
			"md5": fileModel.MD5,
		})
	})

	auth.Post("/upload/chat-image", func(c *fiber.Ctx) error {
		uid := c.Locals("user_id").(string)
		file, err := c.FormFile("image")
		if err != nil {
			return fail(c, "未选择图片")
		}
		src, err := file.Open()
		if err != nil {
			return fail(c, "打开图片失败")
		}
		defer src.Close()

		path, err := UploadChatImage(uid, src.(*os.File), file.Filename)
		if err != nil {
			return fail(c, err.Error())
		}
		return success(c, fiber.Map{
			"url": "/storage/" + filepath.Base(path),
		})
	})
	
	// 上传头像接口
auth.Post("/user/upload-avatar", func(c *fiber.Ctx) error {
    uid := c.Locals("user_id").(string)
    file, err := c.FormFile("avatar")
    if err != nil {
        return fail(c, "未选择头像")
    }
    src, err := file.Open()
    if err != nil {
        return fail(c, "打开头像失败")
    }
    defer src.Close()

    path, err := UploadUserAvatar(uid, src.(*os.File))
    if err != nil {
        return fail(c, err.Error())
    }
    return success(c, fiber.Map{"path": path})
})

// 更新用户资料接口
auth.Post("/user/update", func(c *fiber.Ctx) error {
    uid := c.Locals("user_id").(string)
    updateData := make(map[string]interface{})

    // 解析表单数据
    if oldPwd := c.FormValue("old_password"); oldPwd != "" {
        updateData["old_password"] = oldPwd
        updateData["password"] = c.FormValue("password")
    }
    if nickname := c.FormValue("nickname"); nickname != "" {
        updateData["nickname"] = nickname
    }
    if verifyMode := c.FormValue("verify_mode"); verifyMode != "" {
        updateData["verify_mode"] = verifyMode
    }
    if bio := c.FormValue("bio"); bio != "" {
        updateData["bio"] = bio
    }

    user, err := UpdateUser(uid, updateData)
    if err != nil {
        return fail(c, err.Error())
    }
    return success(c, user)
})

// 搜索用户接口
auth.Get("/user/search", func(c *fiber.Ctx) error {
    uid := c.Locals("user_id").(string)
    targetID := c.Query("userId")
    if targetID == uid {
        return fail(c, "不能搜索自己")
    }

    user, err := GetUserByID(targetID)
    if err != nil {
        return fail(c, "用户不存在")
    }
    return success(c, fiber.Map{
        "id":        user.ID,
        "username":  user.Username,
        "nickname":  user.Nickname,
        "bio":       user.Bio,
        "registered": user.Registered,
    })
})

// 删除好友接口
auth.Post("/friend/delete", func(c *fiber.Ctx) error {
    uid := c.Locals("user_id").(string)
    friendID := c.FormValue("friendId")
    if err := DeleteFriend(uid, friendID); err != nil {
        return fail(c, err.Error())
    }
    return success(c, "删除成功")
})


	// 启动服务
	fmt.Println("服务启动成功：http://127.0.0.1:8080")
	app.Listen(":8080")
}
