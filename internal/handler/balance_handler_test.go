package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Luclpor/loyalty_system.git/internal/dto"
	"github.com/Luclpor/loyalty_system.git/internal/handler/apiModel"
	"github.com/Luclpor/loyalty_system.git/internal/service/order"
	userBalance "github.com/Luclpor/loyalty_system.git/internal/service/userBalance"
	"github.com/Luclpor/loyalty_system.git/internal/service/userBalance/withdraw"
	"github.com/Luclpor/loyalty_system.git/internal/storage/postgres"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type stubBalanceManager struct {
	withdrawUserBalanceFn func(ctx context.Context, api *apiModel.BalanceApi, user *dto.UserDto, appLogger *zap.Logger) (*withdraw.WithdrawDto, error)
	getUserWithdrawsFn    func(ctx context.Context, userID uuid.UUID) ([]apiModel.ReadWithdrawApi, error)
	getUserBalanceFn      func(ctx context.Context, userID uuid.UUID) (*apiModel.ReadBalanceApi, error)
}

func (s *stubBalanceManager) WithdrawUserBalance(ctx context.Context, api *apiModel.BalanceApi, user *dto.UserDto, appLogger *zap.Logger) (*withdraw.WithdrawDto, error) {
	if s.withdrawUserBalanceFn == nil {
		return nil, nil
	}
	return s.withdrawUserBalanceFn(ctx, api, user, appLogger)
}

func (s *stubBalanceManager) GetUserWithDraws(ctx context.Context, userID uuid.UUID) ([]apiModel.ReadWithdrawApi, error) {
	if s.getUserWithdrawsFn == nil {
		return nil, nil
	}
	return s.getUserWithdrawsFn(ctx, userID)
}

func (s *stubBalanceManager) GetUserBalance(ctx context.Context, userID uuid.UUID) (*apiModel.ReadBalanceApi, error) {
	if s.getUserBalanceFn == nil {
		return nil, nil
	}
	return s.getUserBalanceFn(ctx, userID)
}

