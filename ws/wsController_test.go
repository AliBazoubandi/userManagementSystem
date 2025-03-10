package ws

import (
	"bytes"
	_ "context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgtype"
	"main/db"
	"main/mocks"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockRows is a custom mock implementation of pgx.Rows
// MockRows is a mock implementation of pgx.Rows
type MockRows struct {
	rows   [][]interface{} // Simulated rows
	index  int             // Current row index
	closed bool            // Whether the rows are closed
}

// NewMockRows creates a new MockRows instance
func NewMockRows(rows [][]interface{}) *MockRows {
	return &MockRows{
		rows:  rows,
		index: -1,
	}
}

// Next advances to the next row
func (m *MockRows) Next() bool {
	m.index++
	return m.index < len(m.rows)
}

// Scan copies the current row's values into the provided destinations
func (m *MockRows) Scan(dest ...interface{}) error {
	if m.index < 0 || m.index >= len(m.rows) {
		return pgx.ErrNoRows
	}
	for i, val := range m.rows[m.index] {
		switch dest[i].(type) {
		case *int32:
			*dest[i].(*int32) = val.(int32)
		case *string:
			*dest[i].(*string) = val.(string)
		default:
			return errors.New("unsupported type")
		}
	}
	return nil
}

// Err returns any error encountered during iteration
func (m *MockRows) Err() error {
	return nil
}

// Close closes the rows
func (m *MockRows) Close() {
	m.closed = true
}

// FieldDescriptions returns the field descriptions of the rows
func (m *MockRows) FieldDescriptions() []pgconn.FieldDescription {
	return []pgconn.FieldDescription{
		{Name: "id", DataTypeOID: pgtype.Int4OID},
		{Name: "name", DataTypeOID: pgtype.TextOID},
	}
}

// RawValues returns the raw values of the current row
func (m *MockRows) RawValues() [][]byte {
	if m.index >= len(m.rows) {
		return nil
	}
	rawValues := make([][]byte, len(m.rows[m.index]))
	for i, val := range m.rows[m.index] {
		switch v := val.(type) {
		case int32:
			rawValues[i] = []byte(string(v))
		case string:
			rawValues[i] = []byte(v)
		default:
			rawValues[i] = []byte{}
		}
	}
	return rawValues
}

// CommandTag returns the command tag for the query
func (m *MockRows) CommandTag() pgconn.CommandTag {
	return pgconn.NewCommandTag("SELECT 1")
}

// Values returns the decoded values of the current row
func (m *MockRows) Values() ([]interface{}, error) {
	if m.index < 0 || m.index >= len(m.rows) {
		return nil, pgx.ErrNoRows
	}
	return m.rows[m.index], nil
}

// Conn returns the underlying connection (not implemented in mock)
func (m *MockRows) Conn() *pgx.Conn {
	return nil
}

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

func TestCreateRoom(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		input        map[string]interface{}
		username     string
		mockBehavior func(mockDB *mocks.DBTX)
		expectedCode int
		expectedErr  bool
	}{
		{
			name: "successful room creation",
			input: map[string]interface{}{
				"name": "Test Room",
			},
			username: "tester",
			mockBehavior: func(mockDB *mocks.DBTX) {
				mockRow := new(MockRow)
				mockRow.On("Scan",
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Run(func(args mock.Arguments) {
					*args.Get(0).(*int32) = 1
					*args.Get(1).(*string) = "Test Room"
				}).Return(nil)

				mockDB.On("QueryRow",
					mock.Anything,
					mock.Anything,
				).Return(mockRow)

				// Mock AddUserToRoom
				mockDB.On("QueryRow",
					mock.Anything,
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return(mockRow)
			},
			expectedCode: http.StatusCreated,
			expectedErr:  false,
		},
		{
			name: "unauthorized request",
			input: map[string]interface{}{
				"name": "Test Room",
			},
			username:     "",
			mockBehavior: func(mockDB *mocks.DBTX) {},
			expectedCode: http.StatusUnauthorized,
			expectedErr:  true,
		},
		{
			name: "invalid request body",
			input: map[string]interface{}{
				"name":    "Test Room",
				"invalid": "data",
			},
			username:     "tester",
			mockBehavior: func(mockDB *mocks.DBTX) {},
			expectedCode: http.StatusBadRequest,
			expectedErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &mocks.DBTX{}
			queries := db.New(mockDB)
			logger, _ := zap.NewDevelopment()
			hub := NewHub()
			wsc := NewWsController(queries, hub, logger)

			tt.mockBehavior(mockDB)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			if tt.username != "" {
				c.Set("username", tt.username)
			}

			jsonValue, _ := json.Marshal(tt.input)
			c.Request, _ = http.NewRequest("POST", "ws/create-room", bytes.NewBuffer(jsonValue))
			c.Request.Header.Set("Content-Type", "application/json")

			wsc.CreateRoom(c)

			assert.Equal(t, tt.expectedCode, w.Code)

			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if !tt.expectedErr {
				assert.Contains(t, response, "id")
				assert.Contains(t, response, "name")
			} else {
				assert.Contains(t, response, "error")
			}
		})
	}
}

func TestGetRooms(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		mockBehavior func(mockDB *mocks.DBTX)
		expectedCode int
		expectedErr  bool
	}{
		{
			name: "successful get rooms",
			mockBehavior: func(mockDB *mocks.DBTX) {
				mockRows := NewMockRows([][]interface{}{
					{int32(1), "Test Room"},
				})

				mockDB.On("Query",
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return(mockRows, nil)
			},
			expectedCode: http.StatusOK,
			expectedErr:  false,
		},
		{
			name: "database error",
			mockBehavior: func(mockDB *mocks.DBTX) {
				mockDB.On("Query",
					mock.Anything,
					mock.Anything,
				).Return(nil, pgx.ErrNoRows)
			},
			expectedCode: http.StatusInternalServerError,
			expectedErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &mocks.DBTX{}
			queries := db.New(mockDB)
			logger, _ := zap.NewDevelopment()
			hub := NewHub()
			wsc := NewWsController(queries, hub, logger)

			tt.mockBehavior(mockDB)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "ws/getRooms", nil)

			wsc.GetRooms(c)

			assert.Equal(t, tt.expectedCode, w.Code)

			if tt.expectedErr {
				// Handle error case
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "error")
			} else {
				// Handle success case
				var response []map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, 1, len(response))
				assert.Equal(t, float64(1), response[0]["id"])
				assert.Equal(t, "Test Room", response[0]["name"])
			}
		})
	}
}

