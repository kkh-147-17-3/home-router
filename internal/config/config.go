package config

import (
	"gopkg.in/yaml.v3"
	"log"
	"os"
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

	PortForwarding []struct {
		Name         string `yaml:"name"`
		Protocol     string `yaml:"protocol"`
		ExternalPort int    `yaml:"external_port"`
		InternalIP   string `yaml:"internal_ip"`
		InternalPort int    `yaml:"internal_port"`
	} `yaml:"port_forwarding"`

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

	return &config
}
