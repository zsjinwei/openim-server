package unipush

import (
	"context"

	"github.com/openimsdk/open-im-server/v3/internal/push/offlinepush/options"
	"github.com/openimsdk/open-im-server/v3/pkg/common/config"
	"github.com/openimsdk/tools/utils/httputil"
	"github.com/openimsdk/tools/utils/jsonutil"
)

type UniPush struct {
	pushConf   *config.Push
	httpClient *httputil.HTTPClient
}

func NewClient(pushConf *config.Push) *UniPush {
	return &UniPush{
		pushConf:   pushConf,
		httpClient: httputil.NewHTTPClient(httputil.NewClientConfig()),
	}
}
func (u *UniPush) Push(ctx context.Context, userIDs []string, title, content string, opts *options.Opts) error {
	req := newPushReq(u.pushConf, title, content)
	var prefixedUserIDs []string
	for _, id := range userIDs {
		prefixedUserIDs = append(prefixedUserIDs, "openim."+id)
	}
	req.Data.PushClientID = prefixedUserIDs
	if opts.Ex != "" {
		ex := options.PushEx{}
		err := jsonutil.JsonStringToStruct(opts.Ex, &ex)
		if err == nil {
			if ex.Badge != "" {
				req.Data.Badge = ex.Badge
			}
			if ex.Sound != "" {
				req.Data.Sound = ex.Sound
			}
			req.Data.Payload = ex.Payload
		}
	}
	resp := &UniPushResp{}
	err := u.postRequest(ctx, req, resp)
	if err != nil {
		return err
	}
	return nil
}
func newPushReq(pushConf *config.Push, title, content string) UniPushReq {
	pushReq := UniPushReq{
		AppID: pushConf.UniPush.AppId,
		Data: UniPushData{
			Title:   title,
			Content: content,
		},
	}
	return pushReq
}
func (u *UniPush) postRequest(ctx context.Context, input any, output any) error {
	headerData := make(map[string]interface{})
	headerData["appId"] = input.(UniPushReq).AppID
	header, err := u.getCommonHeaders("POST", nil, headerData)
	if err != nil {
		return err
	}
	resp := &UniPushResp{}
	resp.Data = output
	return u.postReturn(ctx, u.pushConf.UniPush.PushUrl, header, input, resp, 30)
}
func (u *UniPush) postReturn(
	ctx context.Context,
	url string,
	header map[string]string,
	input any,
	output UniPushRespI,
	timeout int,
) error {
	err := u.httpClient.PostReturn(ctx, url, header, input, output, timeout)
	if err != nil {
		return err
	}
	return output.parseError()
}