func TestWithdrawBalanceHandler(t *testing.T) {
	t.Parallel()

	appLogger := zap.NewNop()
	userID := uuid.New()

	tests := []struct {
		name         string
		auth         *stubAuthService
		body         string
		manager      *stubBalanceManager
		expectedCode int
	}{
		{
			name:         "returns internal server error when user is missing in context",
			auth:         &stubAuthService{err: errors.New("missing user")},
			body:         `{"order":"79927398713","sum":50}`,
			expectedCode: http.StatusInternalServerError,
			manager: &stubBalanceManager{
				withdrawUserBalanceFn: func(context.Context, *apiModel.BalanceApi, *dto.UserDto, *zap.Logger) (*withdraw.WithdrawDto, error) {
					t.Fatal("balance manager should not be called when auth fails")
					return nil, nil
				},
			},
		},
		{
			name:         "returns bad request on invalid json",
			auth:         &stubAuthService{user: &dto.UserDto{ID: userID}},
			body:         `{"order":`,
			expectedCode: http.StatusBadRequest,
			manager: &stubBalanceManager{
				withdrawUserBalanceFn: func(context.Context, *apiModel.BalanceApi, *dto.UserDto, *zap.Logger) (*withdraw.WithdrawDto, error) {
					t.Fatal("balance manager should not be called when request json is invalid")
					return nil, nil
				},
			},
		},
		{
			name:         "returns payment required when balance is not enough",
			auth:         &stubAuthService{user: &dto.UserDto{ID: userID}},
			body:         `{"order":"79927398713","sum":50}`,
			expectedCode: http.StatusPaymentRequired,
			manager: &stubBalanceManager{
				withdrawUserBalanceFn: func(context.Context, *apiModel.BalanceApi, *dto.UserDto, *zap.Logger) (*withdraw.WithdrawDto, error) {
					return nil, userBalance.ErrorNotEnoughBalance
				},
			},
		},
		{
			name:         "returns unprocessable entity on invalid order number",
			auth:         &stubAuthService{user: &dto.UserDto{ID: userID}},
			body:         `{"order":"123","sum":50}`,
			expectedCode: http.StatusUnprocessableEntity,
			manager: &stubBalanceManager{
				withdrawUserBalanceFn: func(context.Context, *apiModel.BalanceApi, *dto.UserDto, *zap.Logger) (*withdraw.WithdrawDto, error) {
					return nil, order.ErrorInvalidOrderNum
				},
			},
		},
		{
			name:         "returns internal server error on unexpected failure",
			auth:         &stubAuthService{user: &dto.UserDto{ID: userID}},
			body:         `{"order":"79927398713","sum":50}`,
			expectedCode: http.StatusInternalServerError,
			manager: &stubBalanceManager{
				withdrawUserBalanceFn: func(context.Context, *apiModel.BalanceApi, *dto.UserDto, *zap.Logger) (*withdraw.WithdrawDto, error) {
					return nil, errors.New("storage failure")
				},
			},
		},
		{
			name:         "returns ok on successful withdraw",
			auth:         &stubAuthService{user: &dto.UserDto{ID: userID}},
			body:         `{"order":"79927398713","sum":50}`,
			expectedCode: http.StatusOK,
			manager: &stubBalanceManager{
				withdrawUserBalanceFn: func(_ context.Context, api *apiModel.BalanceApi, user *dto.UserDto, _ *zap.Logger) (*withdraw.WithdrawDto, error) {
					if user.ID != userID {
						t.Fatalf("unexpected user id: got %s want %s", user.ID, userID)
					}
					if api.Order != "79927398713" {
						t.Fatalf("unexpected order: got %q", api.Order)
					}
					if api.Sum != 50 {
						t.Fatalf("unexpected sum: got %v want 50", api.Sum)
					}
					return &withdraw.WithdrawDto{UserID: userID, Sum: 50}, nil
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodPost, "/api/user/balance/withdraw", bytes.NewBufferString(tt.body))
			rr := httptest.NewRecorder()

			WithdrawBalanceHandler(tt.auth, tt.manager, appLogger).ServeHTTP(rr, req)

			if rr.Code != tt.expectedCode {
				t.Fatalf("unexpected status code: got %d want %d", rr.Code, tt.expectedCode)
			}
		})
	}
}

func TestGetWithdrawBalanceHandler(t *testing.T) {
	t.Parallel()

	appLogger := zap.NewNop()
	userID := uuid.New()
	processedAt := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		auth         *stubAuthService
		manager      *stubBalanceManager
		expectedCode int
		assert       func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name:         "returns internal server error when user is missing in context",
			auth:         &stubAuthService{err: errors.New("missing user")},
			expectedCode: http.StatusInternalServerError,
			manager: &stubBalanceManager{
				getUserWithdrawsFn: func(context.Context, uuid.UUID) ([]apiModel.ReadWithdrawApi, error) {
					t.Fatal("balance manager should not be called when auth fails")
					return nil, nil
				},
			},
		},
		{
			name:         "returns no content when withdraws are absent",
			auth:         &stubAuthService{user: &dto.UserDto{ID: userID}},
			expectedCode: http.StatusNoContent,
			manager: &stubBalanceManager{
				getUserWithdrawsFn: func(context.Context, uuid.UUID) ([]apiModel.ReadWithdrawApi, error) {
					return nil, postgres.ErrorNotFoundBalanceRows
				},
			},
		},
		{
			name:         "returns internal server error on unexpected failure",
			auth:         &stubAuthService{user: &dto.UserDto{ID: userID}},
			expectedCode: http.StatusInternalServerError,
			manager: &stubBalanceManager{
				getUserWithdrawsFn: func(context.Context, uuid.UUID) ([]apiModel.ReadWithdrawApi, error) {
					return nil, errors.New("storage failure")
				},
			},
		},
		{
			name:         "returns withdraw list as json",
			auth:         &stubAuthService{user: &dto.UserDto{ID: userID}},
			expectedCode: http.StatusOK,
			manager: &stubBalanceManager{
				getUserWithdrawsFn: func(_ context.Context, gotUserID uuid.UUID) ([]apiModel.ReadWithdrawApi, error) {
					if gotUserID != userID {
						t.Fatalf("unexpected user id: got %s want %s", gotUserID, userID)
					}
					return []apiModel.ReadWithdrawApi{
						{
							OrderID:     "79927398713",
							Sum:         75.5,
							ProcessedAt: &processedAt,
						},
					}, nil
				},
			},
			assert: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var got []apiModel.ReadWithdrawApi
				if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}
				if len(got) != 1 {
					t.Fatalf("unexpected response length: got %d want 1", len(got))
				}
				if got[0].OrderID != "79927398713" {
					t.Fatalf("unexpected order id: got %q", got[0].OrderID)
				}
				if got[0].Sum != 75.5 {
					t.Fatalf("unexpected sum: got %v", got[0].Sum)
				}
				if got[0].ProcessedAt == nil || !got[0].ProcessedAt.Equal(processedAt) {
					t.Fatalf("unexpected processed time: got %v want %v", got[0].ProcessedAt, processedAt)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/api/user/withdrawals", nil)
			rr := httptest.NewRecorder()

			GetWithdrawBalanceHandler(tt.auth, tt.manager, appLogger).ServeHTTP(rr, req)

			if rr.Code != tt.expectedCode {
				t.Fatalf("unexpected status code: got %d want %d", rr.Code, tt.expectedCode)
			}
			if tt.assert != nil {
				tt.assert(t, rr)
			}
		})
	}
}

