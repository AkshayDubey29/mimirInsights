package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Server ServerConfig `mapstructure:"server"`
	Mimir  MimirConfig  `mapstructure:"mimir"`
	K8s    K8sConfig    `mapstructure:"k8s"`
	Log    LogConfig    `mapstructure:"log"`
	UI     UIConfig     `mapstructure:"ui"`
	LLM    LLMConfig    `mapstructure:"llm"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Port string `mapstructure:"port"`
	Host string `mapstructure:"host"`
}

// MimirConfig holds Mimir-specific configuration
type MimirConfig struct {
	Namespace string          `mapstructure:"namespace"`
	APIURL    string          `mapstructure:"api_url"`
	Timeout   int             `mapstructure:"timeout"`
	OrgID     string          `mapstructure:"org_id"`
	Discovery DiscoveryConfig `mapstructure:"discovery"`
	API       APIConfig       `mapstructure:"api"`
}

// DiscoveryConfig holds auto-discovery configuration
type DiscoveryConfig struct {
	AutoDetect        bool                `mapstructure:"auto_detect"`
	NamespacePatterns []string            `mapstructure:"namespace_patterns"`
	NamespaceLabels   []LabelSelector     `mapstructure:"namespace_labels"`
	ComponentPatterns map[string][]string `mapstructure:"component_patterns"`
	ServicePatterns   []string            `mapstructure:"service_patterns"`
	ConfigMapPatterns []string            `mapstructure:"config_map_patterns"`
}

// LabelSelector represents a label selector for discovery
type LabelSelector struct {
	Key    string   `mapstructure:"key"`
	Values []string `mapstructure:"values"`
}

// APIConfig holds API-specific configuration
type APIConfig struct {
	DistributorService string   `mapstructure:"distributor_service"`
	Port               int      `mapstructure:"port"`
	Timeout            int      `mapstructure:"timeout"`
	MetricsPaths       []string `mapstructure:"metrics_paths"`
}

// K8sConfig holds Kubernetes-specific configuration
type K8sConfig struct {
	ClusterURL   string `mapstructure:"cluster_url"`
	InCluster    bool   `mapstructure:"in_cluster"`
	ConfigPath   string `mapstructure:"config_path"`
	TenantLabel  string `mapstructure:"tenant_label"`
	TenantPrefix string `mapstructure:"tenant_prefix"`
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// UIConfig holds UI-specific configuration
type UIConfig struct {
	Theme           string `mapstructure:"theme"`
	RefreshInterval int    `mapstructure:"refresh_interval"`
}

// LLMConfig holds LLM integration configuration
type LLMConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Provider  string `mapstructure:"provider"`
	APIKey    string `mapstructure:"api_key"`
	Endpoint  string `mapstructure:"endpoint"`
	Model     string `mapstructure:"model"`
	MaxTokens int    `mapstructure:"max_tokens"`
}

var (
	// Global config instance
	globalConfig *Config
)

// Init initializes the configuration from environment variables and config files
func Init() error {
	// Set default values
	setDefaults()

	// Read from environment variables
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Read from config file if it exists
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/mimir-insights")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Unmarshal into struct
	config := &Config{}
	if err := viper.Unmarshal(config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	globalConfig = config
	return nil
}

// Get returns the global configuration
func Get() *Config {
	return globalConfig
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.host", "0.0.0.0")

	// Mimir defaults
	viper.SetDefault("mimir.namespace", "mimir")
	viper.SetDefault("mimir.api_url", "http://mimir-distributor:9090")
	viper.SetDefault("mimir.timeout", 30)
	viper.SetDefault("mimir.org_id", "default")

	// Mimir Discovery defaults
	viper.SetDefault("mimir.discovery.auto_detect", true)
	viper.SetDefault("mimir.discovery.namespace_patterns", []string{
		"mimir.*", ".*mimir.*", "cortex.*", ".*cortex.*", "observability.*", "monitoring.*",
	})
	viper.SetDefault("mimir.discovery.component_patterns.distributor", []string{".*distributor.*", ".*dist.*"})
	viper.SetDefault("mimir.discovery.component_patterns.ingester", []string{".*ingester.*", ".*ingest.*"})
	viper.SetDefault("mimir.discovery.component_patterns.querier", []string{".*querier.*", ".*query.*", ".*frontend.*"})
	viper.SetDefault("mimir.discovery.component_patterns.compactor", []string{".*compactor.*", ".*compact.*"})
	viper.SetDefault("mimir.discovery.component_patterns.ruler", []string{".*ruler.*", ".*rule.*"})
	viper.SetDefault("mimir.discovery.component_patterns.alertmanager", []string{".*alertmanager.*", ".*alert.*"})
	viper.SetDefault("mimir.discovery.component_patterns.store_gateway", []string{".*store.*gateway.*", ".*gateway.*"})
	viper.SetDefault("mimir.discovery.service_patterns", []string{
		"mimir-.*", "cortex-.*", ".*-mimir-.*", ".*-cortex-.*",
	})
	viper.SetDefault("mimir.discovery.config_map_patterns", []string{
		".*mimir.*config.*", ".*cortex.*config.*", ".*runtime.*overrides.*", ".*limits.*config.*",
	})

	// Mimir API defaults
	viper.SetDefault("mimir.api.distributor_service", "")
	viper.SetDefault("mimir.api.port", 9090)
	viper.SetDefault("mimir.api.timeout", 30)
	viper.SetDefault("mimir.api.metrics_paths", []string{"/metrics", "/api/v1/query", "/prometheus/api/v1/query"})

	// K8s defaults
	viper.SetDefault("k8s.in_cluster", true)
	viper.SetDefault("k8s.tenant_label", "team")
	viper.SetDefault("k8s.tenant_prefix", "tenant-")

	// Log defaults
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "json")

	// UI defaults
	viper.SetDefault("ui.theme", "dark")
	viper.SetDefault("ui.refresh_interval", 30)

	// LLM defaults
	viper.SetDefault("llm.enabled", false)
	viper.SetDefault("llm.provider", "openai")
	viper.SetDefault("llm.model", "gpt-4")
	viper.SetDefault("llm.max_tokens", 1000)
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	if config.Mimir.Namespace == "" {
		return fmt.Errorf("mimir namespace is required")
	}

	if config.Mimir.APIURL == "" {
		return fmt.Errorf("mimir API URL is required")
	}

	if config.K8s.TenantLabel == "" {
		return fmt.Errorf("tenant label is required")
	}

	// Validate log level
	validLogLevels := []string{"debug", "info", "warn", "error", "fatal", "panic"}
	logLevelValid := false
	for _, level := range validLogLevels {
		if config.Log.Level == level {
			logLevelValid = true
			break
		}
	}
	if !logLevelValid {
		return fmt.Errorf("invalid log level: %s", config.Log.Level)
	}

	return nil
}

// GetEnvWithDefault gets an environment variable with a default value
func GetEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// IsDevelopment returns true if running in development mode
func IsDevelopment() bool {
	return GetEnvWithDefault("ENV", "production") == "development"
}
