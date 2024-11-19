package push

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/openimsdk/open-im-server/v3/internal/push/offlinepush"
	"github.com/openimsdk/open-im-server/v3/internal/push/offlinepush/options"
	"github.com/openimsdk/open-im-server/v3/pkg/apistruct"
	"github.com/openimsdk/open-im-server/v3/pkg/common/prommetrics"
	"github.com/openimsdk/open-im-server/v3/pkg/rpccache"
	"github.com/openimsdk/open-im-server/v3/pkg/rpcclient"
	"github.com/openimsdk/protocol/constant"
	pbpush "github.com/openimsdk/protocol/push"
	"github.com/openimsdk/protocol/sdkws"
	"github.com/openimsdk/tools/discovery"
	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/log"
	"github.com/openimsdk/tools/mq/kafka"
	"github.com/openimsdk/tools/utils/jsonutil"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"
)

type OfflinePushConsumerHandler struct {
	OfflinePushConsumerGroup *kafka.MConsumerGroup
	offlinePusher            offlinepush.OfflinePusher
	groupRpcClient           rpcclient.GroupRpcClient
	groupLocalCache          *rpccache.GroupLocalCache
}

func NewOfflinePushConsumerHandler(config *Config, offlinePusher offlinepush.OfflinePusher, rdb redis.UniversalClient,
	client discovery.SvcDiscoveryRegistry) (*OfflinePushConsumerHandler, error) {
	var offlinePushConsumerHandler OfflinePushConsumerHandler
	var err error
	offlinePushConsumerHandler.offlinePusher = offlinePusher
	offlinePushConsumerHandler.OfflinePushConsumerGroup, err = kafka.NewMConsumerGroup(config.KafkaConfig.Build(), config.KafkaConfig.ToOfflineGroupID,
		[]string{config.KafkaConfig.ToOfflinePushTopic}, true)
	offlinePushConsumerHandler.groupRpcClient = rpcclient.NewGroupRpcClient(client, config.Share.RpcRegisterName.Group)
	offlinePushConsumerHandler.groupLocalCache = rpccache.NewGroupLocalCache(offlinePushConsumerHandler.groupRpcClient, &config.LocalCacheConfig, rdb)

	if err != nil {
		return nil, err
	}
	return &offlinePushConsumerHandler, nil
}

func (*OfflinePushConsumerHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (*OfflinePushConsumerHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }
func (o *OfflinePushConsumerHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		ctx := o.OfflinePushConsumerGroup.GetContextFromMsg(msg)
		o.handleMsg2OfflinePush(ctx, msg.Value)
		sess.MarkMessage(msg, "")
	}
	return nil
}

func (o *OfflinePushConsumerHandler) handleMsg2OfflinePush(ctx context.Context, msg []byte) {
	offlinePushMsg := pbpush.PushMsgReq{}
	if err := proto.Unmarshal(msg, &offlinePushMsg); err != nil {
		log.ZError(ctx, "offline push Unmarshal msg err", err, "msg", string(msg))
		return
	}
	if offlinePushMsg.MsgData == nil || offlinePushMsg.UserIDs == nil {
		log.ZError(ctx, "offline push msg is empty", errs.New("offlinePushMsg is empty"), "userIDs", offlinePushMsg.UserIDs, "msg", offlinePushMsg.MsgData)
		return
	}
	if offlinePushMsg.MsgData.Status == constant.MsgStatusSending {
		offlinePushMsg.MsgData.Status = constant.MsgStatusSendSuccess
	}
	log.ZInfo(ctx, "receive to OfflinePush MQ", "userIDs", offlinePushMsg.UserIDs, "msg", offlinePushMsg.MsgData)

	err := o.offlinePushMsg(ctx, offlinePushMsg.MsgData, offlinePushMsg.UserIDs)
	if err != nil {
		log.ZWarn(ctx, "offline push failed", err, "msg", offlinePushMsg.String())
	}
}

