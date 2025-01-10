// Copyright © 2023 OpenIM. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package msg

import (
	"context"
	"encoding/json"
	"errors"

	cbapi "github.com/openimsdk/open-im-server/v3/pkg/callbackstruct"
	"github.com/openimsdk/open-im-server/v3/pkg/common/storage/database/mgo"
	"github.com/openimsdk/protocol/constant"
	"github.com/openimsdk/protocol/msg"
	"github.com/openimsdk/protocol/sdkws"
	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/log"
	"github.com/openimsdk/tools/utils/datautil"
	"github.com/redis/go-redis/v9"
)

func filter(seq []int64, val int64) []int64 {
	var result []int64
	for _, v := range seq {
		if v != val {
			result = append(result, v)
		}
	}
	return result
}

func (m *msgServer) GetConversationsHasReadAndMaxSeq(ctx context.Context, req *msg.GetConversationsHasReadAndMaxSeqReq) (*msg.GetConversationsHasReadAndMaxSeqResp, error) {
	var conversationIDs []string
	if len(req.ConversationIDs) == 0 {
		var err error
		conversationIDs, err = m.ConversationLocalCache.GetConversationIDs(ctx, req.UserID)
		if err != nil {
			return nil, err
		}
	} else {
		conversationIDs = req.ConversationIDs
	}

	hasReadSeqs, err := m.MsgDatabase.GetHasReadSeqs(ctx, req.UserID, conversationIDs)
	if err != nil {
		return nil, err
	}

	conversations, err := m.ConversationLocalCache.GetConversations(ctx, req.UserID, conversationIDs)
	if err != nil {
		return nil, err
	}

	conversationMaxSeqMap := make(map[string]int64)
	for _, conversation := range conversations {
		if conversation.MaxSeq != 0 {
			conversationMaxSeqMap[conversation.ConversationID] = conversation.MaxSeq
		}
	}
	maxSeqs, err := m.MsgDatabase.GetMaxSeqsWithTime(ctx, conversationIDs)
	if err != nil {
		return nil, err
	}
	resp := &msg.GetConversationsHasReadAndMaxSeqResp{Seqs: make(map[string]*msg.Seqs)}
	for conversationID, maxSeq := range maxSeqs {
		resp.Seqs[conversationID] = &msg.Seqs{
			HasReadSeq: hasReadSeqs[conversationID],
			MaxSeq:     maxSeq.Seq,
			MaxSeqTime: maxSeq.Time,
		}
		if v, ok := conversationMaxSeqMap[conversationID]; ok {
			resp.Seqs[conversationID].MaxSeq = v
		}
	}
	return resp, nil
}

func (m *msgServer) SetConversationHasReadSeq(ctx context.Context, req *msg.SetConversationHasReadSeqReq) (*msg.SetConversationHasReadSeqResp, error) {
	maxSeq, err := m.MsgDatabase.GetMaxSeq(ctx, req.ConversationID)
	if err != nil {
		return nil, err
	}
	if req.HasReadSeq > maxSeq {
		return nil, errs.ErrArgs.WrapMsg("hasReadSeq must not be bigger than maxSeq")
	}
	if err := m.MsgDatabase.SetHasReadSeq(ctx, req.UserID, req.ConversationID, req.HasReadSeq); err != nil {
		return nil, err
	}
	m.sendMarkAsReadNotification(ctx, req.ConversationID, constant.SingleChatType, req.UserID, req.UserID, nil, req.HasReadSeq, "")
	return &msg.SetConversationHasReadSeqResp{}, nil
}

func (m *msgServer) MarkMsgsAsRead(ctx context.Context, req *msg.MarkMsgsAsReadReq) (*msg.MarkMsgsAsReadResp, error) {
	if len(req.Seqs) < 1 {
		return nil, errs.ErrArgs.WrapMsg("seqs must not be empty")
	}
	maxSeq, err := m.MsgDatabase.GetMaxSeq(ctx, req.ConversationID)
	if err != nil {
		return nil, err
	}

	// -1 means maxSeq, replace -1 with maxSeq
	for i := range req.Seqs {
		if req.Seqs[i] == -1 {
			req.Seqs[i] = maxSeq
		}
	}
	req.Seqs = append(filter(req.Seqs, maxSeq), maxSeq)

	hasReadSeq := req.Seqs[len(req.Seqs)-1]
	if hasReadSeq > maxSeq {
		return nil, errs.ErrArgs.WrapMsg("hasReadSeq must not be bigger than maxSeq")
	}
	conversation, err := m.ConversationLocalCache.GetConversation(ctx, req.UserID, req.ConversationID)
	if err != nil {
		return nil, err
	}
	if err := m.MsgDatabase.MarkSingleChatMsgsAsRead(ctx, req.UserID, req.ConversationID, req.Seqs); err != nil {
		return nil, err
	}
	currentHasReadSeq, err := m.MsgDatabase.GetHasReadSeq(ctx, req.UserID, req.ConversationID)
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}
	if hasReadSeq > currentHasReadSeq {
		err = m.MsgDatabase.SetHasReadSeq(ctx, req.UserID, req.ConversationID, hasReadSeq)
		if err != nil {
			return nil, err
		}
	}

	reqCallback := &cbapi.CallbackSingleMsgReadReq{
		ConversationID: conversation.ConversationID,
		UserID:         req.UserID,
		Seqs:           req.Seqs,
		ContentType:    conversation.ConversationType,
	}
	m.webhookAfterSingleMsgRead(ctx, &m.config.WebhooksConfig.AfterSingleMsgRead, reqCallback)
	m.sendMarkAsReadNotification(ctx, req.ConversationID, conversation.ConversationType, req.UserID,
		m.conversationAndGetRecvID(conversation, req.UserID), req.Seqs, hasReadSeq, "")
	return &msg.MarkMsgsAsReadResp{}, nil
}

