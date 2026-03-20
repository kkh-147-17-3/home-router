package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"path/filepath"
	"sync"
)

type Config struct {
	Network struct {
		Wan struct {
			MacAddress string `yaml:"mac_address"`
		}
		Lan struct {
			MacAddress string `yaml:"mac_address"`
			Subnet     string `yaml:"subnet"`
		}
	}

	PortForwarding []PortForwardEntry `yaml:"port_forwarding"`

	Dhcp struct {
		Server struct {
			RangeStart string `yaml:"range_start"`
			RangeEnd   string `yaml:"range_end"`
			Gateway    string `yaml:"gateway"`
			Dns        string `yaml:"dns"`
			LeaseTime  uint32 `yaml:"lease_time"`
		}
		LeaseFile    string             `yaml:"lease_file"`
		StaticLeases []StaticLeaseEntry `yaml:"static_leases"`
	}

	Dns struct {
		Enabled    bool     `yaml:"enabled"`
		Listen     string   `yaml:"listen"`
		Upstream   []string `yaml:"upstream"`
		Blocklists []string `yaml:"blocklists"`
		Whitelist  []string `yaml:"whitelist"`
		CacheSize  int      `yaml:"cache_size"`
		LogSize    int      `yaml:"log_size"`
	}

	Web struct {
		Enabled      bool   `yaml:"enabled"`
		Listen       string `yaml:"listen"`
		PasswordHash string `yaml:"password_hash"`
	} `yaml:"web"`

	Ddns struct {
		Enabled   bool   `yaml:"enabled"`
		Provider  string `yaml:"provider"`
		Domain    string `yaml:"domain"`
		Token     string `yaml:"token"`
		ZoneID    string `yaml:"zone_id"`
		RecordID  string `yaml:"record_id"`
		Proxied   bool   `yaml:"proxied"`
		UpdateURL string `yaml:"update_url"`
	} `yaml:"ddns"`

	Monitor struct {
		Enabled bool `yaml:"enabled"`
		LogSize int  `yaml:"log_size"`
	} `yaml:"monitor"`

	configPath string     `yaml:"-"`
	mu         sync.Mutex `yaml:"-"`
}

type PortForwardEntry struct {
	Name         string `yaml:"name"`
	Protocol     string `yaml:"protocol"`
	ExternalPort int    `yaml:"external_port"`
	InternalIP   string `yaml:"internal_ip"`
	InternalPort int    `yaml:"internal_port"`
}

type StaticLeaseEntry struct {
	Name       string `yaml:"name"`
	MacAddress string `yaml:"mac_address"`
	IP         string `yaml:"ip"`
}

func GetConfig() *Config {
	var config Config

	yamlFile, err := os.ReadFile("config.yml")
	if err != nil {
		log.Fatal(err)
	}
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Fatal(err)
	}

	absPath, err := filepath.Abs("config.yml")
	if err != nil {
		absPath = "config.yml"
	}
	config.configPath = absPath

	return &config
}

// Save writes the current config to disk atomically (temp file + rename).
func (c *Config) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("config marshal: %w", err)
	}

	dir := filepath.Dir(c.configPath)
	tmp, err := os.CreateTemp(dir, "config-*.yml")
	if err != nil {
		return fmt.Errorf("config temp file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("config write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("config close: %w", err)
	}

	if err := os.Rename(tmpName, c.configPath); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("config rename: %w", err)
	}

	return nil
}
