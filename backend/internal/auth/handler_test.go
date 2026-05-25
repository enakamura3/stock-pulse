package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAuthService é um mock para a interface AuthService
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Register(ctx context.Context, name, email, password string) (*User, error) {
	args := m.Called(ctx, name, email, password)
	if args.Get(0) != nil {
		return args.Get(0).(*User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAuthService) Login(ctx context.Context, email, password string) (*User, string, string, error) {
	args := m.Called(ctx, email, password)
	if args.Get(0) != nil {
		return args.Get(0).(*User), args.String(1), args.String(2), args.Error(3)
	}
	return nil, "", "", args.Error(3)
}

func (m *MockAuthService) RevokeRefreshToken(ctx context.Context, token string) error {
	return m.Called(ctx, token).Error(0)
}

func (m *MockAuthService) ValidateRefreshToken(ctx context.Context, token string) (string, error) {
	args := m.Called(ctx, token)
	return args.String(0), args.Error(1)
}

func (m *MockAuthService) GetUserByID(ctx context.Context, id string) (*User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) != nil {
		return args.Get(0).(*User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAuthService) GenerateAccessToken(user *User) (string, error) {
	args := m.Called(user)
	return args.String(0), args.Error(1)
}

func TestHandler_Register(t *testing.T) {
	tests := []struct {
		name         string
		payload      interface{}
		mockSetup    func(*MockAuthService)
		expectedCode int
	}{
		{
			name: "Success",
			payload: map[string]string{
				"name":     "Test",
				"email":    "test@test.com",
				"password": "password123",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("Register", mock.Anything, "Test", "test@test.com", "password123").Return(&User{ID: "1", Email: "test@test.com"}, nil)
			},
			expectedCode: http.StatusCreated,
		},
		{
			name:         "Invalid JSON",
			payload:      "invalid_json",
			mockSetup:    func(m *MockAuthService) {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "Service Error",
			payload: map[string]string{
				"name":     "Test",
				"email":    "test@test.com",
				"password": "short",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("Register", mock.Anything, "Test", "test@test.com", "short").Return(nil, errors.New("senha fraca"))
			},
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(MockAuthService)
			tt.mockSetup(m)
			h := NewHandler(m)

			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(body))
			rec := httptest.NewRecorder()

			h.Register(rec, req)

			assert.Equal(t, tt.expectedCode, rec.Code)
			m.AssertExpectations(t)
		})
	}
}

func TestHandler_Login(t *testing.T) {
	tests := []struct {
		name         string
		payload      interface{}
		mockSetup    func(*MockAuthService)
		expectedCode int
	}{
		{
			name: "Success",
			payload: map[string]string{
				"email":    "test@test.com",
				"password": "password123",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("Login", mock.Anything, "test@test.com", "password123").Return(&User{ID: "1"}, "access", "refresh", nil)
			},
			expectedCode: http.StatusOK,
		},
		{
			name:         "Invalid JSON",
			payload:      "invalid",
			mockSetup:    func(m *MockAuthService) {},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "Invalid Credentials",
			payload: map[string]string{
				"email":    "wrong@test.com",
				"password": "wrong",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("Login", mock.Anything, "wrong@test.com", "wrong").Return(nil, "", "", errors.New("erro"))
			},
			expectedCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(MockAuthService)
			tt.mockSetup(m)
			h := NewHandler(m)

			body, _ := json.Marshal(tt.payload)
			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
			rec := httptest.NewRecorder()

			h.Login(rec, req)

			assert.Equal(t, tt.expectedCode, rec.Code)
			if tt.expectedCode == http.StatusOK {
				cookies := rec.Result().Cookies()
				assert.Len(t, cookies, 2)
			}
			m.AssertExpectations(t)
		})
	}
}

func TestHandler_Logout(t *testing.T) {
	m := new(MockAuthService)
	m.On("RevokeRefreshToken", mock.Anything, "refresh_token_val").Return(nil)
	h := NewHandler(m)

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "refresh_token_val"})
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	cookies := rec.Result().Cookies()
	assert.Len(t, cookies, 2)
	for _, c := range cookies {
		assert.True(t, c.MaxAge < 0)
	}
	m.AssertExpectations(t)
}

