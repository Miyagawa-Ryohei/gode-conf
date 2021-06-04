package main

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"os"
	"path"
	"sync"
)

type ConfigOption struct {
	FileName string
	Directory string
	HotReload bool
}

var mutex = &sync.Mutex{}
var conf *viper.Viper = nil

func loadConfig(confPath string, confFile string, must bool) *viper.Viper  {
	c := viper.New()
	c.SetConfigName(confFile)
	c.AddConfigPath(confPath)
	c.SetConfigType("toml")
	if err := c.ReadInConfig(); err != nil {
		// 設定ファイルの読み取りエラー対応
		if must {
			panic(fmt.Errorf("not found %s config : %s", path.Join(confPath,confFile+".toml"),err))
		}
	}
	return c
}

func merge(defaultConf *viper.Viper, envConf *viper.Viper) *viper.Viper {
	keys := envConf.AllKeys()
	for _, key := range keys {
		defaultConf.Set(key, envConf.Get(key))
	}
	return defaultConf
}

func overrideByEnv(target *viper.Viper, override *viper.Viper) *viper.Viper {
	keys := override.AllKeys()
	for _, key := range keys {
		if envValue, ok := os.LookupEnv(fmt.Sprint(override.Get(key))); ok {
			target.Set(key, envValue)
		}
	}
	return target
}

func load(confName string, confPath string) (*viper.Viper, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	if confName == "" {
		confName = "default"
	}

	if confPath == "" {
		confPath = path.Join(cwd,"config")
	}

	defaultConf := loadConfig(confPath, "default", true)
	envConf := loadConfig(confPath, confName, false)
	override := loadConfig(confPath,"custom_env", false)
	mergedConf := merge(defaultConf,envConf)
	overrideConf := overrideByEnv(mergedConf,override)
	return overrideConf, nil
}

func LoadTo(config *interface{},option *ConfigOption) error {
	conf := Load(option)
	if err := conf.Unmarshal(config); err != nil {
		return err
	}
	return nil
}

func Load(option * ConfigOption) *viper.Viper {

	if conf != nil {
		return conf
	}
	var confName string = ""
	var confPath string = ""
	var hotReload bool = false

	if option != nil {
		confName = option.FileName
		confPath = option.Directory
		hotReload = option.HotReload
	}

	mutex.Lock()
	defer mutex.Unlock()

	c, err := load(confName, confPath)
	if err != nil {
		panic(fmt.Errorf(" %s \n", err))
	}

	// config file 変更時hot reload
	c.WatchConfig()
	if hotReload {
		c.OnConfigChange(func(e fsnotify.Event) {
			hotLoad, err := load(confName, confPath)
			if err != nil {
				panic(fmt.Errorf(" %s \n", err))
			}
			conf = hotLoad
			fmt.Println("設定ファイルが変更されました:", e.Name)
		})
	}
	conf = c
	return c
}