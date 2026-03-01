package publish

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"iptv-tool-v2/internal/model"
)

// Handler serves published subscription endpoints
// GET /sub/live/:path and GET /sub/epg/:path
func LiveHandler(c *gin.Context) {
	path := c.Param("path")
	if path == "" {
		c.String(http.StatusNotFound, "not found")
		return
	}

	// Look up the publish interface by path and type
	var iface model.PublishInterface
	if err := model.DB.Where("path = ? AND type = ? AND status = ?", path, "live", true).First(&iface).Error; err != nil {
		c.String(http.StatusNotFound, "subscription not found")
		return
	}

	engine, err := NewEngine(iface)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to initialize publish engine: %s", err.Error())
		return
	}

	requestHost := c.Request.Host
	if fwd := c.GetHeader("X-Forwarded-Host"); fwd != "" {
		requestHost = fwd
	}

	serveLive(c, engine, iface, requestHost)
}

// EPGHandler serves EPG subscription endpoints
// GET /sub/epg/:path
func EPGHandler(c *gin.Context) {
	path := c.Param("path")
	if path == "" {
		c.String(http.StatusNotFound, "not found")
		return
	}

	var iface model.PublishInterface
	if err := model.DB.Where("path = ? AND type = ? AND status = ?", path, "epg", true).First(&iface).Error; err != nil {
		c.String(http.StatusNotFound, "subscription not found")
		return
	}

	engine, err := NewEngine(iface)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to initialize publish engine: %s", err.Error())
		return
	}

	requestHost := c.Request.Host
	if fwd := c.GetHeader("X-Forwarded-Host"); fwd != "" {
		requestHost = fwd
	}

	serveEPG(c, engine, iface, requestHost)
}

func serveLive(c *gin.Context, engine *Engine, iface model.PublishInterface, requestHost string) {
	// [修改点1]：在这里传入 requestHost
	channels, err := engine.AggregateLiveChannels(requestHost)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to aggregate channels: %s", err.Error())
		return
	}
	switch iface.Format {
	case model.PublishFormatM3U:
		c.Header("Content-Type", "text/plain; charset=utf-8")
		// [修改点2]：在这里去掉 requestHost
		c.String(http.StatusOK, engine.FormatM3U(channels))
	case model.PublishFormatTXT:
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.String(http.StatusOK, engine.FormatTXT(channels))
	default:
		c.String(http.StatusBadRequest, "unsupported format for live type")
	}
}

func serveEPG(c *gin.Context, engine *Engine, iface model.PublishInterface, requestHost string) {
	programs, err := engine.AggregateEPGPrograms()
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to aggregate EPG: %s", err.Error())
		return
	}

	switch iface.Format {
	case model.PublishFormatXMLTV:
		// Only use gzip when explicitly enabled in the interface settings
		if iface.GzipEnabled {
			if err := engine.FormatXMLTVGzip(programs, c.Writer); err != nil {
				c.String(http.StatusInternalServerError, "gzip encoding failed")
			}
		} else {
			c.Header("Content-Type", "application/xml; charset=utf-8")
			c.String(http.StatusOK, engine.FormatXMLTV(programs))
		}
	case model.PublishFormatDIYP:
		// DIYP JSON format supports query params: ?ch=频道名&date=2024-01-15
		channelName := c.Query("ch")
		dateStr := c.Query("date")
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.String(http.StatusOK, engine.FormatDIYP(programs, channelName, dateStr))
	default:
		c.String(http.StatusBadRequest, "unsupported format for epg type")
	}
}
