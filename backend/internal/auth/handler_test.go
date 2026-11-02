package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/onigiri/stock-pulse/backend/internal/config"
	"github.com/onigiri/stock-pulse/backend/internal/httputils"
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

func (m *MockAuthService) RotateRefreshToken(ctx context.Context, oldToken string) (string, string, error) {
	args := m.Called(ctx, oldToken)
	return args.String(0), args.String(1), args.Error(2)
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

func (m *MockAuthService) GenerateRefreshToken(ctx context.Context, userID string) (string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.Error(1)
}

func (m *MockAuthService) UpdateProfile(ctx context.Context, id, name, email string) (*User, error) {
	args := m.Called(ctx, id, name, email)
	if args.Get(0) != nil {
		return args.Get(0).(*User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAuthService) UpdatePassword(ctx context.Context, id, currentPassword, newPassword string) error {
	return m.Called(ctx, id, currentPassword, newPassword).Error(0)
}

func (m *MockAuthService) DeleteUser(ctx context.Context, id, currentPassword string) error {
	return m.Called(ctx, id, currentPassword).Error(0)
}

func (m *MockAuthService) ListUsers(ctx context.Context) ([]*User, error) {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).([]*User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockAuthService) UpdateUserRole(ctx context.Context, callerID, targetID, newRole string) error {
	return m.Called(ctx, callerID, targetID, newRole).Error(0)
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
		{
			name: "Too Many Requests Rate Limited By Email",
			payload: map[string]string{
				"email":    "bruteforce@test.com",
				"password": "wrong",
			},
			mockSetup: func(m *MockAuthService) {
				m.On("Login", mock.Anything, "bruteforce@test.com", "wrong").Return(nil, "", "", errors.New("muitas tentativas de login para este e-mail. Tente novamente mais tarde"))
			},
			expectedCode: http.StatusTooManyRequests,
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
			name:         "No User ID in Context",
			ctxUserID:    nil,
			mockSetup:    func(m *MockAuthService) {},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "Empty User ID",
			ctxUserID:    "",
			mockSetup:    func(m *MockAuthService) {},
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
			name:         "No Cookie",
			cookie:       nil,
			mockSetup:    func(m *MockAuthService) {},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:   "Invalid Token",
			cookie: &http.Cookie{Name: "refresh_token", Value: "invalid"},
			mockSetup: func(m *MockAuthService) {
				m.On("RotateRefreshToken", mock.Anything, "invalid").Return("", "", errors.New("invalid"))
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:   "User Not Found",
			cookie: &http.Cookie{Name: "refresh_token", Value: "valid"},
			mockSetup: func(m *MockAuthService) {
				m.On("RotateRefreshToken", mock.Anything, "valid").Return("1", "new_refresh", nil)
				m.On("GetUserByID", mock.Anything, "1").Return(nil, errors.New("not found"))
			},
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:   "Failed to generate token",
			cookie: &http.Cookie{Name: "refresh_token", Value: "valid"},
			mockSetup: func(m *MockAuthService) {
				m.On("RotateRefreshToken", mock.Anything, "valid").Return("1", "new_refresh", nil)
				m.On("GetUserByID", mock.Anything, "1").Return(&User{ID: "1"}, nil)
				m.On("GenerateAccessToken", mock.Anything).Return("", errors.New("error generating token"))
			},
			expectedCode: http.StatusInternalServerError,
		},
		{
			name:   "Success",
			cookie: &http.Cookie{Name: "refresh_token", Value: "valid"},
			mockSetup: func(m *MockAuthService) {
				m.On("RotateRefreshToken", mock.Anything, "valid").Return("1", "new_refresh", nil)
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
				assert.Len(t, cookies, 2) // SetCookie chamado para access_token e refresh_token
				var foundAccess, foundRefresh bool
				for _, c := range cookies {
					if c.Name == "access_token" {
						assert.Equal(t, "new_access", c.Value)
						foundAccess = true
					}
					if c.Name == "refresh_token" {
						assert.Equal(t, "new_refresh", c.Value)
						foundRefresh = true
					}
				}
				assert.True(t, foundAccess)
				assert.True(t, foundRefresh)
			}
			if tt.expectedCode == http.StatusUnauthorized && tt.cookie != nil {
				cookies := rec.Result().Cookies()
				assert.Len(t, cookies, 2)
				for _, c := range cookies {
					assert.True(t, c.MaxAge < 0)
				}
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
	rec := httptest.NewRecorder()

	httputils.RespondWithJSON(rec, http.StatusOK, failMarshal{})
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestHandler_UpdateProfile(t *testing.T) {
	t.Run("Unauthorized", func(t *testing.T) {
		m := new(MockAuthService)
		h := NewHandler(m)
		req := httptest.NewRequest(http.MethodPut, "/user/profile", nil)
		rec := httptest.NewRecorder()
		h.UpdateProfile(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		m := new(MockAuthService)
		h := NewHandler(m)
		req := httptest.NewRequest(http.MethodPut, "/user/profile", strings.NewReader("invalid"))
		ctx := context.WithValue(req.Context(), UserIDKey, "1")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		h.UpdateProfile(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		m := new(MockAuthService)
		m.On("UpdateProfile", mock.Anything, "1", "NewName", "new@test.com").Return(nil, errors.New("err"))
		h := NewHandler(m)
		payload := `{"name":"NewName","email":"new@test.com"}`
		req := httptest.NewRequest(http.MethodPut, "/user/profile", strings.NewReader(payload))
		ctx := context.WithValue(req.Context(), UserIDKey, "1")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		h.UpdateProfile(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Success", func(t *testing.T) {
		m := new(MockAuthService)
		m.On("UpdateProfile", mock.Anything, "1", "NewName", "new@test.com").Return(&User{ID: "1", Name: "NewName", Email: "new@test.com"}, nil)
		h := NewHandler(m)

		payload := `{"name":"NewName","email":"new@test.com"}`
		req := httptest.NewRequest(http.MethodPut, "/user/profile", strings.NewReader(payload))
		ctx := context.WithValue(req.Context(), UserIDKey, "1")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		h.UpdateProfile(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var resp User
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, "NewName", resp.Name)
		assert.Equal(t, "new@test.com", resp.Email)
		m.AssertExpectations(t)
	})
}

func TestHandler_UpdatePassword(t *testing.T) {
	t.Run("Unauthorized", func(t *testing.T) {
		m := new(MockAuthService)
		h := NewHandler(m)
		req := httptest.NewRequest(http.MethodPut, "/user/password", nil)
		rec := httptest.NewRecorder()
		h.UpdatePassword(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		m := new(MockAuthService)
		h := NewHandler(m)
		req := httptest.NewRequest(http.MethodPut, "/user/password", strings.NewReader("invalid"))
		ctx := context.WithValue(req.Context(), UserIDKey, "1")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		h.UpdatePassword(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Service Error", func(t *testing.T) {
		m := new(MockAuthService)
		m.On("UpdatePassword", mock.Anything, "1", "oldpass", "newpass").Return(errors.New("err"))
		h := NewHandler(m)
		payload := `{"current_password":"oldpass","new_password":"newpass"}`
		req := httptest.NewRequest(http.MethodPut, "/user/password", strings.NewReader(payload))
		ctx := context.WithValue(req.Context(), UserIDKey, "1")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		h.UpdatePassword(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Success", func(t *testing.T) {
		m := new(MockAuthService)
		user := &User{ID: "1", Email: "test@test.com"}
		m.On("UpdatePassword", mock.Anything, "1", "oldpass", "newpass").Return(nil)
		m.On("GetUserByID", mock.Anything, "1").Return(user, nil)
		m.On("GenerateAccessToken", user).Return("new_access_token", nil)
		m.On("GenerateRefreshToken", mock.Anything, "1").Return("new_refresh_token", nil)
		h := NewHandler(m)

		payload := `{"current_password":"oldpass","new_password":"newpass"}`
		req := httptest.NewRequest(http.MethodPut, "/user/password", strings.NewReader(payload))
		ctx := context.WithValue(req.Context(), UserIDKey, "1")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		h.UpdatePassword(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		cookies := rec.Result().Cookies()
		assert.Len(t, cookies, 2)
		m.AssertExpectations(t)
	})
}

func TestHandler_DeleteUser(t *testing.T) {
	t.Run("Unauthorized", func(t *testing.T) {
		m := new(MockAuthService)
		h := NewHandler(m)
		req := httptest.NewRequest(http.MethodDelete, "/user", nil)
		rec := httptest.NewRecorder()
		h.DeleteUser(rec, req)
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("Invalid JSON Body", func(t *testing.T) {
		m := new(MockAuthService)
		h := NewHandler(m)
		req := httptest.NewRequest(http.MethodDelete, "/user", strings.NewReader("{invalid"))
		ctx := context.WithValue(req.Context(), UserIDKey, "1")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		h.DeleteUser(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("Password Validation Error", func(t *testing.T) {
		m := new(MockAuthService)
		m.On("DeleteUser", mock.Anything, "1", "wrong").Return(errors.New("senha atual incorreta"))
		h := NewHandler(m)
		req := httptest.NewRequest(http.MethodDelete, "/user", strings.NewReader(`{"current_password":"wrong"}`))
		ctx := context.WithValue(req.Context(), UserIDKey, "1")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		h.DeleteUser(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		m.AssertExpectations(t)
	})

	t.Run("Service Error", func(t *testing.T) {
		m := new(MockAuthService)
		m.On("DeleteUser", mock.Anything, "1", "secret123").Return(errors.New("db error"))
		h := NewHandler(m)
		req := httptest.NewRequest(http.MethodDelete, "/user", strings.NewReader(`{"current_password":"secret123"}`))
		ctx := context.WithValue(req.Context(), UserIDKey, "1")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		h.DeleteUser(rec, req)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
	})

	t.Run("Success with Cookie Revocation", func(t *testing.T) {
		m := new(MockAuthService)
		m.On("RevokeRefreshToken", mock.Anything, "refresh_val").Return(nil)
		m.On("DeleteUser", mock.Anything, "1", "secret123").Return(nil)
		h := NewHandler(m)

		req := httptest.NewRequest(http.MethodDelete, "/user", strings.NewReader(`{"current_password":"secret123"}`))
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "refresh_val"})
		ctx := context.WithValue(req.Context(), UserIDKey, "1")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		h.DeleteUser(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		// should clear cookies
		cookies := rec.Result().Cookies()
		assert.Len(t, cookies, 2)
		assert.Equal(t, "access_token", cookies[0].Name)
		assert.Equal(t, "refresh_token", cookies[1].Name)
		m.AssertExpectations(t)
	})
}

func TestHandler_NewHandler_CustomTTL(t *testing.T) {
	origAccess := config.Envs.JWTAccessTokenTTL
	origRefresh := config.Envs.JWTRefreshTokenTTL
	defer func() {
		config.Envs.JWTAccessTokenTTL = origAccess
		config.Envs.JWTRefreshTokenTTL = origRefresh
	}()

	config.Envs.JWTAccessTokenTTL = 30 * time.Minute
	config.Envs.JWTRefreshTokenTTL = 24 * time.Hour

	m := new(MockAuthService)
	h := NewHandler(m)

	assert.Equal(t, 30*time.Minute, h.accessTokenTTL)
	assert.Equal(t, 24*time.Hour, h.refreshTokenTTL)
}

func TestHandler_ListUsersAdmin(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		m := new(MockAuthService)
		expected := []*User{
			{ID: "1", Name: "Admin", Role: RoleAdmin},
			{ID: "2", Name: "User", Role: RoleUser},
		}
		m.On("ListUsers", mock.Anything).Return(expected, nil)
		h := NewHandler(m)

		req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
		rec := httptest.NewRecorder()

		h.ListUsersAdmin(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var actual []*User
		err := json.NewDecoder(rec.Body).Decode(&actual)
		assert.NoError(t, err)
		assert.Len(t, actual, 2)
		m.AssertExpectations(t)
	})

	t.Run("Service Error", func(t *testing.T) {
		m := new(MockAuthService)
		m.On("ListUsers", mock.Anything).Return(([]*User)(nil), errors.New("db error"))
		h := NewHandler(m)

		req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
		rec := httptest.NewRecorder()

		h.ListUsersAdmin(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "Falha ao listar usuários")
		m.AssertExpectations(t)
	})
}

func TestHandler_UpdateUserRoleAdmin(t *testing.T) {
	t.Run("Missing ID", func(t *testing.T) {
		m := new(MockAuthService)
		h := NewHandler(m)

		req := httptest.NewRequest(http.MethodPatch, "/admin/users//role", strings.NewReader(`{"role":"ADMIN"}`))
		rec := httptest.NewRecorder()

		h.UpdateUserRoleAdmin(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "ID do usuário é obrigatório")
	})

	t.Run("Invalid Payload", func(t *testing.T) {
		m := new(MockAuthService)
		h := NewHandler(m)

		req := httptest.NewRequest(http.MethodPatch, "/admin/users/1/role", strings.NewReader(`invalid-json`))
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "1")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		rec := httptest.NewRecorder()

		h.UpdateUserRoleAdmin(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Corpo da requisição inválido")
	})

	t.Run("User not found", func(t *testing.T) {
		m := new(MockAuthService)
		m.On("UpdateUserRole", mock.Anything, "", "1", "ADMIN").Return(errors.New("usuário não encontrado"))
		h := NewHandler(m)

		req := httptest.NewRequest(http.MethodPatch, "/admin/users/1/role", strings.NewReader(`{"role":"ADMIN"}`))
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "1")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		rec := httptest.NewRecorder()

		h.UpdateUserRoleAdmin(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "usuário não encontrado")
		m.AssertExpectations(t)
	})

	t.Run("Invalid role", func(t *testing.T) {
		m := new(MockAuthService)
		m.On("UpdateUserRole", mock.Anything, "", "1", "INVALID").Return(errors.New("papel inválido, deve ser USER ou ADMIN"))
		h := NewHandler(m)

		req := httptest.NewRequest(http.MethodPatch, "/admin/users/1/role", strings.NewReader(`{"role":"INVALID"}`))
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "1")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		rec := httptest.NewRecorder()

		h.UpdateUserRoleAdmin(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "papel inválido")
		m.AssertExpectations(t)
	})

	t.Run("Self demotion rejected", func(t *testing.T) {
		m := new(MockAuthService)
		m.On("UpdateUserRole", mock.Anything, "adm-1", "adm-1", "USER").Return(errors.New("não é possível revogar o próprio privilégio de administrador"))
		h := NewHandler(m)

		req := httptest.NewRequest(http.MethodPatch, "/admin/users/adm-1/role", strings.NewReader(`{"role":"USER"}`))
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "adm-1")
		ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
		ctx = context.WithValue(ctx, UserIDKey, "adm-1")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		h.UpdateUserRoleAdmin(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "não é possível revogar")
		m.AssertExpectations(t)
	})

	t.Run("Service generic error", func(t *testing.T) {
		m := new(MockAuthService)
		m.On("UpdateUserRole", mock.Anything, "", "1", "ADMIN").Return(errors.New("database connection broken"))
		h := NewHandler(m)

		req := httptest.NewRequest(http.MethodPatch, "/admin/users/1/role", strings.NewReader(`{"role":"ADMIN"}`))
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "1")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		rec := httptest.NewRecorder()

		h.UpdateUserRoleAdmin(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "Falha ao atualizar papel do usuário")
		m.AssertExpectations(t)
	})

	t.Run("Success", func(t *testing.T) {
		m := new(MockAuthService)
		m.On("UpdateUserRole", mock.Anything, "adm-1", "2", "ADMIN").Return(nil)
		h := NewHandler(m)

		req := httptest.NewRequest(http.MethodPatch, "/admin/users/2/role", strings.NewReader(`{"role":"ADMIN"}`))
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", "2")
		ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
		ctx = context.WithValue(ctx, UserIDKey, "adm-1")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		h.UpdateUserRoleAdmin(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Papel do usuário atualizado com sucesso")
		m.AssertExpectations(t)
	})
}
