package mintegral

import (
	"context"
	"errors"
	"fmt"
	"io"
)

// Consume 同步分批读取；只有处理器返回 nil 的批次才计入确认行数。
func (s *ReportService) Consume(ctx context.Context, request ReportConsumeRequest, handler ReportHandler, options ...RequestOption) (delivery ReportDelivery, err error) { //nolint:gocritic // 公开 API 按值接收请求，避免调用方共享可变状态。
	if handler == nil {
		return delivery, fmt.Errorf("%w: report handler is nil", ErrInvalidRequest)
	}
	if request.BatchSize < 0 {
		return delivery, fmt.Errorf("%w: batch size cannot be negative", ErrInvalidRequest)
	}
	if request.BatchSize == 0 {
		request.BatchSize = 500
	}
	stream, err := s.Open(ctx, request.Open, options...)
	if err != nil {
		return delivery, err
	}
	if stream == nil {
		return delivery, fmt.Errorf("%w: report open returned a nil stream", ErrUnexpectedResponse)
	}
	delivery.Status = stream.status
	rows := make([]ReportRow, 0, request.BatchSize)
	for {
		row, nextErr := stream.Next()
		if nextErr == nil {
			rows = append(rows, row)
			delivery.ParsedRows++
		}
		if len(rows) == request.BatchSize || (errors.Is(nextErr, io.EOF) && len(rows) > 0) {
			if handleErr := handler(ctx, rows); handleErr != nil {
				return finishReportDelivery(delivery, stream, handleErr)
			}
			delivery.AcknowledgedRows += int64(len(rows))
			rows = rows[:0]
		}
		if errors.Is(nextErr, io.EOF) {
			return finishReportDelivery(delivery, stream, nil)
		}
		if nextErr != nil {
			return finishReportDelivery(delivery, stream, nextErr)
		}
	}
}

func finishReportDelivery(delivery ReportDelivery, stream *ReportStream, cause error) (ReportDelivery, error) {
	return delivery, partialReportError(delivery, errors.Join(cause, stream.Close()))
}

func partialReportError(delivery ReportDelivery, cause error) error {
	if cause == nil {
		return nil
	}
	if delivery.AcknowledgedRows == 0 {
		return cause
	}
	return &partialDeliveryError{cause: cause}
}
