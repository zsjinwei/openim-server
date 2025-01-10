package unipush

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

// EPushPlatform defines the enumeration for different push platforms.
type EPushPlatform string

const (
	ALL                     EPushPlatform = "ALL"
	APP_IOS                 EPushPlatform = "app-ios"                 // iOS App
	APP_ANDROID             EPushPlatform = "app-android"             // Android App
	WEB                     EPushPlatform = "web"                     // 网页
	MP_WEIXIN               EPushPlatform = "mp-weixin"               // 微信小程序
	MP_ALIPAY               EPushPlatform = "mp-alipay"               // 支付宝小程序
	MP_BAIDU                EPushPlatform = "mp-baidu"                // 百度小程序
	MP_TOUTIAO              EPushPlatform = "mp-toutiao"              // 抖音小程序
	MP_LARK                 EPushPlatform = "mp-lark"                 // 飞书小程序
	MP_QQ                   EPushPlatform = "mp-qq"                   // QQ小程序
	MP_KUAISHOU             EPushPlatform = "mp-kuaishou"             // 快手小程序
	MP_JD                   EPushPlatform = "mp-jd"                   // 京东小程序
	MP_360                  EPushPlatform = "mp-360"                  // 360小程序
	QUICKAPP_WEBVIEW        EPushPlatform = "quickapp-webview"        // 快应用通用
	QUICKAPP_WEBVIEW_UNION  EPushPlatform = "quickapp-webview-union"  // 快应用联盟
	QUICKAPP_WEBVIEW_HUAWEI EPushPlatform = "quickapp-webview-huawei" // 快应用华为
)

// IGetuiBigDataTag represents the tag structure for Getui big data.
type IGetuiBigDataTag struct {
	Key     string   `json:"key"`      // 查询条件
	Values  []string `json:"values"`   // 查询条件值列表
	OptType string   `json:"opt_type"` // or(或),and(与),not(非)
}

// IPushSettings represents settings for push notifications.
type IPushSettings struct {
	TTL          *int   `json:"ttl,omitempty"`           // 消息离线时间设置
	Strategy     *any   `json:"strategy,omitempty"`      // 厂商通道策略
	Speed        *int   `json:"speed,omitempty"`         // 定速推送
	ScheduleTime *int64 `json:"schedule_time,omitempty"` // 定时推送时间
}

// IChannelSettings represents channel-specific settings for push notifications.
type IChannelSettings struct {
	HW *string `json:"hw,omitempty"` // 华为消息分类
	XM *string `json:"xm,omitempty"` // 小米推送消息分类
	OP *string `json:"op,omitempty"` // OPush平台消息分类
	VV *int    `json:"vv,omitempty"` // Vivo消息分类
}

// UniPushData represents the payload for sending push notifications.
type UniPushData struct {
	UserID            []string           `json:"user_id,omitempty"`            // 指定接收消息的用户id
	UserTag           []string           `json:"user_tag,omitempty"`           // 指定接收消息的用户标签
	DeviceID          []string           `json:"device_id,omitempty"`          // 指定接收消息的设备id
	PushClientID      []string           `json:"push_clientid,omitempty"`      // 指定接收消息的设备推送标识
	GetuiCustomTag    string             `json:"getui_custom_tag,omitempty"`   // 基于个推getui_custom_tag
	GetuiBigDataTag   []IGetuiBigDataTag `json:"getui_big_data_tag,omitempty"` // 符合筛选条件的设备
	GetuiAlias        []string           `json:"getui_alias,omitempty"`        // 个推自定义客户端别名
	Platform          []EPushPlatform    `json:"platform,omitempty"`           // 指定接收消息的平台
	CheckToken        bool               `json:"check_token,omitempty"`        // 校验客户端登陆状态
	Title             string             `json:"title"`                        // 通知栏标题
	Content           string             `json:"content"`                      // 通知栏内容
	Payload           any                `json:"payload"`                      // 推送透传数据
	ForceNotification bool               `json:"force_notification,omitempty"` // 是否创建通知栏消息
	Badge             string             `json:"badge,omitempty"`              // 应用右上角数字
	Channel           IChannelSettings   `json:"channel,omitempty"`            // 消息渠道设置
	RequestID         string             `json:"request_id,omitempty"`         // 请求唯一标识号
	GroupName         string             `json:"group_name,omitempty"`         // 任务组名
	Sound             string             `json:"sound,omitempty"`              // 消息提醒铃声设置
	ContentAvailable  int                `json:"content_available,omitempty"`  // 静默推送标识
	OpenURL           string             `json:"open_url,omitempty"`           // 打开链接
	Settings          IPushSettings      `json:"settings,omitempty"`           // 推送条件设置
	Options           any                `json:"options,omitempty"`            // 其他配置
}
type UniPushReq struct {
	AppID string      `json:"appId,omitempty"` // 应用ID
	Data  UniPushData `json:"data"`            // Push Data
}
type UniPushResp struct {
	ErrCode   string `json:"errCode"`
	ErrMsg    string `json:"errMsg"`
	ErrDetail string `json:"errDetail"`
	Data      any    `json:"data"`
}

