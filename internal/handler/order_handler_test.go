package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Luclpor/loyalty_system.git/internal/dto"
	"github.com/Luclpor/loyalty_system.git/internal/handler/apiModel"
	"github.com/Luclpor/loyalty_system.git/internal/service/order"
	orderdto "github.com/Luclpor/loyalty_system.git/internal/service/order/dto"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type stubAuthService struct {
	user *dto.UserDto
	err  error
}

func (s *stubAuthService) GetUserFromContext(context.Context) (*dto.UserDto, error) {
	return s.user, s.err
}

type stubOrderManager struct {
	saveNewOrderFn func(ctx context.Context, orderApi *apiModel.OrderApiModel) error
	getUserOrders  func(ctx context.Context, userID uuid.UUID) ([]orderdto.OrderDto, error)
}

func (s *stubOrderManager) SaveNewOrder(ctx context.Context, orderApi *apiModel.OrderApiModel) error {
	if s.saveNewOrderFn == nil {
		return nil
	}
	return s.saveNewOrderFn(ctx, orderApi)
}

func (s *stubOrderManager) GetUserOrders(ctx context.Context, userID uuid.UUID) ([]orderdto.OrderDto, error) {
	if s.getUserOrders == nil {
		return nil, nil
	}
	return s.getUserOrders(ctx, userID)
}

type errReadCloser struct {
	err error
}

func (e errReadCloser) Read(_ []byte) (int, error) {
	return 0, e.err
}

func (e errReadCloser) Close() error {
	return nil
}

func TestCreateNewOrder(t *testing.T) {
	t.Parallel()

	appLogger := zap.NewNop()
	userID := uuid.New()

	tests := []struct {
		name         string
		auth         *stubAuthService
		body         io.ReadCloser
		manager      *stubOrderManager
		expectedCode int
		assert       func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "returns internal server error when user is missing in context",
			auth: &stubAuthService{err: errors.New("missing user")},
			body: io.NopCloser(bytes.NewBufferString("79927398713")),
			manager: &stubOrderManager{
				saveNewOrderFn: func(context.Context, *apiModel.OrderApiModel) error {
					t.Fatal("order manager should not be called when auth fails")
					return nil
				},
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name: "returns bad request when body cannot be read",
			auth: &stubAuthService{user: &dto.UserDto{ID: userID}},
			body: errReadCloser{err: errors.New("read failed")},
			manager: &stubOrderManager{
				saveNewOrderFn: func(context.Context, *apiModel.OrderApiModel) error {
					t.Fatal("order manager should not be called when body read fails")
					return nil
				},
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "returns accepted when order is saved",
			auth: &stubAuthService{user: &dto.UserDto{ID: userID}},
			body: io.NopCloser(bytes.NewBufferString("79927398713")),
			manager: &stubOrderManager{
				saveNewOrderFn: func(_ context.Context, orderApi *apiModel.OrderApiModel) error {
					if orderApi.UserID != userID {
						t.Fatalf("unexpected user id: got %s want %s", orderApi.UserID, userID)
					}
					if orderApi.OrderNum != "79927398713" {
						t.Fatalf("unexpected order number: got %q", orderApi.OrderNum)
					}
					return nil
				},
			},
			expectedCode: http.StatusAccepted,
		},
		{
			name: "returns conflict when order belongs to another user",
			auth: &stubAuthService{user: &dto.UserDto{ID: userID}},
			body: io.NopCloser(bytes.NewBufferString("79927398713")),
			manager: &stubOrderManager{
				saveNewOrderFn: func(context.Context, *apiModel.OrderApiModel) error {
					return order.ErrorOrderAlreadyUploadSomeUser
				},
			},
			expectedCode: http.StatusConflict,
		},
		{
			name: "returns ok when order already belongs to current user",
			auth: &stubAuthService{user: &dto.UserDto{ID: userID}},
			body: io.NopCloser(bytes.NewBufferString("79927398713")),
			manager: &stubOrderManager{
				saveNewOrderFn: func(context.Context, *apiModel.OrderApiModel) error {
					return order.ErrorOrderAlreadyUploadThisUser
				},
			},
			expectedCode: http.StatusOK,
		},
		{
			name: "returns unprocessable entity when order number is invalid",
			auth: &stubAuthService{user: &dto.UserDto{ID: userID}},
			body: io.NopCloser(bytes.NewBufferString("123")),
			manager: &stubOrderManager{
				saveNewOrderFn: func(context.Context, *apiModel.OrderApiModel) error {
					return order.ErrorInvalidOrderNum
				},
			},
			expectedCode: http.StatusUnprocessableEntity,
		},
		{
			name: "returns internal server error on unexpected save failure",
			auth: &stubAuthService{user: &dto.UserDto{ID: userID}},
			body: io.NopCloser(bytes.NewBufferString("79927398713")),
			manager: &stubOrderManager{
				saveNewOrderFn: func(context.Context, *apiModel.OrderApiModel) error {
					return errors.New("db error")
				},
			},
			expectedCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodPost, "/api/user/orders", tt.body)
			rr := httptest.NewRecorder()

			CreateNewOrder(tt.auth, tt.manager, appLogger).ServeHTTP(rr, req)

			if rr.Code != tt.expectedCode {
				t.Fatalf("unexpected status code: got %d want %d", rr.Code, tt.expectedCode)
			}
			if tt.assert != nil {
				tt.assert(t, rr)
			}
		})
	}
}

