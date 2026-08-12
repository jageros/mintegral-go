package mintegral

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// CreativeSetService 提供创意组操作。
type CreativeSetService struct{ client *Client }

// CreativeSets 返回绑定当前客户端的创意组服务。
func (c *Client) CreativeSets() *CreativeSetService { return &CreativeSetService{client: c} }

// List 查询创意组。
func (s *CreativeSetService) List(ctx context.Context, request CreativeSetListRequest, options ...RequestOption) (CreativeSetList, error) {
	if request.Page == 0 {
		request.Page = 1
	}
	if request.Limit == 0 {
		request.Limit = 10
	}
	if request.Page < 1 || request.Limit < 1 || request.Limit > 50 || (request.CombinationMethod != 0 && !request.CombinationMethod.Known()) {
		return CreativeSetList{}, fmt.Errorf("%w: invalid creative set pagination", ErrInvalidRequest)
	}
	spec := requestSpec{operation: "creative_set.list", method: http.MethodGet, path: "/api/open/v1/creative_sets", authenticated: true, retryable: true, query: func() (url.Values, error) {
		values := make(url.Values)
		values.Set("page", strconv.Itoa(request.Page))
		values.Set("limit", strconv.Itoa(request.Limit))
		if request.OfferID != 0 {
			values.Set("offer_id", strconv.FormatInt(int64(request.OfferID), 10))
		}
		if request.CreativeSetID != 0 {
			values.Set("creative_set_id", strconv.FormatInt(int64(request.CreativeSetID), 10))
		}
		if request.CreativeSetName != "" {
			values.Set("creative_set_name", request.CreativeSetName)
		}
		if request.CombinationMethod != 0 {
			values.Set("combination_method", strconv.Itoa(int(request.CombinationMethod)))
		}
		return values, nil
	}}
	return doJSON[CreativeSetList](ctx, s.client, spec, options)
}

// Create 创建创意组。
func (s *CreativeSetService) Create(ctx context.Context, request CreateCreativeSetRequest, options ...RequestOption) (CreativeSetMutationResult, error) { //nolint:gocritic // 公共 API 约定使用值请求，避免调用方共享可变请求。
	if strings.TrimSpace(request.CreativeSetName) == "" || len(request.AdOutputs) == 0 || len(request.Creatives) == 0 || (request.CombinationMethod != 0 && !request.CombinationMethod.Known()) || !validAdOutputs(request.AdOutputs) || !validCreativeSetInputs(request.Creatives) {
		return CreativeSetMutationResult{}, fmt.Errorf("%w: invalid creative set create request", ErrInvalidRequest)
	}
	return doJSON[CreativeSetMutationResult](ctx, s.client, creativeSetWriteSpec(http.MethodPost, "creative_set.create", request), options)
}

// Update 更新创意组。
func (s *CreativeSetService) Update(ctx context.Context, request UpdateCreativeSetRequest, options ...RequestOption) error {
	if request.OfferID < 1 || strings.TrimSpace(request.CreativeSetName) == "" || !validOptionalAdOutputs(request.AdOutputs) || !validCreativeSetEdits(request.Creatives) {
		return fmt.Errorf("%w: invalid creative set update request", ErrInvalidRequest)
	}
	_, err := doJSON[struct{}](ctx, s.client, creativeSetWriteSpec(http.MethodPut, "creative_set.update", request), options)
	return err
}

func validAdOutputs(outputs []AdOutput) bool {
	for _, output := range outputs {
		if !output.Known() {
			return false
		}
	}
	return true
}

func validOptionalAdOutputs(outputs *[]AdOutput) bool {
	return outputs == nil || validAdOutputs(*outputs)
}

func validCreativeSetInputs(inputs []CreativeSetInput) bool {
	for _, input := range inputs {
		if strings.TrimSpace(input.CreativeName) == "" {
			return false
		}
		if _, err := ParseContentMD5(input.CreativeMD5.String()); err != nil {
			return false
		}
	}
	return true
}

func validCreativeSetEdits(edits *[]CreativeSetEdit) bool {
	if edits == nil {
		return true
	}
	for _, edit := range *edits {
		if strings.TrimSpace(edit.CreativeName) == "" || !edit.Option.valid() {
			return false
		}
		if _, err := ParseContentMD5(edit.CreativeMD5.String()); err != nil {
			return false
		}
	}
	return true
}

// Delete 删除创意组。
func (s *CreativeSetService) Delete(ctx context.Context, request DeleteCreativeSetRequest, options ...RequestOption) error {
	if request.OfferID < 1 || strings.TrimSpace(request.CreativeSetName) == "" {
		return fmt.Errorf("%w: invalid creative set delete request", ErrInvalidRequest)
	}
	_, err := doJSON[struct{}](ctx, s.client, creativeSetWriteSpec(http.MethodDelete, "creative_set.delete", request), options)
	return err
}

func creativeSetWriteSpec[T CreateCreativeSetRequest | UpdateCreativeSetRequest | DeleteCreativeSetRequest](method, operation string, request T) requestSpec {
	return requestSpec{operation: operation, method: method, path: "/api/open/v1/creative_set", body: jsonBody(request), contentType: "application/json", authenticated: true, outcomeRisk: true}
}
