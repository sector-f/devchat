package main

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type config struct {
	Port        int    `yaml:"port"`
	Scrollback  int    `yaml:"scrollback"`
	BanFilename string `yaml:"ban_filename"`
	KeyFilename string `yaml:"key_filename"`
}

func defaultConfig() config {
	c := config{
		Port:        2222,
		Scrollback:  16,
		BanFilename: "bans.json",
		KeyFilename: "id_rsa",
	}

	return c
}

func configFromFile(filename string) (config, error) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return config{}, err
	}

	var conf config

	err = yaml.Unmarshal(file, &conf)
	return conf, err
}
