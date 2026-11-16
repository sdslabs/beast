package system

import (
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// OpenAPISpecHandler handles GET /openapi.json
func OpenAPISpecHandler(c *gin.Context) {
	// TODO: implement dynamic OpenAPI specification generation
	httputil.RespondNotImplemented(c, "OpenAPISpecHandler")
}

// DocsUIHandler handles GET /docs
func DocsUIHandler(c *gin.Context) {
	// TODO: implement Swagger / OpenAPI interactive documentation UI
	httputil.RespondNotImplemented(c, "DocsUIHandler")
}