func TestGetClients(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		roomID       string
		mockBehavior func(mockDB *mocks.DBTX)
		expectedCode int
		expectedErr  bool
	}{
		{
			name:   "successful get clients",
			roomID: "1",
			mockBehavior: func(mockDB *mocks.DBTX) {
				mockRowsForQueryRow := NewMockRows([][]interface{}{
					{int32(1), "Test Room"},
				})

				mockDB.On("QueryRow",
					mock.Anything, // context.Context
					mock.Anything, // query string
					mock.Anything, // arguments (if any)
				).Run(func(args mock.Arguments) {
					mockRowsForQueryRow.Next()
				}).Return(mockRowsForQueryRow)

				mockRowsForQuery := NewMockRows([][]interface{}{
					{int32(1), "Client 1"},
					{int32(2), "Client 2"},
				})

				mockDB.On("Query",
					mock.Anything, // context.Context
					mock.Anything, // query string
					mock.Anything, // arguments (if any)
				).Return(mockRowsForQuery, nil)
			},
			expectedCode: http.StatusOK,
			expectedErr:  false,
		},
		{
			name:         "invalid room id",
			roomID:       "invalid",
			mockBehavior: func(mockDB *mocks.DBTX) {},
			expectedCode: http.StatusBadRequest,
			expectedErr:  true,
		},
		{
			name:   "room not found",
			roomID: "999",
			mockBehavior: func(mockDB *mocks.DBTX) {
				mockRow := new(MockRow)
				mockRow.On("Scan",
					mock.Anything,
					mock.Anything,
				).Return(pgx.ErrNoRows)

				mockDB.On("QueryRow",
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return(mockRow)
			},
			expectedCode: http.StatusBadRequest,
			expectedErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &mocks.DBTX{}
			queries := db.New(mockDB)
			logger, _ := zap.NewDevelopment()
			hub := NewHub()
			wsc := NewWsController(queries, hub, logger)

			tt.mockBehavior(mockDB)

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "ws/getClients/"+tt.roomID, nil)
			c.Params = []gin.Param{{Key: "roomId", Value: tt.roomID}}

			wsc.GetClients(c)

			assert.Equal(t, tt.expectedCode, w.Code)

			if tt.expectedErr {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "error")
			} else {
				var response []map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, 2, len(response))
				assert.Equal(t, float64(1), response[0]["id"])
				assert.Equal(t, "Client 1", response[0]["username"])
				assert.Equal(t, float64(2), response[1]["id"])
				assert.Equal(t, "Client 2", response[1]["username"])
			}
		})
	}
}
