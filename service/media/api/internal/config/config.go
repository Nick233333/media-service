package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
	rest.RestConf
	Domains    []string `json:"Domains"`
	Standards  []string `json:"Standards"`
	MediaTypes []string `json:"MediaTypes"`
	Kafka      struct {
		Addrs      []string
		MediaTopic string
	}
}
