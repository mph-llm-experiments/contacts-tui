package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds the application configuration
type Config struct {
	Database DatabaseConfig `toml:"database"`
	Tasks    TasksConfig    `toml:"tasks"`
	External ExternalConfig `toml:"external"`
}

// DatabaseConfig holds database-related configuration
type DatabaseConfig struct {
	Path string `toml:"path"`
}

// TasksConfig holds task management configuration
type TasksConfig struct {
	Backend string       `toml:"backend"` // "taskwarrior", "dstask", "things", or "none"
	Things  ThingsConfig `toml:"things"`
}

// ThingsConfig holds Things-specific configuration
type ThingsConfig struct {
	AuthToken    string `toml:"auth_token"`    // Required for task creation
	DefaultList  string `toml:"default_list"`  // Optional: default list for tasks
	TagTemplate  string `toml:"tag_template"`  // Optional: template for tags
}

// ExternalConfig holds external tool integration settings
type ExternalConfig struct {
	NotesTUI bool `toml:"notes_tui"` // Enable notes-tui integration
}

// Default returns the default configuration
func Default() *Config {
	homeDir, _ := os.UserHomeDir()
	return &Config{
		Database: DatabaseConfig{
			Path: filepath.Join(homeDir, ".config", "contacts", "contacts.db"),
		},
		Tasks: TasksConfig{
			Backend: "", // Empty means auto-detect
		},
		External: ExternalConfig{
			NotesTUI: false, // Disabled by default
		},
	}
}

// Load loads configuration from the standard location
func Load() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home dir: %w", err)
	}
	
	configPath := filepath.Join(homeDir, ".config", "contacts", "config.toml")
	return LoadFrom(configPath)
}

// LoadFrom loads configuration from a specific path
func LoadFrom(configPath string) (*Config, error) {
	// Start with defaults
	cfg := Default()
	
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// No config file, return defaults
		return cfg, nil
	}
	
	// Read and parse config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}
	
	if _, err := toml.Decode(string(data), cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}
	
	// Expand home directory in paths
	if cfg.Database.Path != "" {
		cfg.Database.Path = expandPath(cfg.Database.Path)
	}
	
	return cfg, nil
}

// expandPath expands ~ to home directory
func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		homeDir, _ := os.UserHomeDir()
		return filepath.Join(homeDir, path[1:])
	}
	return path
}

// Save saves the configuration to the standard location
func (c *Config) Save() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home dir: %w", err)
	}
	
	configDir := filepath.Join(homeDir, ".config", "contacts")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	
	configPath := filepath.Join(configDir, "config.toml")
	return c.SaveTo(configPath)
}

// SaveTo saves the configuration to a specific path
func (c *Config) SaveTo(configPath string) error {
	f, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("creating config file: %w", err)
	}
	defer f.Close()
	
	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(c); err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	
	return nil
}