func TestHandler_Logout_NoCookie(t *testing.T) {
	m := new(MockAuthService)
	h := NewHandler(m)

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	rec := httptest.NewRecorder()

	h.Logout(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	cookies := rec.Result().Cookies()
	assert.Len(t, cookies, 2) // Ainda deve limpar cookies
}

func TestHandler_Me(t *testing.T) {
	tests := []struct {
		name         string
		ctxUserID    interface{}
		mockSetup    func(*MockAuthService)
		expectedCode int
	}{
		{
			name:      "No User ID in Context",
			ctxUserID: nil,
			mockSetup: func(m *MockAuthService) {},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:      "Empty User ID",
			ctxUserID: "",
			mockSetup: func(m *MockAuthService) {},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:      "User Not Found",
			ctxUserID: "2",
			mockSetup: func(m *MockAuthService) {
				m.On("GetUserByID", mock.Anything, "2").Return(nil, errors.New("not found"))
			},
			expectedCode: http.StatusNotFound,
		},
		{
			name:      "Success",
			ctxUserID: "1",
			mockSetup: func(m *MockAuthService) {
				m.On("GetUserByID", mock.Anything, "1").Return(&User{ID: "1"}, nil)
			},
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(MockAuthService)
			tt.mockSetup(m)
			h := NewHandler(m)

			req := httptest.NewRequest(http.MethodGet, "/me", nil)
			if tt.ctxUserID != nil {
				ctx := context.WithValue(req.Context(), UserIDKey, tt.ctxUserID)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()

			h.Me(rec, req)

			assert.Equal(t, tt.expectedCode, rec.Code)
			m.AssertExpectations(t)
		})
	}
}

func TestHandler_Refresh(t *testing.T) {
	tests := []struct {
		name         string
		cookie       *http.Cookie
		mockSetup    func(*MockAuthService)
		expectedCode int
	}{
		{
			name: "No Cookie",
			cookie: nil,
			mockSetup: func(m *MockAuthService) {},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name: "Invalid Token",
			cookie: &http.Cookie{Name: "refresh_token", Value: "invalid"},
			mockSetup: func(m *MockAuthService) {
				m.On("ValidateRefreshToken", mock.Anything, "invalid").Return("", errors.New("invalid"))
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name: "User Not Found",
			cookie: &http.Cookie{Name: "refresh_token", Value: "valid"},
			mockSetup: func(m *MockAuthService) {
				m.On("ValidateRefreshToken", mock.Anything, "valid").Return("1", nil)
				m.On("GetUserByID", mock.Anything, "1").Return(nil, errors.New("not found"))
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name: "Failed to generate token",
			cookie: &http.Cookie{Name: "refresh_token", Value: "valid"},
			mockSetup: func(m *MockAuthService) {
				m.On("ValidateRefreshToken", mock.Anything, "valid").Return("1", nil)
				m.On("GetUserByID", mock.Anything, "1").Return(&User{ID: "1"}, nil)
				m.On("GenerateAccessToken", mock.Anything).Return("", errors.New("error generating token"))
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name: "Success",
			cookie: &http.Cookie{Name: "refresh_token", Value: "valid"},
			mockSetup: func(m *MockAuthService) {
				m.On("ValidateRefreshToken", mock.Anything, "valid").Return("1", nil)
				m.On("GetUserByID", mock.Anything, "1").Return(&User{ID: "1"}, nil)
				m.On("GenerateAccessToken", mock.Anything).Return("new_access", nil)
			},
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(MockAuthService)
			tt.mockSetup(m)
			h := NewHandler(m)

			req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}
			rec := httptest.NewRecorder()

			h.Refresh(rec, req)

			assert.Equal(t, tt.expectedCode, rec.Code)
			if tt.expectedCode == http.StatusOK {
				cookies := rec.Result().Cookies()
				assert.Len(t, cookies, 1) // SetCookie was called for access_token
				assert.Equal(t, "new_access", cookies[0].Value)
			}
			m.AssertExpectations(t)
		})
	}
}

// Para atingir 100% de cobertura no handler precisamos simular falha no json.Marshal
type failMarshal struct{}

func (f failMarshal) MarshalJSON() ([]byte, error) {
	return nil, errors.New("error")
}

func TestHandler_RespondWithJSON_Error(t *testing.T) {
	h := NewHandler(new(MockAuthService))
	rec := httptest.NewRecorder()

	h.respondWithJSON(rec, http.StatusOK, failMarshal{})
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