func (m *msgServer) MarkConversationAsRead(ctx context.Context, req *msg.MarkConversationAsReadReq) (*msg.MarkConversationAsReadResp, error) {
	conversation, err := m.ConversationLocalCache.GetConversation(ctx, req.UserID, req.ConversationID)
	if err != nil {
		return nil, err
	}
	hasReadSeq, err := m.MsgDatabase.GetHasReadSeq(ctx, req.UserID, req.ConversationID)
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}
	var seqs []int64

	log.ZDebug(ctx, "MarkConversationAsRead", "hasReadSeq", hasReadSeq, "req.HasReadSeq", req.HasReadSeq)
	if conversation.ConversationType == constant.SingleChatType {
		for i := hasReadSeq + 1; i <= req.HasReadSeq; i++ {
			seqs = append(seqs, i)
		}
		// avoid client missed call MarkConversationMessageAsRead by order
		for _, val := range req.Seqs {
			if !datautil.Contain(val, seqs...) {
				seqs = append(seqs, val)
			}
		}
		if len(seqs) > 0 {
			log.ZDebug(ctx, "MarkConversationAsRead", "seqs", seqs, "conversationID", req.ConversationID)
			if err = m.MsgDatabase.MarkSingleChatMsgsAsRead(ctx, req.UserID, req.ConversationID, seqs); err != nil {
				return nil, err
			}
		}
		if req.HasReadSeq > hasReadSeq {
			err = m.MsgDatabase.SetHasReadSeq(ctx, req.UserID, req.ConversationID, req.HasReadSeq)
			if err != nil {
				return nil, err
			}
			hasReadSeq = req.HasReadSeq
		}
		m.sendMarkAsReadNotification(ctx, req.ConversationID, conversation.ConversationType, req.UserID,
			m.conversationAndGetRecvID(conversation, req.UserID), seqs, hasReadSeq, "")
	} else if conversation.ConversationType == constant.ReadGroupChatType ||
		conversation.ConversationType == constant.NotificationChatType {
		notificationEx := ""
		if conversation.ConversationType == constant.ReadGroupChatType {
			for i := hasReadSeq + 1; i <= req.HasReadSeq; i++ {
				seqs = append(seqs, i)
			}
			// avoid client missed call MarkConversationMessageAsRead by order
			for _, val := range req.Seqs {
				if !datautil.Contain(val, seqs...) {
					seqs = append(seqs, val)
				}
			}
			if len(seqs) > 0 {
				log.ZDebug(ctx, "MarkConversationAsRead", "seqs", seqs, "conversationID", req.ConversationID)
				if notificationEx, err = m.MsgDatabase.MarkGroupChatMsgsAsRead(ctx, req.UserID, req.ConversationID, seqs); err != nil {
					return nil, err
				}
			}
		}
		if req.HasReadSeq > hasReadSeq {
			err = m.MsgDatabase.SetHasReadSeq(ctx, req.UserID, req.ConversationID, req.HasReadSeq)
			if err != nil {
				return nil, err
			}
			hasReadSeq = req.HasReadSeq
		}
		m.sendMarkAsReadNotification(ctx, req.ConversationID, conversation.ConversationType, req.UserID,
			m.conversationAndGetRecvID(conversation, req.UserID), seqs, hasReadSeq, notificationEx)
	}

	if conversation.ConversationType == constant.SingleChatType {
		reqCall := &cbapi.CallbackSingleMsgReadReq{
			ConversationID: conversation.ConversationID,
			UserID:         conversation.OwnerUserID,
			Seqs:           req.Seqs,
			ContentType:    conversation.ConversationType,
		}
		m.webhookAfterSingleMsgRead(ctx, &m.config.WebhooksConfig.AfterSingleMsgRead, reqCall)
	} else if conversation.ConversationType == constant.ReadGroupChatType {
		reqCall := &cbapi.CallbackGroupMsgReadReq{
			SendID:       conversation.OwnerUserID,
			ReceiveID:    req.UserID,
			UnreadMsgNum: req.HasReadSeq,
			ContentType:  int64(conversation.ConversationType),
		}
		m.webhookAfterGroupMsgRead(ctx, &m.config.WebhooksConfig.AfterGroupMsgRead, reqCall)
	}
	return &msg.MarkConversationAsReadResp{}, nil
}

