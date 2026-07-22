package config

import (
	"fmt"

	// Fixme: find a suitable place to import database driver
	_ "github.com/go-sql-driver/mysql" // MySQL driver
	_ "github.com/lib/pq"              // PostgreSQL driver
)

type DbDriverName string

const (
	DbDriverMySql    DbDriverName = "mysql"
	DbDriverPostgres DbDriverName = "postgres"
	DbDriverSqlite   DbDriverName = "sqlite3"
	DbDriverGremlin  DbDriverName = "gremlin"
)

type LogConfig struct {
	Director      string `default:"/var/log/platform" mapstructure:"director" yaml:"director" comment:"日志文件夹"`  // 日志文件夹
	Level         string `default:"info" mapstructure:"level" yaml:"level" comment:"日志级别"`                       // 级别
	Module        string `default:"unknown" mapstructure:"module" yaml:"module" comment:"模块名"`                   // 模块
	Format        string `default:"console" yaml:"format" comment:"日志格式 (console, json)"`                        // 输出
	ShowLine      bool   `default:"true" mapstructure:"show-line" yaml:"show-line" comment:"是否显示行"`              // 显示行
	EncodeLevel   string `default:"CapitalLevelEncoder" yaml:"encode-level"`                                     // 编码级
	StacktraceKey string `default:"stacktrace" yaml:"stacktrace-key"`                                            // 栈名
	LogInConsole  bool   `default:"true" mapstructure:"log-in-console" yaml:"log-in-console" comment:"是否在控制台输出"` // 输出控制台
	MaxAge        int64  `default:"168" mapstructure:"max-age" yaml:"max-age"  comment:"日志保留时间(小时)"`             // 日志保留时间(小时)
	RotationTime  int64  `default:"24" mapstructure:"rotation-time" yaml:"rotation-time" comment:"日志滚动时间(小时)"`   // 日志滚动时间(小时)
	FileNameExt   string `yaml:"file-name-ext"`                                                                  // 日志文件名额外字段， logPath/logModule-%s.log
}

type DbConfig struct {
	Driver  DbDriverName `default:"mysql" mapstructure:"driver" yaml:"driver" comment:"数据库类型"` // 数据库类型 (mysql, postgres)
	ConnCfg DbConnConfig `mapstructure:"conn-cfg" yaml:"conn-cfg" comment:"数据库连接配置"`
}

type DbConnConfig struct {
	Host         string `default:"mysql"  mapstructure:"host" yaml:"host" comment:"服务器地址"`                    // 服务器地址
	Port         string `default:"3306" mapstructure:"port" yaml:"port" comment:"端口"`                         // 端口
	Database     string `default:"platform" mapstructure:"database" yaml:"database" comment:"数据库名"`          // 数据库名
	Username     string `default:"root" mapstructure:"username" yaml:"username" comment:"数据库用户名"`             // 数据库用户名
	Password     string `default:"root" mapstructure:"password" yaml:"password" comment:"数据库密码"`              // 数据库密码
	TimeOut      uint32 `default:"30" mapstructure:"timeout" yaml:"timeout"`                                  // 连接超时时间
	MaxIdleConns int    `default:"10" mapstructure:"max-idle-conns" yaml:"max-idle-conns" comment:"最大空闲连接数"`  // 最大空闲连接数
	MaxOpenConns int    `default:"100" mapstructure:"max-open-conns" yaml:"max-open-conns" comment:"最大打开连接数"` // 最大打开连接数
	// pgsql
	Schema string `default:"default" mapstructure:"schema" yaml:"schema" comment:"数据库模式"` // 数据库模式
}

// Deprecated: use DbConnConfig instead. will be remove in future versions.
type MySqlConfig struct {
	Host     string `default:"mysql"  mapstructure:"host" yaml:"host" comment:"服务器地址"`           // 服务器地址
	Port     string `default:"3306" mapstructure:"port" yaml:"port" comment:"端口"`                // 端口
	Database string `default:"platform" mapstructure:"database" yaml:"database" comment:"数据库名"` // 数据库名
	Username string `default:"root" mapstructure:"username" yaml:"username" comment:"数据库用户名"`    // 数据库用户名
	Password string `default:"root" mapstructure:"password" yaml:"password" comment:"数据库密码"`     // 数据库密码
	TimeOut  uint32 `default:"30" mapstructure:"timeout" yaml:"timeout"`                         // 连接超时时间
}

// Deprecated: use DbConnConfig instead. will be remove in future versions.
type PgsqlConfig struct {
	Host     string `default:"pgsql"  mapstructure:"host" yaml:"host" comment:"服务器地址"`           // 服务器地址
	Port     string `default:"5432" mapstructure:"port" yaml:"port" comment:"端口"`                // 端口
	Database string `default:"platform" mapstructure:"database" yaml:"database" comment:"数据库名"` // 数据库名
	Schema   string `default:"rivexa" mapstructure:"schema" yaml:"schema" comment:"数据库模式"`       // 数据库模式
	Username string `default:"root" mapstructure:"username" yaml:"username" comment:"数据库用户名"`    // 数据库用户名
	Password string `default:"root" mapstructure:"password" yaml:"password" comment:"数据库密码"`     // 数据库密码
	TimeOut  uint32 `default:"30" mapstructure:"timeout" yaml:"timeout"`                         // 连接超时时间
}

