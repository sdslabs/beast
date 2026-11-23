package system

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdslabs/beastv4/api/httputil"
)

// OpenAPISpecHandler handles GET /openapi.json
func OpenAPISpecHandler(c *gin.Context) {
	// TODO: implement dynamic OpenAPI specification generation
	httputil.RespondJSON(c, http.StatusOK, gin.H{})
}

// DocsUIHandler handles GET /docs
func DocsUIHandler(c *gin.Context) {
	// TODO: implement Swagger / OpenAPI interactive documentation UI
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte("<!-- Swagger UI Placeholder -->"))
}
