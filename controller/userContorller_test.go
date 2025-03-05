package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"main/db"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// MockRow implements pgx.Row interface
type MockRow struct {
	mock.Mock
}

func (m *MockRow) Scan(dest ...interface{}) error {
	args := m.Called(dest...)
	if err, ok := args.Get(0).(error); ok {
		return err
	}
	return nil
}

type MockDBTX struct {
	mock.Mock
}

func (m *MockDBTX) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	args := m.Called(ctx, sql, arguments)
	return args.Get(0).(pgconn.CommandTag), args.Error(1)
}

func (m *MockDBTX) Query(ctx context.Context, sql string, arguments ...interface{}) (pgx.Rows, error) {
	args := m.Called(ctx, sql, arguments)
	return args.Get(0).(pgx.Rows), args.Error(1)
}

func (m *MockDBTX) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return m.Called(ctx, sql, args).Get(0).(pgx.Row)
}

func TestSignUP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		input        db.CreateUserParams
		mockBehavior func(mockDB *MockDBTX)
		expectedCode int
		expectedErr  bool
	}{
		{
			name: "successful signup",
			input: db.CreateUserParams{
				Username: "tester",
				Password: "password",
				Email:    "testuser@test.com",
				Age:      pgtype.Int4{Int32: 25, Valid: true},
			},
			mockBehavior: func(mockDB *MockDBTX) {
				mockRow := new(MockRow)
				mockRow.On("Scan",
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Run(func(args mock.Arguments) {
					*args.Get(0).(*string) = "tester"
					*args.Get(1).(*string) = "testuser@test.com"
					*args.Get(2).(*pgtype.Int4) = pgtype.Int4{Int32: 25, Valid: true}
				}).Return(nil)

				mockDB.On("QueryRow",
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return(mockRow)
			},
			expectedCode: http.StatusCreated,
			expectedErr:  false,
		},
		{
			name: "Invalid Email",
			input: db.CreateUserParams{
				Username: "tester",
				Email:    "invalid-email",
				Password: "password123",
				Age:      pgtype.Int4{Int32: 25, Valid: true},
			},
			mockBehavior: func(mockDB *MockDBTX) {},
			expectedCode: http.StatusBadRequest,
			expectedErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(MockDBTX)
			queries := db.New(mockDB)
			mockRedisClient := redis.NewClient(&redis.Options{})
			logger, _ := zap.NewDevelopment()
			uc := NewUserController(queries, mockRedisClient, logger, true)

			tt.mockBehavior(mockDB)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			jsonValue, _ := json.Marshal(tt.input)
			c.Request, _ = http.NewRequest("POST", "/users/signup", bytes.NewBuffer(jsonValue))
			c.Request.Header.Set("Content-Type", "application/json")

			uc.SignUp(c)

			assert.Equal(t, tt.expectedCode, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if !tt.expectedErr {
				assert.Contains(t, response, "token")
				assert.Contains(t, response, "user")
				assert.Contains(t, response, "message")
				assert.Equal(t, "User created successfully", response["message"])
			} else {
				assert.Contains(t, response, "error")
			}
		})
	}
}

func TestLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

	tests := []struct {
		name         string
		input        LoginRequest
		mockBehavior func(mockDB *MockDBTX)
		expectedCode int
		expectedErr  bool
	}{
		{
			name: "successful login",
			input: LoginRequest{
				Username: "tester",
				Password: "password123",
			},
			mockBehavior: func(mockDB *MockDBTX) {
				mockRows := new(MockRow)
				mockRows.On("Scan",
					mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
				).Return(nil).Run(func(args mock.Arguments) {
					*args.Get(0).(*int32) = 1
					*args.Get(1).(*string) = "tester"
					*args.Get(2).(*string) = "testuser@test.com"
					*args.Get(3).(*string) = string(hashedPassword)
					*args.Get(4).(*pgtype.Int4) = pgtype.Int4{Int32: 25, Valid: true}
					*args.Get(5).(*pgtype.Int4) = pgtype.Int4{Int32: 1, Valid: true}
					*args.Get(6).(*pgtype.Timestamp) = pgtype.Timestamp{Time: time.Now(), Valid: true}
				})
				mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRows)
			},
			expectedCode: http.StatusOK,
			expectedErr:  false,
		},
		{
			name: "Invalid Credentials",
			input: LoginRequest{
				Username: "tester",
				Password: "wrong password",
			},
			mockBehavior: func(mockDB *MockDBTX) {
				mockRows := new(MockRow)
				mockRows.On("Scan",
					mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
				).Return(nil).Run(func(args mock.Arguments) {
					*args.Get(0).(*int32) = 1
					*args.Get(1).(*string) = "tester"
					*args.Get(2).(*string) = "testuser@test.com"
					*args.Get(3).(*string) = string(hashedPassword)
					*args.Get(4).(*pgtype.Int4) = pgtype.Int4{Int32: 25, Valid: true}
					*args.Get(5).(*pgtype.Int4) = pgtype.Int4{Int32: 1, Valid: true}
					*args.Get(6).(*pgtype.Timestamp) = pgtype.Timestamp{Time: time.Now(), Valid: true}
				})
				mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRows)
			},
			expectedCode: http.StatusUnauthorized,
			expectedErr:  true,
		},
		{
			name: "User not found",
			input: LoginRequest{
				Username: "non existent username",
				Password: "password123",
			},
			mockBehavior: func(mockDB *MockDBTX) {
				mockRow := new(MockRow)
				mockRow.On("Scan", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(pgx.ErrNoRows)
				mockDB.On("QueryRow", mock.Anything, mock.Anything, mock.Anything).Return(mockRow)
			},
			expectedCode: http.StatusNotFound,
			expectedErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := new(MockDBTX)
			queries := db.New(mockDB)
			mockRedisClient := redis.NewClient(&redis.Options{})
			logger, _ := zap.NewDevelopment()
			uc := NewUserController(queries, mockRedisClient, logger, true)

			tt.mockBehavior(mockDB)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			jsonValue, _ := json.Marshal(tt.input)
			c.Request, _ = http.NewRequest("POST", "/users/login", bytes.NewBuffer(jsonValue))
			c.Request.Header.Set("Content-Type", "application/json")

			uc.Login(c)

			assert.Equal(t, tt.expectedCode, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if !tt.expectedErr {
				assert.Contains(t, response, "token")
				assert.NotEmpty(t, response["token"])
			} else {
				assert.Contains(t, response, "error")
				assert.NotEmpty(t, response["error"])
			}
		})
	}
}