// Deprecated: use DbConnConfig instead. will be remove in future versions.
func (m *PgsqlConfig) GetDSN() string {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable connect_timeout=%d options='-c search_path=%s'",
		m.Host,
		m.Port,
		m.Username,
		m.Password,
		m.Database,
		m.TimeOut,
		m.Schema,
	)
	return dsn
}

type KafkaConfig struct {
	Addr      []string   `yaml:"addr"`
	Auth      *KafkaAuth `yaml:"auth,omitempty"`
	Parititon *int32     `yaml:"parititon,omitempty"`
}

type KafkaAuth struct {
	Plain  *KafkaAuthPlain  `yaml:"plain,omitempty"`
	GSSAPI *KafkaAuthGSSAPI `yaml:"gssapi,omitempty"`
}

type KafkaAuthPlain struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type KafkaAuthGSSAPI struct {
	Config   string `yaml:"configPath"`
	Keytab   string `yaml:"keytabPath"`
	Service  string `yaml:"service"`
	Realm    string `yaml:"realm"`
	Username string `yaml:"username"`
}

// Deprecated: use DbConnConfig instead. will be remove in future versions.
func (m *MySqlConfig) GetDSN(multiStatements bool) string {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=%ds",
		m.Username,
		m.Password,
		m.Host,
		m.Port,
		m.Database,
		m.TimeOut,
	)
	if multiStatements {
		dsn += "&multiStatements=true"
	}
	return dsn
}

type RedisConfig struct {
	Addr        []string `default:"[\"127.0.0.1:6379\"]" mapstructure:"addr" yaml:"addr" json:"addr" comment:"redis服务器地址"`
	Password    string   `default:"" yaml:"password" mapstructure:"password" json:"password" comment:"redis服务器密码"`
	DB          int      `default:"0" yaml:"db" mapstructure:"db" json:"db" comment:"所用数据库"`
	OperationDB int      `default:"0" yaml:"operation-db" mapstructure:"operation-db" json:"operation-db" comment:"操作日志消息队列数据库"`
}

type MongoDBConfig struct {
	Host     string `default:"127.0.0.1" yaml:"host"`  // 主机
	Port     int    `default:"27017" yaml:"port"`      // 端口
	Username string `default:"admin" yaml:"username"`  // 用户名
	Password string `default:"123456" yaml:"password"` // 密码
	AuthDB   string `default:"admin" yaml:"authDB"`    // 认证数据库
}

type ClickhouseConfig struct {
	Database      string   `yaml:"database"`
	Host          []string `yaml:"host"`
	Port          int      `yaml:"port"`
	Username      string   `yaml:"user-name"`
	Password      string   `yaml:"password"`
	Timeout       int      `yaml:"timeout"`
	MaxConnection int      `yaml:"max-connection"`
}

type WhitePath struct {
	Path   string `yaml:"path" json:"path"`     // service prefix + path = full path
	Method string `yaml:"method" json:"method"` // GET, POST, PUT, DELETE
}

func (d *DbConfig) GetDSN(multiStatements bool) (string, error) {
	connCfg := d.ConnCfg
	switch d.Driver {
	case DbDriverMySql:
		dsn := fmt.Sprintf(
			"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=%ds",
			connCfg.Username,
			connCfg.Password,
			connCfg.Host,
			connCfg.Port,
			connCfg.Database,
			connCfg.TimeOut,
		)
		if multiStatements {
			dsn += "&multiStatements=true"
		}
		return dsn, nil
	case DbDriverPostgres:
		dsn := fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable connect_timeout=%d options='-c search_path=%s'",
			connCfg.Host,
			connCfg.Port,
			connCfg.Username,
			connCfg.Password,
			connCfg.Database,
			connCfg.TimeOut,
			connCfg.Schema,
		)
		return dsn, nil
	default:
		return "", fmt.Errorf("db driver %s not supported", d.Driver)
	}
}

func (d *DbConfig) GetDsnWithoutDatabase() (string, error) {
	connCfg := d.ConnCfg
	switch d.Driver {
	case DbDriverMySql:
		dsn := fmt.Sprintf(
			"%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=Local&timeout=%ds",
			connCfg.Username,
			connCfg.Password,
			connCfg.Host,
			connCfg.Port,
			connCfg.TimeOut,
		)
		return dsn, nil
	case DbDriverPostgres:
		dsn := fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=postgres sslmode=disable connect_timeout=%d",
			connCfg.Host,
			connCfg.Port,
			connCfg.Username,
			connCfg.Password,
			connCfg.TimeOut,
		)
		return dsn, nil
	default:
		return "", fmt.Errorf("db driver %s not supported", d.Driver)
	}
}

type MailConfig struct {
	Host               string `yaml:"host"`
	Port               int    `yaml:"port"`
	Username           string `yaml:"username"`
	Password           string `yaml:"password"`
	FromName           string `yaml:"fromName"`
	InsecureSkipVerify bool   `yaml:"insecure-skip-verify" default:"false"`
}