func TestGetUserOrdersHandler(t *testing.T) {
	t.Parallel()

	appLogger := zap.NewNop()
	userID := uuid.New()
	now := time.Date(2026, 5, 16, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name         string
		auth         *stubAuthService
		manager      *stubOrderManager
		expectedCode int
		assert       func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "returns internal server error when user is missing in context",
			auth: &stubAuthService{err: errors.New("missing user")},
			manager: &stubOrderManager{
				getUserOrders: func(context.Context, uuid.UUID) ([]orderdto.OrderDto, error) {
					t.Fatal("order manager should not be called when auth fails")
					return nil, nil
				},
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name: "returns no content on domain no content error",
			auth: &stubAuthService{user: &dto.UserDto{ID: userID}},
			manager: &stubOrderManager{
				getUserOrders: func(context.Context, uuid.UUID) ([]orderdto.OrderDto, error) {
					return nil, order.ErrorInvalidOrderNum
				},
			},
			expectedCode: http.StatusNoContent,
		},
		{
			name: "returns internal server error on unexpected failure",
			auth: &stubAuthService{user: &dto.UserDto{ID: userID}},
			manager: &stubOrderManager{
				getUserOrders: func(context.Context, uuid.UUID) ([]orderdto.OrderDto, error) {
					return nil, errors.New("storage failure")
				},
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name: "returns user orders as json",
			auth: &stubAuthService{user: &dto.UserDto{ID: userID}},
			manager: &stubOrderManager{
				getUserOrders: func(_ context.Context, gotUserID uuid.UUID) ([]orderdto.OrderDto, error) {
					if gotUserID != userID {
						t.Fatalf("unexpected user id: got %s want %s", gotUserID, userID)
					}
					return []orderdto.OrderDto{
						{
							OrderNumber: "79927398713",
							Status:      "PROCESSED",
							Point:       123.45,
							UploadedAt:  now,
						},
					}, nil
				},
			},
			expectedCode: http.StatusOK,
			assert: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var got []apiModel.ReadOrderApiModel
				if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}
				if len(got) != 1 {
					t.Fatalf("unexpected response length: got %d want 1", len(got))
				}
				if got[0].OrderNum != "79927398713" {
					t.Fatalf("unexpected order number: got %q", got[0].OrderNum)
				}
				if got[0].Status != "PROCESSED" {
					t.Fatalf("unexpected status: got %q", got[0].Status)
				}
				if got[0].Accrual != 123.45 {
					t.Fatalf("unexpected accrual: got %v", got[0].Accrual)
				}
				if !got[0].UploadedAt.Equal(now) {
					t.Fatalf("unexpected uploaded at: got %s want %s", got[0].UploadedAt, now)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
			rr := httptest.NewRecorder()

			GetUserOrdersHandler(tt.auth, tt.manager, appLogger).ServeHTTP(rr, req)

			if rr.Code != tt.expectedCode {
				t.Fatalf("unexpected status code: got %d want %d", rr.Code, tt.expectedCode)
			}
			if tt.assert != nil {
				tt.assert(t, rr)
			}
		})
	}
}
