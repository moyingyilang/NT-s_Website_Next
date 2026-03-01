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

// 验证JWT令牌（补全截断逻辑，跨文件调用）
func VerifyToken(tokenStr string) (*Claims, error) {
	// 处理Bearer前缀（前端常见传参格式）
	if strings.HasPrefix(tokenStr, "Bearer ") {
		tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")
	}

	// 解析token并校验签名
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 校验签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("令牌签名算法非法")
		}
		return []byte(JWTSecret), nil
	})
	if err != nil {
		return nil, errors.New("令牌无效或已过期")
	}

	// 校验token有效性并返回声明
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("令牌验证失败")
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