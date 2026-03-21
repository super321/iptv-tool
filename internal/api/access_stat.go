package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"iptv-tool-v2/internal/service"
	"iptv-tool-v2/pkg/i18n"
)

// AccessStatController handles access statistics API
type AccessStatController struct {
	accessStatSvc *service.AccessStatService
}

func NewAccessStatController(accessStatSvc *service.AccessStatService) *AccessStatController {
	return &AccessStatController{accessStatSvc: accessStatSvc}
}

// GetAccessStats returns paginated access statistics from the last 7 days
// GET /api/settings/access-stats?page=1&page_size=20
func (asc *AccessStatController) GetAccessStats(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	lang := i18n.Lang(c)
	localLabel := i18n.T(lang, "label.local_network")
	items, total := asc.accessStatSvc.Query(page, pageSize, lang, localLabel)

	c.JSON(http.StatusOK, gin.H{
		"total": total,
		"items": items,
	})
}
