package config

import (
	"flag"
	"os"

	"github.com/BurntSushi/toml"
)

// indexOf は文字列内の特定の文字の位置を返す
func indexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

const (
	// DefaultConfigFile はデフォルトの設定ファイル名
	DefaultConfigFile = "config.toml"
)

// Config はアプリケーション全体の設定を表す
type Config struct {
	Debug bool `toml:"debug"`
	Log   struct {
		Filename string `toml:"filename"`
	} `toml:"log"`
	WebSocket struct {
		Enabled                bool   `toml:"enabled"`
		PeriodicUpdateInterval string `toml:"periodic_update_interval"` // e.g., "1m", "30s", "0" to disable
	} `toml:"websocket"`
	TLS struct {
		Enabled  bool   `toml:"enabled"`
		CertFile string `toml:"cert_file"`
		KeyFile  string `toml:"key_file"`
	} `toml:"tls"`
	WebSocketClient struct {
		Enabled bool   `toml:"enabled"`
		Addr    string `toml:"addr"`
	} `toml:"websocket_client"`

	// Daemon mode settings
	Daemon struct {
		Enabled bool   `toml:"enabled"`
		PIDFile string `toml:"pid_file"`
	} `toml:"daemon"`
	HTTPServer struct {
		Enabled bool   `toml:"enabled"`
		Host    string `toml:"host"`
		Port    int    `toml:"port"`
		WebRoot string `toml:"web_root"`
	} `toml:"http_server"`
}

// NewConfig はデフォルト設定を持つConfigを作成する
func NewConfig() *Config {
	cfg := &Config{
		Debug: false,
	}
	cfg.Log.Filename = "echonet-list.log"
	cfg.WebSocket.PeriodicUpdateInterval = "1m" // Default to 1 minute
	cfg.WebSocketClient.Addr = "ws://localhost:8080/ws"
	// Default daemon settings
	cfg.Daemon.Enabled = false
	cfg.Daemon.PIDFile = ""
	// Default HTTP server settings
	cfg.HTTPServer.Enabled = false
	cfg.HTTPServer.Host = "localhost"
	cfg.HTTPServer.Port = 8080
	cfg.HTTPServer.WebRoot = "web/bundle"
	return cfg
}

// LoadConfig は設定を読み込む
// 以下の優先順位でロードする:
// 1. 指定されたパスの設定ファイル（指定がある場合）
// 2. カレントディレクトリのデフォルト設定ファイル（存在する場合）
// 3. デフォルト設定
func LoadConfig(configPath string) (*Config, error) {
	config := NewConfig()

	// 設定ファイルパスの解決
	filePath := configPath
	if filePath == "" {
		// 指定がなければデフォルトファイルを探す
		if _, err := os.Stat(DefaultConfigFile); err == nil {
			filePath = DefaultConfigFile
		} else {
			// デフォルトファイルもなければ、デフォルト設定をそのまま返す
			return config, nil
		}
	}

	// 設定ファイルが指定または存在する場合は読み込む
	if _, err := toml.DecodeFile(filePath, config); err != nil {
		return nil, err
	}

	return config, nil
}

