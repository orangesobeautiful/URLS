package models

import (
	"URLS/internal/common"
	"URLS/internal/utils/bytestream"
	linkModels "URLS/link/models"
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// shSplit short 和 host 之間的分隔符號
const shSplit = "$"

// CurDBDBSerializerMethod 當前的將資料寫入 DB 的方式
const CurDBDBSerializerMethod = DataEncodeMethodV1

const (
	// 將資料寫入 DB 的方式
	_ byte = iota
	DataEncodeMethodV1
)

func linkInfoEncode(info *linkModels.LinkInfo) []byte {
	w := bytestream.NewWriter()
	fullDest := info.FullDest()
	w.Byte(CurDBDBSerializerMethod).
		Bool(false). // 沒被刪除
		Int32(int32(info.Type)).
		String(fullDest)

	return w.ToBytes()
}

func deleteLinkInfoEncode() []byte {
	w := bytestream.NewWriter()
	w.Byte(CurDBDBSerializerMethod).Bool(true)

	return w.ToBytes()
}

func linkInfoDecode(bs []byte) (t linkModels.LinkType, fullDest string, deleted bool, err error) {
	r := bytestream.NewReader(bs)

	if len(bs) > 0 {
		var encMethod byte
		r.Byte(&encMethod)
		switch encMethod {
		case DataEncodeMethodV1:
			var linkTypeUint32 int32
			r.Bool(&deleted)
			if deleted {
				return
			}

			r.Int32(&linkTypeUint32)
			var convOK bool
			t, convOK = linkModels.LinkTypeFromInteger(linkTypeUint32)
			if !convOK {
				err = fmt.Errorf("unknow LinkType(%d)", linkTypeUint32)
				return
			}

			r.String(&fullDest)
			if r.HasErr() {
				err = errors.New("deocde failed")
				return
			}
		default:
			err = fmt.Errorf("unknow method(%d)", bs[0])
			return
		}
	} else {
		err = errors.New("bytes len is zero")
		return
	}

	return t, fullDest, deleted, nil
}

func linkKey(short, host string) string {
	return short + shSplit + host
}

func LinkAdd(ctx context.Context, info *linkModels.LinkInfo) (err error) {
	infoBs := linkInfoEncode(info)

	var redisSetOk bool
	redisSetOk, err = redisDB.SetNX(ctx, linkKey(info.Short, info.Host), infoBs, 0).Result()
	if err != nil {
		logger.Error("redisDB.SetNX failed", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}
	if !redisSetOk {
		logger.Error("duplicate short+host want write to redisDB")
		err = common.GRPCErrInternal
		return
	}

	return
}

func LinkGetInfo(ctx context.Context, short, host string) (fullDest string, deleted, exist bool, err error) {
	linkBs, err := redisDB.Get(ctx, linkKey(short, host)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			exist = false
			err = nil
			return
		}
		logger.Error("redisDB.Get failed", zap.Error(err))
		return
	}
	_, fullDest, deleted, err = linkInfoDecode(linkBs)
	if err != nil {
		logger.Error("linkInfoDecode failed", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}

	exist = true
	return
}

// LinkDelete 將對應的 (short, host) 資料設為已被刪除
func LinkDelete(ctx context.Context, info *linkModels.LinkInfo) (err error) {
	_, err = redisDB.Set(ctx, linkKey(info.Short, info.Host), deleteLinkInfoEncode(), 0).Result()
	if err != nil {
		logger.Error("redisDB.Set failed", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}

	return nil
}
