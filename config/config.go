package config

import "os"

// 服务配置
const (
	ServerPort = ":8080"       // 服务端口
	DBType     = "mysql"       // 数据库类型
	DBUser     = "root"        // 数据库账号
	DBPwd      = "123456"      // 数据库密码
	DBHost     = "127.0.0.1"   // 数据库地址
	DBPort     = "3306"        // 数据库端口
	DBDatabase = "ntc_chat"    // 数据库名
	DBCharset  = "utf8mb4"     // 数据库编码
)

// 目录配置
const (
	StorageDir  = "./storage"  // 根存储目录
	PackageDir  = StorageDir + "/packages" // 压缩包存储目录
	WikiDocDir  = StorageDir + "/wiki"     // Wiki文档存储目录
	AvatarDir   = StorageDir + "/avatars"  // 头像存储目录
)

// JWT配置
var (
	JWTSecret = []byte(os.Getenv("JWT_SECRET") || "ntc_chat_2025_jwt_secret") // JWT密钥
	JWTExpire = 7 * 24 * 3600                                                // JWT过期时间（7天）
)

// 初始化目录
func InitDir() {
	_ = os.MkdirAll(StorageDir, 0755)
	_ = os.MkdirAll(PackageDir, 0755)
	_ = os.MkdirAll(WikiDocDir, 0755)
	_ = os.MkdirAll(AvatarDir, 0755)
}