// ApplyCommandLineArgs はコマンドライン引数で指定された値を設定に適用する
func (c *Config) ApplyCommandLineArgs(args CommandLineArgs) {
	// コマンドライン引数で指定された値で上書き
	if args.DebugSpecified {
		c.Debug = args.Debug
	}
	if args.LogFilenameSpecified {
		c.Log.Filename = args.LogFilename
	}
	// websocket
	if args.WebSocketEnabledSpecified {
		c.WebSocket.Enabled = args.WebSocketEnabled
	}
	// WebSocketアドレスはHTTPサーバーのポートから自動決定されるため削除
	// websocket TLS
	if args.WebSocketTLSEnabledSpecified {
		c.TLS.Enabled = args.WebSocketTLSEnabled
	}
	if args.WebSocketTLSCertFileSpecified {
		c.TLS.CertFile = args.WebSocketTLSCertFile
	}
	if args.WebSocketTLSKeyFileSpecified {
		c.TLS.KeyFile = args.WebSocketTLSKeyFile
	}
	// websocket client
	if args.WebSocketClientEnabledSpecified {
		c.WebSocketClient.Enabled = args.WebSocketClientEnabled
	}
	if args.WebSocketClientAddrSpecified {
		c.WebSocketClient.Addr = args.WebSocketClientAddr
	}
	// ws-both フラグの特殊処理
	if args.WebSocketBothSpecified && args.WebSocketBoth {
		c.WebSocket.Enabled = true
		c.WebSocketClient.Enabled = true
	}
	// Daemon mode flags
	if args.DaemonEnabledSpecified {
		c.Daemon.Enabled = args.DaemonEnabled
	}
	if args.PIDFileSpecified {
		c.Daemon.PIDFile = args.PIDFile
	}
	// HTTP server
	if args.HTTPServerEnabledSpecified {
		c.HTTPServer.Enabled = args.HTTPServerEnabled
	}
	if args.HTTPServerHostSpecified {
		c.HTTPServer.Host = args.HTTPServerHost
	}
	if args.HTTPServerPortSpecified {
		c.HTTPServer.Port = args.HTTPServerPort
	}
	if args.HTTPServerWebRootSpecified {
		c.HTTPServer.WebRoot = args.HTTPServerWebRoot
	}
}

// CommandLineArgs はコマンドライン引数からの値を保持する
type CommandLineArgs struct {
	// 設定ファイル (メタ設定)
	ConfigFile      string
	ConfigSpecified bool

	// 一般設定
	Debug          bool
	DebugSpecified bool

	// ログ設定
	LogFilename          string
	LogFilenameSpecified bool

	// WebSocketサーバー設定
	WebSocketEnabled          bool
	WebSocketEnabledSpecified bool

	// WebSocket TLS設定
	WebSocketTLSEnabled           bool
	WebSocketTLSEnabledSpecified  bool
	WebSocketTLSCertFile          string
	WebSocketTLSCertFileSpecified bool
	WebSocketTLSKeyFile           string
	WebSocketTLSKeyFileSpecified  bool

	// WebSocketクライアント設定
	WebSocketClientEnabled          bool
	WebSocketClientEnabledSpecified bool
	WebSocketClientAddr             string
	WebSocketClientAddrSpecified    bool

	// 特殊フラグ
	WebSocketBoth          bool
	WebSocketBothSpecified bool
	// Daemon mode flags
	DaemonEnabled          bool
	DaemonEnabledSpecified bool
	PIDFile                string
	PIDFileSpecified       bool

	// HTTPサーバー設定
	HTTPServerEnabled          bool
	HTTPServerEnabledSpecified bool
	HTTPServerHost             string
	HTTPServerHostSpecified    bool
	HTTPServerPort             int
	HTTPServerPortSpecified    bool
	HTTPServerWebRoot          string
	HTTPServerWebRootSpecified bool
}

