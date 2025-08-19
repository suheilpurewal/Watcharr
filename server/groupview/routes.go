package groupview

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(db *gorm.DB, api *gin.RouterGroup) {
	a := &API{DB: db}
	g := api.Group("/group")
	g.GET("/members", a.GetMembers)
	g.POST("/members", a.PostMember)
	g.POST("/viewings", a.PostViewing)
	g.GET("/history", a.GetHistory)
}
