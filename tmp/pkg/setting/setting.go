package setting

import (
	"log"
	"time"

	"github.com/go-ini/ini"
)

type App struct {
	JwtSecret string
	PageSize  int
	PrefixUrl string

	RuntimeRootPath string

	// // ImageSavePath  string
	// // ImageMaxSize   int
	// // ImageAllowExts []string

	// ExportSavePath string
	// QrCodeSavePath string
	// FontSavePath   string

	// LogSavePath string
	// LogSaveName string
	// LogFileExt  string
	TimeFormat string
}

type Server struct {
	RunMode      string
	HttpPort     int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type Database struct {
	Type     string
	User     string
	Password string
	Host     string
	Name     string
	// TablePrefix string
}

type Redis struct {
	Host        string
	Password    string
	MaxIdle     int
	MaxActive   int
	IdleTimeout time.Duration
}

// global settings
var (
	AppSetting      = &App{}
	ServerSetting   = &Server{}
	DatabaseSetting = &Database{}
	RedisSetting    = &Redis{}
)

var cfg *ini.File

// Setup initialize the configuration instance
func Setup(configPath string) {
	var err error
	cfg, err = ini.Load(configPath)
	if err != nil {
		log.Fatalf("setting.Setup, fail to parse 'conf/app.ini': %v", err)
	}

	// parse
	mapTo("app", AppSetting)
	mapTo("server", ServerSetting)
	mapTo("database", DatabaseSetting)
	mapTo("redis", RedisSetting)

	// AppSetting.ImageMaxSize = AppSetting.ImageMaxSize * 1024 * 1024
	ServerSetting.ReadTimeout = ServerSetting.ReadTimeout * time.Second
	ServerSetting.WriteTimeout = ServerSetting.WriteTimeout * time.Second
	RedisSetting.IdleTimeout = RedisSetting.IdleTimeout * time.Second
}

// mapTo map section
func mapTo(section string, v interface{}) {
	err := cfg.Section(section).MapTo(v)
	if err != nil {
		log.Fatalf("Cfg.MapTo %s err: %v", section, err)
	}
}
