package handler

import (
	"github.com/gin-gonic/gin"
)

func ResponseJSONP(c *gin.Context, res Resp) {
	c.IndentedJSON(int(res.Code), res)
}
