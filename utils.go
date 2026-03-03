package main

import (
    "encoding/base64"
    "encoding/json"
    "errors"
    "fmt"          // 用于 fmt.Sprintf
    "net/http"
    "regexp"       // 用于正则表达式
    "strconv"      // 用于 strconv.ParseInt
    "strings"      // 用于字符串替换

    "github.com/gorilla/sessions"
    "golang.org/x/crypto/bcrypt"
)

// ---------------------- 密码哈希（完全复刻原PHP password_hash/password_verify）----------------------
func hashPassword(password string) string {
	hashed, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hashed)
}

func verifyPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// ---------------------- 用户ID验证（复刻原PHP isValidId）----------------------
func isValidId(id string) bool {
	return regexp.MustCompile(`^\d{10}$`).MatchString(id)
}

// ---------------------- CSRF验证（复刻原PHP checkCSRF）----------------------
func checkCSRF(r *http.Request, session *sessions.Session) bool {
	csrfToken := r.PostForm.Get("_csrf")
	return session.Values["csrf_token"] != nil && csrfToken == session.Values["csrf_token"]
}

// ---------------------- 加密/解密（完全复刻原PHP encryptHash/decryptHash）----------------------
func encryptHash(hash string) string {
	// 1. Base64编码（复刻原PHP base64_encode）
	base64Str := base64.StdEncoding.EncodeToString([]byte(hash))

	// 2. 转二进制（每个字符8位）并替换1→7、0→8
	binaryStr := ""
	for _, c := range base64Str {
		// 补零为8位二进制（复刻原PHP sprintf('%08b')）
		binaryStr += fmt.Sprintf("%08b", c)
	}
	binaryStr = strings.ReplaceAll(binaryStr, "1", "7")
	binaryStr = strings.ReplaceAll(binaryStr, "0", "8")

	// 3. 压缩替换（顺序：先三连后78，复刻原PHP preg_replace）
	binaryStr = regexp.MustCompile(`777`).ReplaceAllString(binaryStr, "9")
	binaryStr = regexp.MustCompile(`888`).ReplaceAllString(binaryStr, "1")
	binaryStr = regexp.MustCompile(`78`).ReplaceAllString(binaryStr, "3")

	// 4. 固定3比特编码（复刻原PHP映射表）
	mapChar := map[rune]string{
		'1': "000", '3': "001", '7': "010", '8': "011", '9': "100",
	}
	bitStr := ""
	for _, c := range binaryStr {
		bitStr += mapChar[c]
	}

	// 5. 比特串打包成字节（8位一组，右补零，复刻原PHP chr(bindec)）
	packed := []byte{}
	for i := 0; i < len(bitStr); i += 8 {
		end := i + 8
		if end > len(bitStr) {
			end = len(bitStr)
			// 右补零（复刻原PHP str_pad(..., 8, '0', STR_PAD_RIGHT)）
			bitStr += strings.Repeat("0", 8-(end-i))
		}
		byteVal, _ := strconv.ParseInt(bitStr[i:i+8], 2, 8)
		packed = append(packed, byte(byteVal))
	}

	// 6. 头部加2字节长度（大端序，复刻原PHP pack('n')）
	lenChars := uint16(len(binaryStr))
	header := []byte{byte(lenChars >> 8), byte(lenChars & 0xFF)}

	// 7. 最终Base64编码（复刻原PHP base64_encode）
	return base64.StdEncoding.EncodeToString(append(header, packed...))
}

func decryptHash(encrypted string) (string, error) {
	// 1. Base64解码（复刻原PHP base64_decode）
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", errors.New("无效的Base64数据")
	}
	if len(data) < 2 {
		return "", errors.New("数据太短")
	}

	// 2. 解析头部长度（复刻原PHP unpack('n')）
	lenChars := uint16(data[0])<<8 | uint16(data[1])
	packed := data[2:]

	// 3. 还原比特串（每个字节8位，高位在前，复刻原PHP str_pad(decbin, 8, '0', STR_PAD_LEFT)）
	bitStrFull := ""
	for _, b := range packed {
		// 补零为8位（复刻原PHP str_pad）
		bitStrFull += fmt.Sprintf("%08b", b)
	}

	// 4. 截取有效比特（复刻原PHP substr(..., 0, $totalBitsNeeded)）
	totalBits := int(lenChars) * 3
	if len(bitStrFull) < totalBits {
		return "", errors.New("数据不足，可能损坏")
	}
	bitStr := bitStrFull[:totalBits]

	// 5. 3比特解码（复刻原PHP revMap）
	revMap := map[string]rune{
		"000": '1', "001": '3', "010": '7', "011": '8', "100": '9',
	}
	str := ""
	for i := 0; i < totalBits; i += 3 {
		triple := bitStr[i:i+3]
		c, ok := revMap[triple]
		if !ok {
			return "", errors.New("无效的比特组合: " + triple)
		}
		str += string(c)
	}

	// 6. 逆向压缩（顺序：先3→78，再9→777，再1→888，复刻原PHP preg_replace）
	str = regexp.MustCompile(`3`).ReplaceAllString(str, "78")
	str = regexp.MustCompile(`9`).ReplaceAllString(str, "777")
	str = regexp.MustCompile(`1`).ReplaceAllString(str, "888")

	// 7. 还原二进制（7→1、8→0，复刻原PHP str_replace）
	binaryStr := strings.ReplaceAll(str, "7", "1")
	binaryStr = strings.ReplaceAll(binaryStr, "8", "0")

	// 8. 二进制转字符（每8位一个，复刻原PHP chr(bindec)）
	base64Str := ""
	for i := 0; i < len(binaryStr); i += 8 {
		if i+8 > len(binaryStr) {
			break
		}
		byteVal, _ := strconv.ParseInt(binaryStr[i:i+8], 2, 8)
		base64Str += string(byteVal)
	}

	// 9. Base64解码（复刻原PHP base64_decode）
	original, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return "", errors.New("Base64解码失败")
	}
	return string(original), nil
}

// ---------------------- 字符串辅助函数（复刻原PHP htmlspecialchars/escapeHtml）----------------------
func escapeHtml(unsafe string) string {
	if unsafe == "" {
		return ""
	}
	unsafe = strings.ReplaceAll(unsafe, "&", "&amp;")
	unsafe = strings.ReplaceAll(unsafe, "<", "&lt;")
	unsafe = strings.ReplaceAll(unsafe, ">", "&gt;")
	unsafe = strings.ReplaceAll(unsafe, "\"", "&quot;")
	unsafe = strings.ReplaceAll(unsafe, "'", "&#039;")
	return unsafe
}

// ---------------------- JSON辅助函数（复刻原PHP json_encode）----------------------
func jsonEncode(data any) string {
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}