func (r *UniPushResp) parseError() (err error) {
	switch r.ErrCode {
	case "200":
		err = nil
	case "0":
		err = nil
	default:
		err = fmt.Errorf("code %s, msg %s", r.ErrCode, r.ErrMsg)
	}
	return err
}

type UniPushRespI interface {
	parseError() error
}

func getSignBody(obj map[string]interface{}, filterBase bool) string {
	if obj == nil {
		return ""
	}
	signBody := make(map[string]string)
	signKeys := []string{}
	// Populate signBody and signKeys
	for key, value := range obj {
		if !filterBase || isBasicType(value) {
			signBody[key] = fmt.Sprintf("%v", value)
			signKeys = append(signKeys, key)
		}
	}
	// Sort the keys
	sort.Strings(signKeys)
	signArr := []string{}
	for _, key := range signKeys {
		signArr = append(signArr, fmt.Sprintf("%s=%s", key, signBody[key]))
	}
	return join(signArr, "&")
}

// Helper function to check if the value is a basic type (string, number, or boolean)
func isBasicType(value interface{}) bool {
	switch value.(type) {
	case string, int, int32, int64, float32, float64, bool:
		return true
	default:
		return false
	}
}

// Join function to mimic JavaScript's array join method
func join(arr []string, sep string) string {
	if len(arr) == 0 {
		return ""
	}
	result := arr[0]
	for _, s := range arr[1:] {
		result += sep + s
	}
	return result
}
func (u *UniPush) getCommonHeaders(method string, params map[string]interface{}, data map[string]interface{}) (map[string]string, error) {
	headers := make(map[string]string)
	if u.pushConf.UniPush.SecType == "connectCode" {
		headers["Unicloud-S2s-Authorization"] = fmt.Sprintf("CONNECTCODE %s", u.pushConf.UniPush.ConnectCode)
	} else if u.pushConf.UniPush.SecType == "sign" {
		timestamp := time.Now().UnixNano() / int64(time.Millisecond)
		headers["Unicloud-S2s-Timestamp"] = fmt.Sprintf("%d", timestamp)
		signature, err := u.getSignature(timestamp, method, params, data)
		if err != nil {
			return nil, err
		}
		headers["Unicloud-S2s-Signature"] = signature
	}
	return headers, nil
}
func (u *UniPush) getSignature(timestamp int64, method string, params map[string]interface{}, data map[string]interface{}) (string, error) {
	var payloadStr string
	switch method = strings.ToUpper(method); method {
	case "GET":
		payloadStr = getSignBody(params, false)
	case "POST":
		payloadStr = getSignBody(data, true)
	default:
		return "", nil
	}
	var s string
	switch u.pushConf.UniPush.SignMethod {
	case "md5":
		h := md5.New()
		h.Write([]byte(fmt.Sprintf("%d\n%s\n%s", timestamp, payloadStr, u.pushConf.UniPush.SignKey)))
		s = hex.EncodeToString(h.Sum(nil))
	case "sha1":
		h := sha1.New()
		h.Write([]byte(fmt.Sprintf("%d\n%s\n%s", timestamp, payloadStr, u.pushConf.UniPush.SignKey)))
		s = hex.EncodeToString(h.Sum(nil))
	case "sha256":
		h := sha256.New()
		h.Write([]byte(fmt.Sprintf("%d\n%s\n%s", timestamp, payloadStr, u.pushConf.UniPush.SignKey)))
		s = hex.EncodeToString(h.Sum(nil))
	case "hmac-sha256":
		h := hmac.New(sha256.New, []byte(u.pushConf.UniPush.SignKey))
		h.Write([]byte(fmt.Sprintf("%d\n%s", timestamp, payloadStr)))
		s = hex.EncodeToString(h.Sum(nil))
	default:
		return "", fmt.Errorf("unsupported sign method: %s", u.pushConf.UniPush.SignMethod)
	}
	return fmt.Sprintf("%s %s", u.pushConf.UniPush.SignMethod, s), nil
}