func TestGetUserBalanceHandler(t *testing.T) {
	t.Parallel()

	appLogger := zap.NewNop()
	userID := uuid.New()

	tests := []struct {
		name         string
		auth         *stubAuthService
		manager      *stubBalanceManager
		expectedCode int
		assert       func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name:         "returns internal server error when user is missing in context",
			auth:         &stubAuthService{err: errors.New("missing user")},
			expectedCode: http.StatusInternalServerError,
			manager: &stubBalanceManager{
				getUserBalanceFn: func(context.Context, uuid.UUID) (*apiModel.ReadBalanceApi, error) {
					t.Fatal("balance manager should not be called when auth fails")
					return nil, nil
				},
			},
		},
		{
			name:         "returns internal server error on unexpected failure",
			auth:         &stubAuthService{user: &dto.UserDto{ID: userID}},
			expectedCode: http.StatusInternalServerError,
			manager: &stubBalanceManager{
				getUserBalanceFn: func(context.Context, uuid.UUID) (*apiModel.ReadBalanceApi, error) {
					return nil, errors.New("storage failure")
				},
			},
		},
		{
			name:         "returns balance as json",
			auth:         &stubAuthService{user: &dto.UserDto{ID: userID}},
			expectedCode: http.StatusOK,
			manager: &stubBalanceManager{
				getUserBalanceFn: func(_ context.Context, gotUserID uuid.UUID) (*apiModel.ReadBalanceApi, error) {
					if gotUserID != userID {
						t.Fatalf("unexpected user id: got %s want %s", gotUserID, userID)
					}
					return &apiModel.ReadBalanceApi{Current: 150.25, Withdrawn: 49.75}, nil
				},
			},
			assert: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var got apiModel.ReadBalanceApi
				if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}
				if got.Current != 150.25 {
					t.Fatalf("unexpected current balance: got %v", got.Current)
				}
				if got.Withdrawn != 49.75 {
					t.Fatalf("unexpected withdrawn amount: got %v", got.Withdrawn)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/api/user/balance", nil)
			rr := httptest.NewRecorder()

			GetUserBalanceHandler(tt.auth, tt.manager, appLogger).ServeHTTP(rr, req)

			if rr.Code != tt.expectedCode {
				t.Fatalf("unexpected status code: got %d want %d", rr.Code, tt.expectedCode)
			}
			if tt.assert != nil {
				tt.assert(t, rr)
			}
		})
	}
}
