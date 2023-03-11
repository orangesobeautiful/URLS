package sessions

import (
	"context"
	"errors"
	"net/http"
	"time"

	"URLS/internal/common"
	"URLS/internal/utils/bytesext"
	"URLS/internal/utils/bytestream"
	"URLS/internal/utils/strconvext"
	"URLS/user/configs"

	"github.com/gorilla/securecookie"
	"github.com/redis/go-redis/v9"
)

const (
	// 將資料寫入 DB 的方式
	_                    byte = iota
	DBSerializerMethodV1      // json encode & decode
)

const (
	// cookie 加解密的方式
	CookieMethodV1 = '1'
)

const sessionRedisPrefix = ""

// sessionExpTime session 的過期時間(30 days)
const sessionExpTime = 30 * 24 * time.Hour

func GenerateSession(
	ctx context.Context,
	redisDB *redis.Client,
	cookieCfg configs.AuthCookieInfo,
	cookieCodes []securecookie.Codec,
	userIDHex string) (authCookie *http.Cookie, err error) {
	// 生成 session 並將其寫入 redis DB

	const randKeySize = 32
	randKey, err := bytesext.Rand(randKeySize)
	if err != nil {
		return
	}
	sessionID := strconvext.B2S(randKey)

	w := bytestream.NewWriter()
	w.Byte(DBSerializerMethodV1)
	w.String(userIDHex)

	var redisSetOk bool
	redisSetOk, err = redisDB.SetNX(ctx, sessionRedisPrefix+sessionID, w.ToBytes(), sessionExpTime).Result()
	if err != nil {
		return
	}
	if !redisSetOk {
		err = errors.New("duplicate sessionID")
		return
	}

	// 將 sessionID 和其餘內容加密寫入至 cookie

	encoded, err := securecookie.EncodeMulti(common.SigninCookieName, sessionID, cookieCodes...)
	if err != nil {
		return
	}
	encoded += string(CookieMethodV1)

	authCookie = &http.Cookie{
		Name:     common.SigninCookieName,
		Value:    encoded,
		Domain:   cookieCfg.Domain,
		Secure:   !cookieCfg.DisableSecure,
		SameSite: cookieCfg.GetHTTPSameSite(),
		Expires:  time.Now().Add(sessionExpTime),
	}

	return authCookie, nil
}

// GetSessionIDFromCookie 從 cookie 中解析 sessionID
func GetSessionIDFromCookie(
	ctx context.Context,
	cookieCodes []securecookie.Codec,
	signinCookieValue string) string {
	// 從 cookie 中解析 sessionID

	var sessionID string
	cookieValueLen := len(signinCookieValue)
	if cookieValueLen == 0 {
		return ""
	}
	switch signinCookieValue[cookieValueLen-1] {
	case CookieMethodV1:
		err := securecookie.DecodeMulti(common.SigninCookieName, signinCookieValue[:cookieValueLen-1], &sessionID, cookieCodes...)
		if err != nil {
			return ""
		}
	default:
		return ""
	}

	return sessionID
}

// GetSessionValues 根據 sessionID 取得 session Values
func GetSessionValues(
	ctx context.Context,
	redisDB *redis.Client, sessionID string) (val common.SigninSessValsInfo, err error) {
	// 從 DB 讀取 session 的內容

	keyExist := true
	valuesBytes, err := redisDB.Get(ctx, sessionRedisPrefix+sessionID).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			keyExist = false
			err = nil
		} else {
			return
		}
	}

	if !keyExist {
		return
	}

	if len(valuesBytes) == 0 {
		return
	}

	// 解析 DB 內容

	var resValues common.SigninSessValsInfo
	switch valuesBytes[0] {
	case DBSerializerMethodV1:
		r := bytestream.NewReader(valuesBytes[1:])
		var userIDHex string
		r.String(&userIDHex)
		if r.HasErr() {
			// TODO: Add Logger
			return
		}
		resValues.UserIDHex = userIDHex
	default:
		// TODO: Add Logger
		_, _ = redisDB.Del(ctx, sessionRedisPrefix+sessionID).Result()
		return
	}

	return resValues, nil
}

func DelSession(ctx context.Context, redisDB *redis.Client, sessionID string) (toDelCookie *http.Cookie, err error) {
	// delete session from db
	_, err = redisDB.Del(ctx, sessionRedisPrefix+sessionID).Result()
	if err != nil {
		return
	}

	//  delete client cookie
	toDelCookie = &http.Cookie{
		Name:   common.SigninCookieName,
		Value:  "",
		MaxAge: -1,
	}
	return
}
