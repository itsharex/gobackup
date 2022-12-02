package config

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/spf13/viper"

	"github.com/huacnlee/gobackup/logger"
)

var (
	// Exist Is config file exist
	Exist bool
	// Models configs
	Models []ModelConfig
	// HomeDir of user
	HomeDir = os.Getenv("HOME")
)

// ModelConfig for special case
type ModelConfig struct {
	Name         string
	TempPath     string
	DumpPath     string
	CompressWith SubConfig
	EncryptWith  SubConfig
	Archive      *viper.Viper
	Databases    map[string]SubConfig
	Storages     map[string]SubConfig
	Viper        *viper.Viper
}

// SubConfig sub config info
type SubConfig struct {
	Name  string
	Type  string
	Viper *viper.Viper
}

// loadConfig from:
// - ./gobackup.yml
// - ~/.gobackup/gobackup.yml
// - /etc/gobackup/gobackup.yml
func Init(configFile string) {
	viper.SetConfigType("yaml")

	// set config file directly
	if len(configFile) > 0 {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("gobackup")

		// ./gobackup.yml
		viper.AddConfigPath(".")
		// ~/.gobackup/gobackup.yml
		viper.AddConfigPath("$HOME/.gobackup") // call multiple times to add many search paths
		// /etc/gobackup/gobackup.yml
		viper.AddConfigPath("/etc/gobackup/") // path to look for the config file in
	}

	err := viper.ReadInConfig()
	if err != nil {
		logger.Error("Load gobackup config faild: ", err)
		return
	}

	viper.SetDefault("workdir", path.Join(os.TempDir(), "gobackup"))

	Exist = true
	Models = []ModelConfig{}
	for key := range viper.GetStringMap("models") {
		Models = append(Models, loadModel(key))
	}

	if len(Models) == 0 {
		logger.Fatalf("No model found in %s", viper.ConfigFileUsed())
	}
}

func loadModel(key string) (model ModelConfig) {
	model.Name = key
	model.TempPath = path.Join(viper.GetString("workdir"), fmt.Sprintf("%d", time.Now().UnixNano()))
	model.DumpPath = path.Join(model.TempPath, key)
	model.Viper = viper.Sub("models." + key)

	model.CompressWith = SubConfig{
		Type:  model.Viper.GetString("compress_with.type"),
		Viper: model.Viper.Sub("compress_with"),
	}

	model.EncryptWith = SubConfig{
		Type:  model.Viper.GetString("encrypt_with.type"),
		Viper: model.Viper.Sub("encrypt_with"),
	}

	model.Archive = model.Viper.Sub("archive")

	loadDatabasesConfig(&model)
	loadStoragesConfig(&model)

	return
}

func loadDatabasesConfig(model *ModelConfig) {
	subViper := model.Viper.Sub("databases")
	model.Databases = map[string]SubConfig{}
	for key := range model.Viper.GetStringMap("databases") {
		dbViper := subViper.Sub(key)
		model.Databases[key] = SubConfig{
			Name:  key,
			Type:  dbViper.GetString("type"),
			Viper: dbViper,
		}
	}
}

func loadStoragesConfig(model *ModelConfig) {
	// Backward compatible with `store_with` config
	storeWith := model.Viper.Sub("store_with")
	model.Storages = map[string]SubConfig{}
	if storeWith != nil {
		logger.Warn(`[Deprecated] "store_with" is deprecated now, please use "storages" which supports multiple storages.`)
		model.Storages["store_with"] = SubConfig{
			Name:  "",
			Type:  model.Viper.GetString("store_with.type"),
			Viper: model.Viper.Sub("store_with"),
		}
	}

	subViper := model.Viper.Sub("storages")
	model.Storages = map[string]SubConfig{}
	for key := range model.Viper.GetStringMap("storages") {
		storageViper := subViper.Sub(key)
		model.Storages[key] = SubConfig{
			Name:  key,
			Type:  storageViper.GetString("type"),
			Viper: storageViper,
		}
	}
}

// GetModelByName get model by name
func GetModelByName(name string) (model *ModelConfig) {
	for _, m := range Models {
		if m.Name == name {
			model = &m
			return
		}
	}
	return
}

// GetDatabaseByName get database config by name
func (model *ModelConfig) GetDatabaseByName(name string) (subConfig *SubConfig) {
	for _, m := range model.Databases {
		if m.Name == name {
			subConfig = &m
			return
		}
	}
	return
}
