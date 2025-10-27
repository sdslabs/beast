package plugins

import (
	"github.com/sdslabs/beastv4/core/config"
	log "github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
)

type Plugin interface {
	Name() string
	Description() string
	Init(*gin.Engine) error
}

//All plugins will be initialized with the gin engine, at startup
//Logging twice, once when we register and once when we initialize

var loadedPlugins []Plugin

func Register(p Plugin) {
	loadedPlugins = append(loadedPlugins, p)
	log.Infof("Plugin registered: %s\n %s", p.Name(), p.Description())

}

func isPluginEnabled(pluginName string) bool {
	enabledPlugins := config.Cfg.PluginsEnabled.EnabledPlugins
	for _, name := range enabledPlugins {
		if name == pluginName {
			return true
		}
	}
	return false
}

func InitPlugins(router *gin.Engine) {
	for _, p := range loadedPlugins {
		if !isPluginEnabled(p.Name()) {
			log.Warnf("%s is not enabled, skipping initialization", p.Name())
			continue
		}
		log.Infof("Intializing plugin: %s", p.Name())
		err := p.Init(router)
		if err != nil {
			log.Errorf("Error in Plugin Initializing %s: %v", p.Name(), err)
		}
	}
}
