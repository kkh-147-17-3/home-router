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

	Dhcp struct {
		Server struct {
			RangeStart string `yaml:"range_start"`
			RangeEnd   string `yaml:"range_end"`
			Gateway    string `yaml:"gateway"`
			Dns        string `yaml:"dns"`
			LeaseTime  uint32 `yaml:"lease_time"`
		}
	}
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
