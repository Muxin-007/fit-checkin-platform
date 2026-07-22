package config

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	commonConfig "platform/common/config"
	"platform/common/tools"
)

type SystemConfig struct {
	RouterPrefix      string                   `yaml:"router-prefix"`
	NodeID            int                      `yaml:"node-id" default:"1"`
	DbConfig          commonConfig.DbConfig    `yaml:"db"`
	RedisCfg          commonConfig.RedisConfig `yaml:"redis"`
	Oss               Oss                      `yaml:"oss"`
	Wechat            Wechat                   `yaml:"wechat"`
	SessionTTL        string                   `yaml:"session-ttl" default:"720h"`
	AdminKey          string                   `yaml:"admin-key"`
	PublicURL         string                   `yaml:"public-url"`
	Timezone          string                   `yaml:"timezone" default:"Asia/Shanghai"`
	DevelopmentMode   bool                     `yaml:"development-mode"`
	DevelopmentOpenID string                   `yaml:"development-openid"`
}

type Wechat struct {
	AppID                string `yaml:"app-id"`
	AppSecret            string `yaml:"app-secret"`
	ReminderTemplateID   string `yaml:"reminder-template-id"`
	ReminderPage         string `yaml:"reminder-page" default:"pages/index/index"`
	InvitePage           string `yaml:"invite-page" default:"pages/invite/index"`
	ReminderScanInterval string `yaml:"reminder-scan-interval" default:"1m"`
	ContentSecurityScene int    `yaml:"content-security-scene" default:"2"`
	MessageToken         string `yaml:"message-token"`
}

type Config struct {
	System SystemConfig `yaml:"system"`
}

type requiredValue struct {
	key   string
	value string
}

// Validate implement Loader interface
func (c *Config) Validate() error {
	required := []requiredValue{
		{"system.router-prefix", c.System.RouterPrefix},
		{"system.public-url", c.System.PublicURL},
		{"system.admin-key", c.System.AdminKey},
		{"system.db.conn-cfg.host", c.System.DbConfig.ConnCfg.Host},
		{"system.db.conn-cfg.username", c.System.DbConfig.ConnCfg.Username},
		{"system.db.conn-cfg.database", c.System.DbConfig.ConnCfg.Database},
	}
	if c.System.DevelopmentMode {
		required = append(required, requiredValue{"system.development-openid", c.System.DevelopmentOpenID})
	} else {
		required = append(required,
			requiredValue{"system.wechat.app-id", c.System.Wechat.AppID},
			requiredValue{"system.wechat.app-secret", c.System.Wechat.AppSecret},
			requiredValue{"system.wechat.message-token", c.System.Wechat.MessageToken},
		)
	}
	for _, item := range required {
		if strings.TrimSpace(item.value) == "" {
			return fmt.Errorf("%s is required", item.key)
		}
	}
	publicURL, err := url.Parse(c.System.PublicURL)
	if err != nil || publicURL.Host == "" || (publicURL.Scheme != "http" && publicURL.Scheme != "https") {
		return fmt.Errorf("system.public-url must be an absolute HTTP(S) URL")
	}
	if c.System.DevelopmentMode {
		hostname := publicURL.Hostname()
		ip := net.ParseIP(hostname)
		if hostname != "localhost" && (ip == nil || !ip.IsLoopback()) {
			return fmt.Errorf("system.development-mode may only be enabled with a loopback public-url")
		}
	}
	if len(c.System.AdminKey) < 16 {
		return fmt.Errorf("system.admin-key must contain at least 16 characters")
	}
	if len(c.System.RedisCfg.Addr) == 0 || strings.TrimSpace(c.System.RedisCfg.Addr[0]) == "" {
		return fmt.Errorf("system.redis.addr is required")
	}
	supportedStorage := map[string]bool{
		"local": true, "qiniu": true, "tencent-cos": true, "aliyun-oss": true,
		"huawei-obs": true, "aws-s3": true, "cloudflare-r2": true, "minio": true,
	}
	if !supportedStorage[c.System.Oss.Type] {
		return fmt.Errorf("system.oss.type is unsupported")
	}
	if c.System.Oss.Type == "local" && strings.TrimSpace(c.System.Oss.Local.StorePath) == "" {
		return fmt.Errorf("system.oss.local.store-path is required")
	}
	sessionTTL, err := tools.ParseDuration(c.System.SessionTTL)
	if err != nil || sessionTTL <= 0 {
		return fmt.Errorf("system.session-ttl must be a positive duration")
	}
	scanInterval, err := tools.ParseDuration(c.System.Wechat.ReminderScanInterval)
	if err != nil || scanInterval < time.Minute {
		return fmt.Errorf("system.wechat.reminder-scan-interval must be at least 1m")
	}
	if _, err = time.LoadLocation(c.System.Timezone); err != nil {
		return fmt.Errorf("system.timezone is invalid: %w", err)
	}
	return nil
}

// LoadConfig from file
func LoadConfig(path string) (*Config, error) {
	return commonConfig.Load[*Config](path)
}
