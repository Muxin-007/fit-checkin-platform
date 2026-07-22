package support

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"platform/modules/system/internal/global"
)

const wechatAPI = "https://api.weixin.qq.com"

var wechatClient = &http.Client{Timeout: 15 * time.Second}

type CodeSession struct {
	OpenID     string `json:"openid"`
	UnionID    string `json:"unionid"`
	SessionKey string `json:"session_key"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

type apiResult struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

type AccessToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	apiResult
}

type SecurityResult struct {
	Suggest string `json:"suggest"`
	Label   int    `json:"label"`
}

type TextSecurityResponse struct {
	apiResult
	Result  SecurityResult `json:"result"`
	TraceID string         `json:"trace_id"`
}

type MediaSecurityResponse struct {
	apiResult
	TraceID string `json:"trace_id"`
}

type SubscribeResponse struct {
	apiResult
	MsgID int64 `json:"msgid"`
}

func Code2Session(ctx context.Context, code string) (*CodeSession, error) {
	if global.Cfg.System.DevelopmentMode {
		if code != "local-development" {
			return nil, errors.New("invalid local development login code")
		}
		return &CodeSession{OpenID: global.Cfg.System.DevelopmentOpenID}, nil
	}
	cfg := global.Cfg.System.Wechat
	if cfg.AppID == "" || cfg.AppSecret == "" {
		return nil, errors.New("wechat app-id/app-secret is not configured")
	}
	endpoint := wechatAPI + "/sns/jscode2session?appid=" + url.QueryEscape(cfg.AppID) +
		"&secret=" + url.QueryEscape(cfg.AppSecret) + "&js_code=" + url.QueryEscape(code) +
		"&grant_type=authorization_code"
	var result CodeSession
	if err := getJSON(ctx, endpoint, &result); err != nil {
		return nil, err
	}
	if result.ErrCode != 0 || result.OpenID == "" {
		return nil, fmt.Errorf("wechat code2session failed: %d %s", result.ErrCode, result.ErrMsg)
	}
	return &result, nil
}

func CheckText(ctx context.Context, openID, content string) (string, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "approved", nil
	}
	if global.Cfg.System.DevelopmentMode {
		return "approved", nil
	}
	var result TextSecurityResponse
	err := postWechat(ctx, "/wxa/msg_sec_check", map[string]any{
		"content": content,
		"version": 2,
		"scene":   global.Cfg.System.Wechat.ContentSecurityScene,
		"openid":  openID,
	}, &result)
	if err != nil {
		return "", err
	}
	switch result.Result.Suggest {
	case "pass":
		return "approved", nil
	case "review":
		return "pending", nil
	default:
		return "rejected", nil
	}
}

func CheckImage(ctx context.Context, openID, mediaURL string) (string, error) {
	if global.Cfg.System.PublicURL == "" {
		return "", errors.New("public-url is required for image security review")
	}
	var result MediaSecurityResponse
	err := postWechat(ctx, "/wxa/media_check_async", map[string]any{
		"media_url":  mediaURL,
		"media_type": 2,
		"version":    2,
		"scene":      global.Cfg.System.Wechat.ContentSecurityScene,
		"openid":     openID,
	}, &result)
	if err != nil {
		return "", err
	}
	if result.TraceID == "" {
		return "", errors.New("wechat media security did not return trace_id")
	}
	return result.TraceID, nil
}

func SendReminder(ctx context.Context, openID, groupName, deadline string) (int64, error) {
	cfg := global.Cfg.System.Wechat
	if cfg.ReminderTemplateID == "" {
		return 0, errors.New("reminder-template-id is not configured")
	}
	var result SubscribeResponse
	err := postWechat(ctx, "/cgi-bin/message/subscribe/send", map[string]any{
		"touser":            openID,
		"template_id":       cfg.ReminderTemplateID,
		"page":              cfg.ReminderPage,
		"miniprogram_state": "formal",
		"lang":              "zh_CN",
		"data": map[string]any{
			"thing1":  map[string]string{"value": truncateRunes(groupName, 20)},
			"phrase2": map[string]string{"value": "今天还没有完成打卡"},
			"time3":   map[string]string{"value": deadline},
		},
	}, &result)
	return result.MsgID, err
}

func GenerateInviteQRCode(ctx context.Context, code string) ([]byte, error) {
	if global.Cfg.System.DevelopmentMode {
		return developmentInviteImage(code)
	}
	token, err := accessToken(ctx)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(map[string]any{
		"scene":       code,
		"page":        global.Cfg.System.Wechat.InvitePage,
		"check_path":  false,
		"env_version": "release",
		"width":       430,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		wechatAPI+"/wxa/getwxacodeunlimit?access_token="+url.QueryEscape(token), bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := wechatClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 3*1024*1024))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wechat qr api returned http %d", resp.StatusCode)
	}
	if strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		var result apiResult
		if jsonErr := json.Unmarshal(body, &result); jsonErr != nil {
			return nil, jsonErr
		}
		return nil, fmt.Errorf("wechat qr api failed: %d %s", result.ErrCode, result.ErrMsg)
	}
	if len(body) == 0 {
		return nil, errors.New("wechat qr api returned empty image")
	}
	return body, nil
}

func developmentInviteImage(code string) ([]byte, error) {
	const size = 430
	canvas := image.NewRGBA(image.Rect(0, 0, size, size))
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	black := color.RGBA{R: 31, G: 38, B: 35, A: 255}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			canvas.Set(x, y, white)
		}
	}
	digest := sha256.Sum256([]byte(code))
	const cells = 29
	const cellSize = 12
	const offset = 41
	for row := 0; row < cells; row++ {
		for col := 0; col < cells; col++ {
			bit := uint((row*cells + col) % (len(digest) * 8))
			if digest[bit/8]&(1<<(bit%8)) == 0 {
				continue
			}
			for y := offset + row*cellSize; y < offset+(row+1)*cellSize; y++ {
				for x := offset + col*cellSize; x < offset+(col+1)*cellSize; x++ {
					canvas.Set(x, y, black)
				}
			}
		}
	}
	var output bytes.Buffer
	if err := png.Encode(&output, canvas); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func VerifyMessageSignature(token, timestamp, nonce, signature string) bool {
	if token == "" || timestamp == "" || nonce == "" || signature == "" {
		return false
	}
	parts := []string{token, timestamp, nonce}
	sort.Strings(parts)
	sum := sha1.Sum([]byte(strings.Join(parts, "")))
	return hex.EncodeToString(sum[:]) == signature
}

func postWechat(ctx context.Context, path string, body any, result interface{}) error {
	token, err := accessToken(ctx)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, wechatAPI+path+"?access_token="+url.QueryEscape(token), bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := wechatClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("wechat api returned http %d", resp.StatusCode)
	}
	if err = json.NewDecoder(resp.Body).Decode(result); err != nil {
		return err
	}
	encoded, _ := json.Marshal(result)
	var common apiResult
	_ = json.Unmarshal(encoded, &common)
	if common.ErrCode != 0 {
		return fmt.Errorf("wechat api failed: %d %s", common.ErrCode, common.ErrMsg)
	}
	return nil
}

func accessToken(ctx context.Context) (string, error) {
	const key = "fitness:wechat:access-token"
	if token, err := global.Redis.Get(ctx, key).Result(); err == nil && token != "" {
		return token, nil
	}
	cfg := global.Cfg.System.Wechat
	if cfg.AppID == "" || cfg.AppSecret == "" {
		return "", errors.New("wechat app-id/app-secret is not configured")
	}
	endpoint := wechatAPI + "/cgi-bin/token?grant_type=client_credential&appid=" +
		url.QueryEscape(cfg.AppID) + "&secret=" + url.QueryEscape(cfg.AppSecret)
	var result AccessToken
	if err := getJSON(ctx, endpoint, &result); err != nil {
		return "", err
	}
	if result.ErrCode != 0 || result.AccessToken == "" {
		return "", fmt.Errorf("wechat token failed: %d %s", result.ErrCode, result.ErrMsg)
	}
	ttl := time.Duration(result.ExpiresIn-300) * time.Second
	if ttl <= 0 {
		ttl = time.Hour
	}
	if err := global.Redis.Set(ctx, key, result.AccessToken, ttl).Err(); err != nil {
		return "", err
	}
	return result.AccessToken, nil
}

func getJSON(ctx context.Context, endpoint string, result any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := wechatClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("remote api returned http %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(result)
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
