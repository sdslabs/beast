package dummy

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/plugins"
)

type DummyPlugin struct{}

func (p *DummyPlugin) Name() string {
	return "DummyPlugin"
}

func (p *DummyPlugin) Description() string {
	return "A dummy plugin to do the testing of plugin system"
}

func (p *DummyPlugin) Init(router *gin.Engine) error {
	router.GET("api/plugins/dummy", func(context *gin.Context) {
		context.JSON(http.StatusOK, gin.H{"plugin": "The dummy plugin works"})
	})
	return nil
}

func init() {
	plugins.Register(&DummyPlugin{})
}
