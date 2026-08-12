package mintegral

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// BidGoal 是优化目标类型。
type BidGoal string

// 当前支持的优化目标类型。
const (
	BidGoalTargetCPE  BidGoal = "Target-CPE"
	BidGoalTargetROAS BidGoal = "Target-ROAS"
)

// Known 报告优化目标类型是否为当前 SDK 已知值。
func (value BidGoal) Known() bool { return value == BidGoalTargetCPE || value == BidGoalTargetROAS }

// BidGoalSupportsRequest 是可优化事件查询条件。
type BidGoalSupportsRequest struct {
	// CampaignID 按广告活动筛选；与 PackageName 至少提供一个。
	CampaignID CampaignID
	// PackageName 按应用包名筛选；与 CampaignID 至少提供一个。
	PackageName string
	// BidGoal 指定要查询的优化目标类型。
	BidGoal BidGoal
}

// BidGoalSupportsResponse 是可优化事件查询结果。
type BidGoalSupportsResponse struct {
	// SupportEvents 是可用于该优化目标的事件列表。
	SupportEvents []SupportedBidGoalEvent `json:"support_events"`
}

// SupportedBidGoalEvent 是一个支持指定优化目标的事件。
type SupportedBidGoalEvent struct {
	// OriginalEvent 是 Target-CPE 对应的原始事件名。
	OriginalEvent string `json:"original_event"`
	// MTGEvent 是 Mintegral 标准事件名。
	MTGEvent string `json:"mtg_event"`
	// TargetGoalWindow 是该事件支持的目标时间窗。
	TargetGoalWindow []string `json:"target_goal_window"`
}

// EventService 提供事件元数据接口。
type EventService struct{ client *Client }

// Events 返回与 Client 共享传输和凭据配置的事件服务。
func (c *Client) Events() *EventService { return &EventService{client: c} }

// BidGoalSupports 查询指定广告活动或包名支持的优化事件。
func (s EventService) BidGoalSupports(ctx context.Context, request BidGoalSupportsRequest, options ...RequestOption) (BidGoalSupportsResponse, error) {
	if request.CampaignID <= 0 && strings.TrimSpace(request.PackageName) == "" {
		return BidGoalSupportsResponse{}, fmt.Errorf("%w: campaign ID or package name is required", ErrInvalidRequest)
	}
	if request.BidGoal != BidGoalTargetCPE && request.BidGoal != BidGoalTargetROAS {
		return BidGoalSupportsResponse{}, fmt.Errorf("%w: unsupported bid goal", ErrInvalidRequest)
	}
	return doJSON[BidGoalSupportsResponse](ctx, s.client, requestSpec{
		operation: "event.bid_goal_supports", method: http.MethodGet,
		path: "/api/open/v3/event/bid_goal_supports", authenticated: true, retryable: true,
		query: func() (url.Values, error) {
			query := make(url.Values)
			query.Set("bid_goal", string(request.BidGoal))
			if request.CampaignID > 0 {
				query.Set("campaign_id", strconv.FormatInt(int64(request.CampaignID), 10))
			}
			if request.PackageName != "" {
				query.Set("package_name", request.PackageName)
			}
			return query, nil
		},
	}, options)
}
