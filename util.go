package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"strings"
	"time"
    "io"
	"github.com/golang-jwt/jwt/v4"
	"mime/multipart"
)

// JWT声明结构体（跨文件调用，首字母大写）
type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// 密码加密（SHA256+盐值，跨文件调用）
func HashPassword(password string) string {
	hash := sha256.New()
	hash.Write([]byte(password + JWTSecret)) // 复用model.go的全局盐值
	return hex.EncodeToString(hash.Sum(nil))
}

// 密码验证（跨文件调用）
func CheckPassword(password, hash string) bool {
	return HashPassword(password) == hash
}

// 生成JWT令牌（跨文件调用）
func GenerateToken(userID string) (string, error) {
	expireTime := time.Now().Add(time.Duration(JWTExpire) * time.Second) // 复用model.go的过期时间
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "ntc-chat",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(JWTSecret))
}

// 验证JWT令牌（集成黑名单校验，跨文件调用）
func VerifyToken(tokenStr string) (*Claims, error) {
	// 1. 处理前端可能传递的 "Bearer " 前缀（如 Authorization: Bearer <token>）
	if strings.HasPrefix(tokenStr, "Bearer ") {
		tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")
	}

	// 2. 黑名单校验：判断Token是否已被Logout标记为失效（同一package下直接访问main.go的全局变量）
	mutex.Lock() // 加锁防止并发读写冲突
	defer mutex.Unlock() // 函数结束自动解锁
	if tokenBlacklist[tokenStr] {
		return nil, errors.New("令牌已失效")
	}

	// 3. 解析JWT令牌并校验签名
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&Claims{}, // 用于接收解析后的声明
		func(token *jwt.Token) (interface{}, error) {
			// 校验签名算法是否为预期的HS256（防止算法篡改攻击）
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("令牌签名算法非法")
			}
			// 返回JWT密钥（复用model.go定义的全局变量JWTSecret，确保与GenerateToken一致）
			return []byte(JWTSecret), nil
		},
	)

	// 4. 处理解析错误（如Token格式错误、签名无效、已过期等）
	if err != nil {
		return nil, errors.New("令牌无效或已过期")
	}

	// 5. 校验Token有效性并提取Claims（用户ID等信息）
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("令牌验证失败")
	}

	// 6. 验证通过，返回解析后的用户声明
	return claims, nil
}

// 获取文件扩展名（小写，跨文件调用，main.go多处用到）
func GetFileExt(filename string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		return ""
	}
	return strings.ToLower(ext[1:]) // 去掉点并转小写，例：.PNG → png
}

// 验证是否为允许的图片MIME类型（跨文件调用，头像/图片上传用）
func IsAllowedImageType(mime string) bool {
	for _, t := range AllowedImageTypes { // 复用model.go的全局配置
		if t == mime {
			return true
		}
	}
	return false
}

// 验证是否为允许的文件扩展名（跨文件调用，分块上传用）
func IsAllowedExtension(ext string) bool {
	for _, e := range AllowedExtensions { // 复用model.go的全局配置
		if e == ext {
			return true
		}
	}
	return false
}

// 从multipart文件计算MD5（兼容文件头，和model.go方法互补，防止遗漏）
func CalculateMD5FromMultipartFile(file *multipart.FileHeader) (string, error) {
 	f, err := file.Open() // 106行的err现在会被正常使用
 	if err != nil {
 		return "", err
 	}
 	defer f.Close()
 	hash := sha256.New() // 正确定义hash变量，解决「undefined: hash」
 	// 用io.Copy替代ReadFrom，解决「hash.ReadFrom undefined」
 	if _, err := io.Copy(hash, f); err != nil {
 		return "", err
 	}
 	return hex.EncodeToString(hash.Sum(nil)), nil
 }