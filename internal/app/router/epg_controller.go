package router

import (
	"context"
	"encoding/xml"
	"errors"
	"iptv/internal/app/iptv"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var (
	// 缓存最新的节目单数据
	epgPtr atomic.Pointer[[]iptv.ChannelProgramList]
)

// ChannelDateJsonEPG 频道的JSON格式EPG
type ChannelDateJsonEPG struct {
	ChannelName string    `json:"channel_name"`
	Date        string    `json:"date"`
	EPGData     []JsonEPG `json:"epg_data"`
}

// JsonEPG JSON格式EPG
type JsonEPG struct {
	Title string `json:"title"` // 标题
	Desc  string `json:"desc"`  // 描述
	Start string `json:"start"` // 开始时间
	End   string `json:"end"`   // 结束时间
}

// GetJsonEPG 获取JSON格式的EPG
func GetJsonEPG(c *gin.Context) {
	// 获取频道名称
	chName := c.Query("ch")
	// 获取日期
	dateStr := c.DefaultQuery("date", time.Now().Format("2006-01-02"))

	// 校验频道名称是否为空
	if chName == "" {
		logger.Warn("The name of the channel is null.")
		// 返回响应
		c.Status(http.StatusBadRequest)
		return
	}

	// 解析日期
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		logger.Error("Date format error", zap.Error(err))
		c.Status(http.StatusBadRequest)
		return
	}

	// 空响应
	emptyResp := ChannelDateJsonEPG{
		ChannelName: chName,
		Date:        dateStr,
		EPGData:     []JsonEPG{},
	}

	// 如果缓存的节目单列表为空则直接返回空数据
	chProgLists := *epgPtr.Load()
	if len(chProgLists) == 0 {
		c.JSON(http.StatusOK, &emptyResp)
		return
	}

	// 根据频道名称查询到该频道所有日期的节目单列表
	var tagerChProgList *iptv.ChannelProgramList
	for _, chProgList := range chProgLists {
		if chProgList.ChannelName == chName {
			tagerChProgList = &chProgList
			break
		}
	}
	if tagerChProgList == nil || len(tagerChProgList.DateProgramList) == 0 {
		c.JSON(http.StatusOK, &emptyResp)
		return
	}

	// 查询该频道指定日期的节目单列表
	dateEPGData := make([]JsonEPG, 0)
	for _, dateProgList := range tagerChProgList.DateProgramList {
		if dateProgList.Date.Equal(date) {
			if len(dateProgList.ProgramList) > 0 {
				for _, program := range dateProgList.ProgramList {
					dateEPGData = append(dateEPGData, JsonEPG{
						Title: program.ProgramName,
						Start: program.StartTime,
						End:   program.EndTime,
					})
				}
			}
			break
		}
	}

	// 返回最终响应
	c.JSON(http.StatusOK, &ChannelDateJsonEPG{
		ChannelName: chName,
		Date:        dateStr,
		EPGData:     dateEPGData,
	})
}

// XmlEPG XMLTV格式的EPG
type XmlEPG struct {
	XMLName           xml.Name          `xml:"tv"`
	SourceInfoUrl     string            `xml:"source-info-url,attr,omitempty"`
	SourceInfoName    string            `xml:"source-info-name,attr,omitempty"`
	SourceDataUrl     string            `xml:"source-data-url,attr,omitempty"`
	GeneratorInfoName string            `xml:"generator-info-name,attr,omitempty"`
	GeneratorInfoUrl  string            `xml:"generator-info-url,attr,omitempty"`
	Channels          []XmlEPGChannel   `xml:"channel,omitempty"`
	Programmes        []XmlEPGProgramme `xml:"programme,omitempty"`
}

type XmlEPGChannel struct {
	Id          string         `xml:"id,attr"`
	DisplayName *XmlEPGDisplay `xml:"display-name"`
}

type XmlEPGProgramme struct {
	Start   string         `xml:"start,attr"`
	Stop    string         `xml:"stop,attr"`
	Channel string         `xml:"channel,attr"`
	Title   *XmlEPGDisplay `xml:"title"`
	Desc    *XmlEPGDisplay `xml:"desc,omitempty"`
}

type XmlEPGDisplay struct {
	Lang  string `xml:"lang,attr"`
	Value string `xml:",chardata"`
}

// GetXmlEPG 返回XMLTV格式的EPG
func GetXmlEPG(c *gin.Context) {
	// 如果缓存的节目单列表为空则直接返回空数据
	chProgLists := *epgPtr.Load()
	if len(chProgLists) == 0 {
		c.XML(http.StatusOK, &XmlEPG{})
		return
	}

	channels := make([]XmlEPGChannel, 0, len(chProgLists))
	programmes := make([]XmlEPGProgramme, 0)
	for _, chProgList := range chProgLists {
		// 获取频道的相关信息
		channels = append(channels, XmlEPGChannel{
			Id: chProgList.ChannelId,
			DisplayName: &XmlEPGDisplay{
				Lang:  "zh",
				Value: chProgList.ChannelName,
			},
		})

		if len(chProgList.DateProgramList) == 0 {
			continue
		}

		for _, dateProgList := range chProgList.DateProgramList {
			if len(dateProgList.ProgramList) == 0 {
				continue
			}
			for _, program := range dateProgList.ProgramList {
				// 获取节目的相关信息
				programmes = append(programmes, XmlEPGProgramme{
					Start:   program.BeginTimeFormat + " +0800",
					Stop:    program.EndTimeFormat + " +0800",
					Channel: chProgList.ChannelId,
					Title: &XmlEPGDisplay{
						Lang:  "zh",
						Value: program.ProgramName,
					},
				})
			}
		}
	}

	c.XML(http.StatusOK, &XmlEPG{
		Channels:   channels,
		Programmes: programmes,
	})
}

// updateEPG 更新缓存的节目单数据
func updateEPG(ctx context.Context, iptvClient *iptv.Client) error {
	channels := *channelsPtr.Load()
	if len(channels) == 0 {
		return errors.New("no channels")
	}

	// 登录认证获取Token等信息
	token, err := iptvClient.GenerateToken(ctx)
	if err != nil {
		return err
	}

	epg := make([]iptv.ChannelProgramList, 0, len(channels))
	for _, channel := range channels {
		// 跳过不支持回看的频道
		if channel.TimeShift != "1" || channel.TimeShiftLength <= 0 {
			continue
		}

		progList, err := iptvClient.GetChannelProgramList(ctx, token, channel.ChannelID)
		if err != nil {
			logger.Sugar().Warnf("Failed to get the program list for channel %s. Error: %v", channel.ChannelName, err)
			continue
		}
		// 将频道名称设置上，方便后续查询
		progList.ChannelName = channel.ChannelName
		epg = append(epg, *progList)
	}

	logger.Sugar().Infof("EPG data updated, rows: %d.", len(epg))
	// 更新缓存的频道列表
	epgPtr.Store(&epg)

	return nil
}
