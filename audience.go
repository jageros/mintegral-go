package mintegral

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const audiencePath = "/api/open/v1/audience"

// AudienceService 提供人群包查询、上传和变更操作。
type AudienceService struct {
	client *Client
}

// Audiences 返回绑定当前客户端的人群包服务。
func (c *Client) Audiences() *AudienceService {
	return &AudienceService{client: c}
}

// List 查询人群包列表。
func (s *AudienceService) List(ctx context.Context, request AudienceListRequest, options ...RequestOption) (AudienceList, error) {
	if request.Limit == 0 {
		request.Limit = 10
	}
	if request.Page == 0 {
		request.Page = 1
	}
	if request.Limit < 1 || request.Limit > 500 || request.Page < 1 || request.Platform < 0 || request.Platform > 3 {
		return AudienceList{}, fmt.Errorf("%w: invalid audience list pagination or platform", ErrInvalidRequest)
	}
	spec := requestSpec{
		operation: "audience.list", method: http.MethodGet, path: audiencePath,
		authenticated: true, retryable: true,
		query: func() (url.Values, error) {
			values := make(url.Values)
			values.Set("limit", strconv.Itoa(request.Limit))
			values.Set("page", strconv.Itoa(request.Page))
			if len(request.TAIDs) > 0 {
				ids := make([]string, len(request.TAIDs))
				for index, id := range request.TAIDs {
					ids[index] = strconv.FormatInt(int64(id), 10)
				}
				values.Set("ta_ids", strings.Join(ids, ","))
			}
			if request.TAName != "" {
				values.Set("ta_name", request.TAName)
			}
			if request.Platform != 0 {
				values.Set("platform", strconv.Itoa(request.Platform))
			}
			return values, nil
		},
	}
	return doJSON[AudienceList](ctx, s.client, spec, options)
}

// Create 创建人群包。
func (s *AudienceService) Create(ctx context.Context, request CreateAudienceRequest, options ...RequestOption) (AudienceMutationResult, error) {
	if err := validateAudiencePaths(request.DataPath); err != nil || strings.TrimSpace(request.TAName) == "" || request.AreaType < 1 || request.AreaType > 2 || request.Platform < 1 || request.Platform > 3 {
		return AudienceMutationResult{}, fmt.Errorf("%w: invalid audience create request", ErrInvalidRequest)
	}
	return s.mutate(ctx, http.MethodPost, "audience.create", request, options)
}

// Update 替换人群包文件；服务端要求创建或上次修改十二小时后才能再次更新。
func (s *AudienceService) Update(ctx context.Context, request UpdateAudienceRequest, options ...RequestOption) (AudienceMutationResult, error) {
	if request.TAID < 1 || validateAudiencePaths(request.DataPath) != nil {
		return AudienceMutationResult{}, fmt.Errorf("%w: invalid audience update request", ErrInvalidRequest)
	}
	return s.mutate(ctx, http.MethodPut, "audience.update", request, options)
}

// Delete 批量删除人群包。
func (s *AudienceService) Delete(ctx context.Context, request DeleteAudienceRequest, options ...RequestOption) error {
	if len(request.TAIDs) == 0 {
		return fmt.Errorf("%w: audience IDs are required", ErrInvalidRequest)
	}
	for _, id := range request.TAIDs {
		if id < 1 {
			return fmt.Errorf("%w: audience ID must be positive", ErrInvalidRequest)
		}
	}
	_, err := doJSON[struct{}](ctx, s.client, requestSpec{operation: "audience.delete", method: http.MethodDelete, path: audiencePath, body: jsonBody(request), contentType: "application/json", authenticated: true, outcomeRisk: true, allowEmptyData: true}, options)
	return err
}

func (s *AudienceService) mutate(ctx context.Context, method, operation string, request any, options []RequestOption) (AudienceMutationResult, error) {
	return doJSON[AudienceMutationResult](ctx, s.client, requestSpec{operation: operation, method: method, path: audiencePath, body: jsonBody(request), contentType: "application/json", authenticated: true, outcomeRisk: true}, options)
}

func validateAudiencePaths(paths []AudienceDataPath) error {
	if len(paths) == 0 {
		return ErrInvalidRequest
	}
	for _, path := range paths {
		switch path.DeviceType {
		case AudienceDeviceIMEI, AudienceDeviceIDFA, AudienceDeviceGAID, AudienceDeviceOAID,
			AudienceDeviceIMEIMD5, AudienceDeviceIDFAMD5, AudienceDeviceGAIDMD5,
			AudienceDeviceOAIDMD5, AudienceDeviceIDFV, AudienceDeviceIDFVMD5:
		default:
			return ErrInvalidRequest
		}
		if strings.TrimSpace(path.DataPath) == "" {
			return ErrInvalidRequest
		}
	}
	return nil
}
