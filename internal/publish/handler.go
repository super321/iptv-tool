package publish

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"iptv-tool-v2/internal/model"
	"iptv-tool-v2/pkg/i18n"
)

// Handler serves published subscription endpoints
// GET /sub/live/:path
func LiveHandler(c *gin.Context) {
	path := c.Param("path")
	if path == "" {
		c.String(http.StatusNotFound, i18n.T(i18n.Lang(c), "publish_handler.not_found"))
		return
	}

	// Look up the publish interface by path and type
	var iface model.PublishInterface
	if err := model.DB.Where("path = ? AND type = ? AND status = ?", path, "live", true).First(&iface).Error; err != nil {
		c.String(http.StatusNotFound, i18n.T(i18n.Lang(c), "publish_handler.sub_not_found"))
		return
	}

	engine, err := NewEngine(iface)
	if err != nil {
		c.String(http.StatusInternalServerError, "%s: %s", i18n.T(i18n.Lang(c), "publish_handler.engine_init_failed"), err.Error())
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
		c.String(http.StatusNotFound, i18n.T(i18n.Lang(c), "publish_handler.not_found"))
		return
	}

	var iface model.PublishInterface
	if err := model.DB.Where("path = ? AND type = ? AND status = ?", path, "epg", true).First(&iface).Error; err != nil {
		c.String(http.StatusNotFound, i18n.T(i18n.Lang(c), "publish_handler.sub_not_found"))
		return
	}

	engine, err := NewEngine(iface)
	if err != nil {
		c.String(http.StatusInternalServerError, "%s: %s", i18n.T(i18n.Lang(c), "publish_handler.engine_init_failed"), err.Error())
		return
	}

	requestHost := c.Request.Host
	if fwd := c.GetHeader("X-Forwarded-Host"); fwd != "" {
		requestHost = fwd
	}

	serveEPG(c, engine, iface, requestHost)
}

func serveLive(c *gin.Context, engine *Engine, iface model.PublishInterface, requestHost string) {
	channels, err := LoadOrStoreLiveChannels(iface.ID, engine.AggregateLiveChannels)
	if err != nil {
		c.String(http.StatusInternalServerError, "%s: %s", i18n.T(i18n.Lang(c), "publish_handler.channel_agg_failed"), err.Error())
		return
	}

	switch iface.Format {
	case model.PublishFormatM3U:
		c.Header("Content-Type", "text/plain; charset=utf-8")
		// Pass requestHost for logo URL resolution in M3U format
		c.String(http.StatusOK, engine.FormatM3U(channels, requestHost))
	case model.PublishFormatTXT:
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.String(http.StatusOK, engine.FormatTXT(channels))
	default:
		c.String(http.StatusBadRequest, i18n.T(i18n.Lang(c), "publish_handler.live_format_unsupported"))
	}
}

func serveEPG(c *gin.Context, engine *Engine, iface model.PublishInterface, _ string) {
	epg, err := LoadOrStoreEPGPrograms(iface.ID, engine.AggregateEPG)
	if err != nil {
		c.String(http.StatusInternalServerError, "%s: %s", i18n.T(i18n.Lang(c), "publish_handler.epg_agg_failed"), err.Error())
		return
	}

	switch iface.Format {
	case model.PublishFormatXMLTV:
		// Only use gzip when explicitly enabled in the interface settings
		if iface.GzipEnabled {
			if err := engine.FormatXMLTVGzip(epg, c.Writer); err != nil {
				c.String(http.StatusInternalServerError, i18n.T(i18n.Lang(c), "publish_handler.gzip_failed"))
			}
		} else {
			c.Header("Content-Type", "application/xml; charset=utf-8")
			c.String(http.StatusOK, engine.FormatXMLTV(epg))
		}
	case model.PublishFormatDIYP:
		// DIYP JSON format supports query params: ?ch=频道名&date=2024-01-15
		channelName := c.Query("ch")
		dateStr := c.Query("date")
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.String(http.StatusOK, engine.FormatDIYP(epg, channelName, dateStr))
	default:
		c.String(http.StatusBadRequest, i18n.T(i18n.Lang(c), "publish_handler.epg_format_unsupported"))
	}
}