// ParseCommandLineArgs はコマンドライン引数をパースする
func ParseCommandLineArgs() CommandLineArgs {
	var args CommandLineArgs

	// フラグの定義
	configFileFlag := flag.String("config", "", "TOML設定ファイルのパスを指定する")

	debugFlag := flag.Bool("debug", false, "デバッグモードを有効にする")
	logFilenameFlag := flag.String("log", "echonet-list.log", "ログファイル名を指定する")

	websocketFlag := flag.Bool("websocket", false, "WebSocketサーバーモードを有効にする")

	wsTLSFlag := flag.Bool("ws-tls", false, "WebSocketサーバーでTLSを有効にする")
	wsCertFileFlag := flag.String("ws-cert-file", "", "TLS証明書ファイルのパスを指定する")
	wsKeyFileFlag := flag.String("ws-key-file", "", "TLS秘密鍵ファイルのパスを指定する")

	wsClientFlag := flag.Bool("ws-client", false, "WebSocketクライアントモードを有効にする")
	wsClientAddrFlag := flag.String("ws-client-addr", "ws://localhost:8080/ws", "WebSocketクライアントの接続先アドレスを指定する")

	wsBothFlag := flag.Bool("ws-both", false, "WebSocketサーバーとクライアントの両方を有効にする（テスト用）")
	daemonFlag := flag.Bool("daemon", false, "デーモンモードを有効にする")
	pidFileFlag := flag.String("pidfile", "", "PIDファイルのパスを指定する")

	httpEnabledFlag := flag.Bool("http-enabled", false, "HTTPサーバーを有効にする")
	httpHostFlag := flag.String("http-host", "localhost", "HTTPサーバーのホスト名を指定する")
	httpPortFlag := flag.Int("http-port", 8080, "HTTPサーバーのポートを指定する")
	httpWebRootFlag := flag.String("http-webroot", "web/bundle", "HTTPサーバーのWebルートディレクトリを指定する")

	// コマンドライン引数を解析
	flag.Parse()

	// コマンドライン引数を直接解析して、フラグが指定されたかどうかを確認
	argsMap := make(map[string]bool)
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if len(arg) > 0 && arg[0] == '-' {
			// フラグ名を抽出 (-flag または --flag の形式)
			flagName := arg
			if len(flagName) > 1 && flagName[1] == '-' {
				flagName = flagName[2:] // --flag の場合
			} else {
				flagName = flagName[1:] // -flag の場合
			}

			// = が含まれている場合は分割
			if idx := indexOf(flagName, '='); idx >= 0 {
				flagName = flagName[:idx]
			}

			argsMap[flagName] = true

			// 次の引数が値の場合はスキップ
			if i+1 < len(os.Args) && len(os.Args[i+1]) > 0 && os.Args[i+1][0] != '-' {
				i++
			}
		}
	}

	// 値と指定有無の設定
	args.ConfigFile = *configFileFlag
	args.ConfigSpecified = argsMap["config"]

	args.Debug = *debugFlag
	args.DebugSpecified = argsMap["debug"]

	args.LogFilename = *logFilenameFlag
	args.LogFilenameSpecified = argsMap["log"]

	args.WebSocketEnabled = *websocketFlag
	args.WebSocketEnabledSpecified = argsMap["websocket"]

	args.WebSocketTLSEnabled = *wsTLSFlag
	args.WebSocketTLSEnabledSpecified = argsMap["ws-tls"]

	args.WebSocketTLSCertFile = *wsCertFileFlag
	args.WebSocketTLSCertFileSpecified = argsMap["ws-cert-file"]

	args.WebSocketTLSKeyFile = *wsKeyFileFlag
	args.WebSocketTLSKeyFileSpecified = argsMap["ws-key-file"]

	args.WebSocketClientEnabled = *wsClientFlag
	args.WebSocketClientEnabledSpecified = argsMap["ws-client"]

	args.WebSocketClientAddr = *wsClientAddrFlag
	args.WebSocketClientAddrSpecified = argsMap["ws-client-addr"]

	args.WebSocketBoth = *wsBothFlag
	args.WebSocketBothSpecified = argsMap["ws-both"]
	args.DaemonEnabled = *daemonFlag
	args.DaemonEnabledSpecified = argsMap["daemon"]
	args.PIDFile = *pidFileFlag
	args.PIDFileSpecified = argsMap["pidfile"]

	args.HTTPServerEnabled = *httpEnabledFlag
	args.HTTPServerEnabledSpecified = argsMap["http-enabled"]
	args.HTTPServerHost = *httpHostFlag
	args.HTTPServerHostSpecified = argsMap["http-host"]
	args.HTTPServerPort = *httpPortFlag
	args.HTTPServerPortSpecified = argsMap["http-port"]
	args.HTTPServerWebRoot = *httpWebRootFlag
	args.HTTPServerWebRootSpecified = argsMap["http-webroot"]

	return args
}