type MarkAsReadTipsEx struct {
	*sdkws.MarkAsReadTips
	Ex string `json:"ex,omitempty"`
}

func (m *msgServer) sendMarkAsReadNotification(ctx context.Context, conversationID string, sessionType int32, sendID, recvID string, seqs []int64, hasReadSeq int64, ex string) {
	tips := &sdkws.MarkAsReadTips{
		MarkAsReadUserID: sendID,
		ConversationID:   conversationID,
		Seqs:             seqs,
		HasReadSeq:       hasReadSeq,
	}
	tipsEx := &MarkAsReadTipsEx{
		MarkAsReadTips: tips,
		Ex:             ex,
	}
	m.notificationSender.NotificationWithSessionType(ctx, sendID, recvID, constant.HasReadReceipt, sessionType, tipsEx)
}

const (
	GroupMessageTypeRead   = int32(0)
	GroupMessageTypeUnread = int32(1)
)

func (m *msgServer) GetGroupMessageHasRead(ctx context.Context, req *msg.GetGroupMessageHasReadReq) (*msg.GetGroupMessageHasReadResp, error) {
	msgInfo, err := m.MsgDatabase.GetMsgInfoByClientMsgID(ctx, req.ConversationID, req.ClientMsgID)
	if err != nil {
		return nil, err
	}

	if msgInfo == nil {
		return nil, errs.ErrArgs.WrapMsg("cannot find msgInfo by clientMsgID")
	}

	msgData := msgInfo.Msg
	if msgData == nil {
		return nil, errs.ErrArgs.WrapMsg("cannot find msgData by clientMsgID")
	}

	attachedInfoStr := msgData.AttachedInfo
	var attachedInfo mgo.AttachedInfoElem
	err = json.Unmarshal([]byte(attachedInfoStr), &attachedInfo)
	if err != nil {
		return nil, err
	}
	groupMemberCount := attachedInfo.GroupHasReadInfo.GroupMemberCount
	readCount := attachedInfo.GroupHasReadInfo.HasReadCount
	unreadCount := groupMemberCount - readCount - 1
	if unreadCount < 0 {
		unreadCount = 0
	}
	readResp := &msg.GetGroupMessageHasReadResp{}
	skipNumber := int32(0)
	limitNumber := int32(0)
	if req.Pagination != nil {
		limitNumber = req.Pagination.ShowNumber
		if req.Pagination.PageNumber > 0 {
			skipNumber = (req.Pagination.PageNumber - 1) * limitNumber
			if skipNumber < 0 {
				skipNumber = 0
			}
		}
	}
	reqIndex := int32(-1)
	if req.Type == GroupMessageTypeRead {
		// 已读
		readResp.Count = int64(readCount)
		readInfoMap := msgInfo.ReadInfo
		if len(readInfoMap) > 0 {
			for _, readInfo := range readInfoMap {
				readInfo := &msg.GroupMessageReadInfo{
					UserID:   readInfo.UserID,
					ReadTime: readInfo.ReadTime,
				}
				reqIndex += 1
				if limitNumber > 0 && skipNumber >= 0 {
					if reqIndex < skipNumber || reqIndex >= skipNumber+limitNumber {
						continue
					}
				}
				readResp.Reads = append(readResp.Reads, readInfo)
			}
		} else if len(attachedInfo.GroupHasReadInfo.HasReadUserIDList) > 0 {
			for _, userID := range attachedInfo.GroupHasReadInfo.HasReadUserIDList {
				readInfo := &msg.GroupMessageReadInfo{
					UserID:   userID,
					ReadTime: 0,
				}
				reqIndex += 1
				if limitNumber > 0 && skipNumber >= 0 {
					if reqIndex < skipNumber || reqIndex >= skipNumber+limitNumber {
						continue
					}
				}
				readResp.Reads = append(readResp.Reads, readInfo)
			}
		}
		return readResp, nil
	} else if req.Type == GroupMessageTypeUnread {
		// 未读
		readResp.Count = int64(unreadCount)
		memberIDs, err := m.GroupLocalCache.GetGroupMemberIDs(ctx, msgData.GroupID)
		if err != nil {
			return nil, err
		}
		for _, memberID := range memberIDs {
			readInfoMap := msgInfo.ReadInfo
			if len(readInfoMap) > 0 {
				_, exists := readInfoMap[memberID]
				if exists {
					continue
				}
			} else if len(attachedInfo.GroupHasReadInfo.HasReadUserIDList) > 0 {
				if datautil.IndexOf(memberID, attachedInfo.GroupHasReadInfo.HasReadUserIDList...) >= 0 {
					continue
				}
			}
			readInfo := &msg.GroupMessageReadInfo{
				UserID:   memberID,
				ReadTime: 0,
			}
			reqIndex += 1
			if limitNumber > 0 && skipNumber >= 0 {
				if reqIndex < skipNumber || reqIndex >= skipNumber+limitNumber {
					continue
				}
			}
			readResp.Reads = append(readResp.Reads, readInfo)
		}
		return readResp, nil
	} else {
		return nil, errs.ErrArgs.WrapMsg("params type is invalid")
	}
}