func (o *OfflinePushConsumerHandler) getOfflinePushInfos(ctx context.Context, msg *sdkws.MsgData) (title, content string, opts *options.Opts, err error) {
	type AtTextElem struct {
		Text       string   `json:"text,omitempty"`
		AtUserList []string `json:"atUserList,omitempty"`
		IsAtSelf   bool     `json:"isAtSelf"`
	}

	opts = &options.Opts{Signal: &options.Signal{ClientMsgID: msg.ClientMsgID}}
	if msg.OfflinePushInfo != nil {
		opts.IOSBadgeCount = msg.OfflinePushInfo.IOSBadgeCount
		opts.IOSPushSound = msg.OfflinePushInfo.IOSPushSound
		ex := options.PushEx{}
		_ = jsonutil.JsonStringToStruct(msg.OfflinePushInfo.Ex, &ex)
		ex.Payload.SessionType = int(msg.SessionType)

		if msg.SessionType == constant.SingleChatType {
			ex.Payload.SourceID = msg.SendID
		} else if msg.SessionType == constant.WriteGroupChatType || msg.SessionType == constant.ReadGroupChatType {
			ex.Payload.SourceID = msg.GroupID
		}

		opts.Ex = jsonutil.StructToJsonString(ex)
	}

	if msg.OfflinePushInfo != nil {
		title = msg.OfflinePushInfo.Title
		content = msg.OfflinePushInfo.Desc
	}
	if title == "" {
		ignoreSenderNickname := true
		if msg.SessionType == constant.SingleChatType {
			title = msg.SenderNickname
			ignoreSenderNickname = true
		} else if msg.SessionType == constant.WriteGroupChatType || msg.SessionType == constant.ReadGroupChatType {
			gm, err := o.groupLocalCache.GetGroupInfo(ctx, msg.GroupID)
			if err == nil {
				title = gm.GroupName
			}
			ignoreSenderNickname = false
		}
		switch msg.ContentType {
		case constant.Text:
			c := apistruct.TextElem{}
			if err := jsonutil.JsonStringToStruct(string(msg.Content), &c); err != nil {
				content = "[文本]"
			} else {
				content = c.Content
			}
		case constant.Picture:
			content = "[图片]"
		case constant.Voice:
			c := apistruct.SoundElem{}
			if err := jsonutil.JsonStringToStruct(string(msg.Content), &c); err != nil {
				content = "[语音]"
			} else {
				content = fmt.Sprintf("[语音] %d\"", c.Duration)
			}
		case constant.Video:
			content = "[视频]"
		case constant.File:
			c := apistruct.FileElem{}
			if err := jsonutil.JsonStringToStruct(string(msg.Content), &c); err != nil {
				content = "[文件]"
			} else {
				content = fmt.Sprintf("[文件] %s", c.FileName)
			}
		case constant.AtText:
			c := apistruct.AtElem{}
			if err := jsonutil.JsonStringToStruct(string(msg.Content), &c); err != nil {
				content = "[文本]"
			} else {
				if c.IsAtSelf {
					content = fmt.Sprintf("[有人@我] %s", c.Text)
				} else {
					content = c.Text
				}
			}
		default:
			content = "您有一条新消息"
		}
		if !ignoreSenderNickname {
			content = fmt.Sprintf("%s: %s", msg.SenderNickname, content)
		}
	}
	if content == "" {
		content = title
	}
	return
}

func (o *OfflinePushConsumerHandler) offlinePushMsg(ctx context.Context, msg *sdkws.MsgData, offlinePushUserIDs []string) error {
	title, content, opts, err := o.getOfflinePushInfos(ctx, msg)
	if err != nil {
		return err
	}
	err = o.offlinePusher.Push(ctx, offlinePushUserIDs, title, content, opts)
	if err != nil {
		prommetrics.MsgOfflinePushFailedCounter.Inc()
		return err
	}
	return nil
}
