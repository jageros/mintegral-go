package mintegral

import (
	"context"
	"net/http"
)

// AccountService 提供 Mintegral 账户接口。
type AccountService struct{ client *Client }

// AccountBalance 是账户余额列表。
type AccountBalance struct {
	// Total 是返回的账户数量。
	Total int `json:"total"`
	// List 是各账户的余额明细。
	List []AccountBalanceItem `json:"list"`
}

// AccountBalanceItem 是一个账户的结算币种与余额。
type AccountBalanceItem struct {
	// UserID 是账户唯一标识。
	UserID UserID `json:"user_id"`
	// Username 是账户名称。
	Username string `json:"username"`
	// Currency 是结算币种代码。
	Currency string `json:"currency"`
	// Balance 是账户余额的精确十进制文本。
	Balance DecimalText `json:"balance"`
}

// Balance 获取当前凭据可访问账户的余额。
func (s *AccountService) Balance(ctx context.Context, options ...RequestOption) (AccountBalance, error) {
	return doJSON[AccountBalance](ctx, s.client, requestSpec{
		operation: "account.balance", method: http.MethodGet, path: "/api/open/v1/account/balance",
		authenticated: true, retryable: true,
	}, options)
}
