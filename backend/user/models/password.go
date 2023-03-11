package models

import "golang.org/x/crypto/bcrypt"

// PHType 密碼 Hash 的方法
type PHType uint8

const (
	_ PHType = iota
	PHBcrypt

	// PHDefaultType 預設使用的密碼 Hash 方式
	PHDefaultType = PHBcrypt
)

// PwdFormatCheck 檢查密碼格式是否符合標準
func PwdFormatCheck(pwdOrg string) bool {
	const pwdMinLen = 8
	const pwdMaxLen = 50
	pwdLen := len(pwdOrg)
	if pwdLen < pwdMinLen || pwdLen > pwdMaxLen {
		return false
	}
	return true
}

// PwdHash 使用預設方法 Hash 密碼
func PwdHash(pwdOrg string) (PHType, []byte, error) {
	var pwdHash []byte
	var phType PHType
	var err error

	switch PHDefaultType {
	case PHBcrypt:
		phType = PHBcrypt
		pwdHash, err = bcrypt.GenerateFromPassword([]byte(pwdOrg), bcrypt.DefaultCost)
	default:
		panic("unknow PHDefaultType")
	}

	return phType, pwdHash, err
}

// IsPwdEqual 比較原始密碼與 Hash 過的密碼是否相同
func IsPwdEqual(phType PHType, pwdHash, pwdOrg []byte) bool {
	var equal = false

	switch phType {
	case PHBcrypt:
		err := bcrypt.CompareHashAndPassword(pwdHash, pwdOrg)
		if err == nil {
			equal = true
		}
	default:
		panic("unknow PHDefaultType")
	}

	return equal
}
