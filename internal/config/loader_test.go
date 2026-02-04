package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestLoadConfig_Integration 是一个综合集成测试
// 它会创建一个临时配置文件，设置环境变量，然后加载配置并验证结果
func TestLoadConfig_Integration(t *testing.T) {
	// 1. 准备测试数据 (YAML 内容)
	// 故意漏掉 scanner.workers，测试默认值是否生效
	// 故意写一个 server.timeout，稍后尝试用环境变量覆盖它
	yamlContent := []byte(`
agent:
  log_level: "warn"
  data_dir: "/tmp/lfw_data"

server:
  url: "https://original-url.com"
  timeout: "5s"

scanner:
  watch_dirs:
    - "/home/user"
  rate_limit: 200

security:
  netguard:
    enable: true
    whitelist:
      - "1.1.1.1"
`)

	// 2. 创建临时配置文件
	tmpDir := t.TempDir() // Go 1.15+ 新特性，测试结束后自动清理
	tmpFile := filepath.Join(tmpDir, "config_test.yaml")
	if err := os.WriteFile(tmpFile, yamlContent, 0644); err != nil {
		t.Fatalf("Failed to create temp config file: %v", err)
	}

	// 3. 设置环境变量 (测试 Viper 的 Env 覆盖能力)
	// 我们尝试覆盖 server.url
	// 对应 loader.go 中的 SetEnvPrefix("LFW") 和 Replace(".", "_")
	// server.url -> LFW_SERVER_URL
	os.Setenv("LFW_SERVER_URL", "https://env-override.com")
	defer os.Unsetenv("LFW_SERVER_URL") // 清理环境变量

	// 4. 执行加载
	// 注意：由于 loader.go 使用了 sync.Once，这个函数在整个测试包中只能有效运行一次
	if err := LoadConfig(tmpFile); err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// 5. 获取全局配置
	cfg := Get()

	// ==========================================
	// 6. 断言验证
	// ==========================================

	// 验证 A: 配置文件中的值是否正确读取
	if cfg.Agent.LogLevel != "warn" {
		t.Errorf("Expected Agent.LogLevel 'warn', got '%s'", cfg.Agent.LogLevel)
	}
	if cfg.Scanner.RateLimit != 200 {
		t.Errorf("Expected Scanner.RateLimit 200, got %d", cfg.Scanner.RateLimit)
	}

	// 验证 B: 默认值是否生效 (ConfigFile 中没写 Workers，loader.go 默认设为 1)
	if cfg.Scanner.Workers != 1 {
		t.Errorf("Expected Scanner.Workers default 1, got %d", cfg.Scanner.Workers)
	}

	// 验证 C: 环境变量是否覆盖了配置文件
	// 文件里是 "https://original-url.com"，环境变量是 "https://env-override.com"
	// Viper 的优先级：Env > ConfigFile > Default
	if cfg.Server.URL != "https://env-override.com" {
		t.Errorf("Environment variable override failed. Expected 'https://env-override.com', got '%s'", cfg.Server.URL)
	}

	// 验证 D: 复杂类型的解析 (Duration)
	// 文件里是 "5s"
	if cfg.Server.Timeout != 5*time.Second {
		t.Errorf("Duration parsing failed. Expected 5s, got %v", cfg.Server.Timeout)
	}

	// 验证 E: 嵌套结构体与切片
	if len(cfg.Security.NetGuard.Whitelist) != 1 || cfg.Security.NetGuard.Whitelist[0] != "1.1.1.1" {
		t.Errorf("Nested slice parsing failed. Got %v", cfg.Security.NetGuard.Whitelist)
	}

	t.Logf("Config loaded successfully: %+v", cfg)
}
