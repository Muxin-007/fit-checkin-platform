package support

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	ginResp "platform/common/gin/response"
	"platform/common/tools"
	"platform/modules/system/internal/global"
	fitnessResp "platform/modules/system/internal/models/response"
)

const (
	userIDContextKey = "fitness-user-id"
	sessionKeyPrefix = "fitness:session:"
	userSessionsKey  = "fitness:user-sessions:"
)

func CreateSession(ctx context.Context, userID string) (string, time.Time, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", time.Time{}, err
	}
	token := hex.EncodeToString(raw)
	ttl, err := tools.ParseDuration(global.Cfg.System.SessionTTL)
	if err != nil || ttl <= 0 {
		ttl = 30 * 24 * time.Hour
	}
	pipeline := global.Redis.TxPipeline()
	pipeline.Set(ctx, sessionKeyPrefix+token, userID, ttl)
	pipeline.SAdd(ctx, userSessionsKey+userID, token)
	pipeline.Expire(ctx, userSessionsKey+userID, ttl)
	if _, err = pipeline.Exec(ctx); err != nil {
		return "", time.Time{}, err
	}
	return token, time.Now().Add(ttl), nil
}

func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if !strings.HasPrefix(header, "Bearer ") {
			ginResp.RespWithCode(fitnessResp.SessionInvalid, c)
			c.Abort()
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		userID, err := global.Redis.Get(c, sessionKeyPrefix+token).Result()
		if err != nil || userID == "" {
			ginResp.RespWithCode(fitnessResp.SessionInvalid, c)
			c.Abort()
			return
		}
		c.Set(userIDContextKey, userID)
		c.Next()
	}
}

func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		configured := global.Cfg.System.AdminKey
		provided := c.GetHeader("X-Admin-Key")
		if configured == "" || len(configured) != len(provided) ||
			subtle.ConstantTimeCompare([]byte(configured), []byte(provided)) != 1 {
			ginResp.RespWithCode(fitnessResp.PermissionDenied, c)
			c.Abort()
			return
		}
		c.Next()
	}
}

func UserID(c *gin.Context) string {
	value, _ := c.Get(userIDContextKey)
	userID, _ := value.(string)
	return userID
}

func DeleteSession(ctx context.Context, authorization string) {
	token := strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer "))
	if token == "" {
		return
	}
	userID, _ := global.Redis.Get(ctx, sessionKeyPrefix+token).Result()
	pipeline := global.Redis.TxPipeline()
	pipeline.Del(ctx, sessionKeyPrefix+token)
	if userID != "" {
		pipeline.SRem(ctx, userSessionsKey+userID, token)
	}
	_, _ = pipeline.Exec(ctx)
}

func RevokeUserSessions(ctx context.Context, userID string) {
	if userID == "" {
		return
	}
	tokens, _ := global.Redis.SMembers(ctx, userSessionsKey+userID).Result()
	keys := make([]string, 0, len(tokens)+1)
	for _, token := range tokens {
		keys = append(keys, sessionKeyPrefix+token)
	}
	keys = append(keys, userSessionsKey+userID)
	_ = global.Redis.Del(ctx, keys...).Err()
}

func SignFileID(id, scope string, expiresAt int64) string {
	mac := hmac.New(sha256.New, []byte(global.Cfg.System.Wechat.AppSecret))
	_, _ = mac.Write([]byte(id + ":" + scope + ":" + strconv.FormatInt(expiresAt, 10)))
	return hex.EncodeToString(mac.Sum(nil))
}

func VerifyFileSignature(id, scope string, expiresAt int64, signature string) bool {
	if expiresAt <= time.Now().Unix() {
		return false
	}
	expected := SignFileID(id, scope, expiresAt)
	return len(expected) == len(signature) &&
		subtle.ConstantTimeCompare([]byte(expected), []byte(signature)) == 1
}
