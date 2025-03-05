package controller

import (
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"main/config"
	"main/db"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
)

type UserController struct {
	Queries     *db.Queries
	RedisClient *redis.Client
	Logger      *zap.Logger
}

var (
	userRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "user_requests",
		Help: "Total number of requests to the user-related endpoints",
	}, []string{"method", "endpoint"})
)
var prometheusRegistered bool = false

const routeForSingleUser = "/users/:id"

func NewUserController(queries *db.Queries, redisClient *redis.Client, logger *zap.Logger, isTest bool) *UserController {
	if !isTest && !prometheusRegistered {
		prometheus.MustRegister(userRequests)
		prometheusRegistered = true
	}
	return &UserController{Queries: queries, RedisClient: redisClient, Logger: logger}
}

type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

func GenerateJWT(username string) (string, error) {
	expirationTime := time.Now().Add(15 * time.Minute)

	claims := &Claims{
		Username: username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.AppConfig.JWT.Key))
}

// SignUp godoc
// @Summary Sign up a user
// @Description Register a new user with their username, email, and password
// @Tags users
// @Accept json
// @Produce json
// @Param user body db.CreateUserParams true "User Data"
// @Success 200 {object} db.User "Registered User"
// @Failure 400 {object} gin.H "Bad Request"
// @Failure 500 {object} gin.H "Internal Server Error"
// @Router /users/signup [post]
func (uc *UserController) SignUp(c *gin.Context) {
	userRequests.WithLabelValues("POST", "/users").Inc()
	var params db.CreateUserParams

	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if params.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username is required"})
		return
	}

	if params.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password is required"})
		return
	}

	var emailVerification = `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	regexEmail := regexp.MustCompile(emailVerification)
	if !regexEmail.MatchString(params.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email address this is your email:", "email": params.Email})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(params.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "this is to hash password"})
		return
	}
	params.Password = string(hashedPassword)

	user, err := uc.Queries.CreateUser(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "this is to work with db"})
		return
	}

	token, err := GenerateJWT(user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "this is to generate the token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User created successfully",
		"user":    user,
		"token":   token,
	})

}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login godoc
// @Summary Login a user
// @Description Authenticate a user with their username and password
// @Tags users
// @Accept json
// @Produce json
// @Param user body LoginRequest true "User Data"
// @Success 200 {object} gin.H "Token"
// @Failure 400 {object} gin.H "Bad Request"
// @Failure 401 {object} gin.H "Unauthorized"
// @Failure 500 {object} gin.H "Internal Server Error"
// @Router /users/login [post]
func (uc *UserController) Login(c *gin.Context) {
	userRequests.WithLabelValues("POST", "/login").Inc()
	var params LoginRequest
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := uc.Queries.GetUserByUsername(c.Request.Context(), params.Username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(params.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, err := GenerateJWT(user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})

}

// Logout godoc
// @Summary Logout a user
// @Description Invalidate the user's token
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {object} gin.H "Message"
// @Failure 400 {object} gin.H "Bad Request"
// @Router /users/logout [post]
func (uc *UserController) Logout(c *gin.Context) {
	userRequests.WithLabelValues("POST", "/logout").Inc()
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No token provided"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}

type ChangePasswordRequest struct {
	OldUserPassword string `json:"old_password"`
	NewUserPassword string `json:"new_password"`
}

// ChangePassword godoc
// @Summary Change a user's password
// @Description Update the user's password
// @Tags users
// @Accept json
// @Produce json
// @Param user body ChangePasswordRequest true "Password Data"
// @Success 200 {object} gin.H "Message"
// @Failure 400 {object} gin.H "Bad Request"
// @Failure 401 {object} gin.H "Unauthorized"
// @Failure 500 {object} gin.H "Internal Server Error"
// @Router /users/change-password [put]
func (uc *UserController) ChangePassword(c *gin.Context) {
	userRequests.WithLabelValues("PUT", "/change-password").Inc()

	usernameRaw, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	username, ok := usernameRaw.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid token data"})
		return
	}

	user, err := uc.Queries.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not retrieve user information"})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldUserPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Incorrect old password"})
		return
	}

	if req.NewUserPassword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password cannot be empty"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewUserPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process new password"})
		return
	}

	updateParams := db.UpdateUserPasswordParams{
		ID:       user.ID,
		Password: string(hashedPassword),
	}

	if _, err := uc.Queries.UpdateUserPassword(c.Request.Context(), updateParams); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// DeleteUser godoc
// @Summary Delete a user
// @Description Delete a user by their ID
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} db.User "Deleted User"
// @Failure 400 {object} gin.H "Bad Request"
// @Failure 500 {object} gin.H "Internal Server Error"
// @Router /users/{id} [delete]
func (uc *UserController) DeleteUser(c *gin.Context) {
	userRequests.WithLabelValues("DELETE", routeForSingleUser).Inc()
	id, err := strconv.Atoi(c.Param("id"))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := uc.Queries.DeleteUser(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

// GetUser godoc
// @Summary Get a user by ID
// @Description Retrieve a user by their ID
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} db.User "User Information"
// @Failure 400 {object} gin.H "Bad Request"
// @Failure 500 {object} gin.H "Internal Server Error"
// @Router /users/{id} [get]
func (uc *UserController) GetUser(c *gin.Context) {
	userRequests.WithLabelValues("GET", routeForSingleUser).Inc()
	id, err := strconv.Atoi(c.Param("id"))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// user, message, err := uc.CheckCache(c, id)
	// if err != nil {
	// 	uc.Logger.Error(err.Error())
	// 	c.JSON(http.StatusBadRequest, gin.H{"message": message})
	// }

	// userJson, err := json.Marshal(struct {
	// 	ID        int32  `json:"id"`
	// 	Username  string `json:"username"`
	// 	Email     string `json:"email"`
	// 	Password  string `json:"password"`
	// 	Age       int    `json:"age"`
	// 	CreatedAt string `json:"created_at"`
	// }{
	// 	ID:        user.ID,
	// 	Username:  user.Username,
	// 	Email:     user.Email,
	// 	Password:  user.Password,
	// 	Age:       int(user.Age.Int32),
	// 	CreatedAt: user.CreatedAt.Time.Format(time.RFC3339),
	// })

	// if err != nil {
	// 	uc.Logger.Warn("Failed To Marshal User", zap.Error(err))
	// }

	// err = uc.RedisClient.Set(c.Request.Context(), strconv.Itoa(id), userJson, 10*time.Minute).Err()
	// if err != nil {
	// 	uc.Logger.Warn("Failed To Store User In Redis", zap.Error(err))
	// }

	// c.JSON(http.StatusOK, gin.H{"message": message, "user": user})

	cachedUser, err := uc.RedisClient.Get(c.Request.Context(), strconv.Itoa(id)).Result()
	if err != nil {
		uc.Logger.Info("There is nothing in the redis yet or there is problem fetching data")
	}
	type CachedUser struct {
		ID        int       `json:"id"`
		Username  string    `json:"username"`
		Email     string    `json:"email"`
		Password  string    `json:"password"`
		Age       int       `json:"age"`
		CreatedAt time.Time `json:"created_at"`
	}

	var cached CachedUser
	if err := json.Unmarshal([]byte(cachedUser), &cached); err == nil {
		user := db.User{
			ID:        int32(cached.ID),
			Username:  cached.Username,
			Email:     cached.Email,
			Password:  cached.Password,
			Age:       pgtype.Int4{Int32: int32(cached.Age), Valid: cached.Age != 0},
			CreatedAt: pgtype.Timestamp{Time: cached.CreatedAt, Valid: true},
		}

		uc.Logger.Info("Returning user from cache", zap.Any("cachedUser", user))
		c.JSON(http.StatusOK, gin.H{"source": "cache", "user": user})
		return
	} else if cachedUser == "" {
		uc.Logger.Info("Cached user is empty so there is nothing to marshal", zap.Any("cached user", cachedUser))
	} else {
		uc.Logger.Error("There is a problem in unmarshalling", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user, err := uc.Queries.GetUser(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	userJson, err := json.Marshal(struct {
		ID        int32  `json:"id"`
		Username  string `json:"username"`
		Email     string `json:"email"`
		Password  string `json:"password"`
		Age       int    `json:"age"`
		CreatedAt string `json:"created_at"`
	}{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Password:  user.Password,
		Age:       int(user.Age.Int32),
		CreatedAt: user.CreatedAt.Time.Format(time.RFC3339),
	})

	if err != nil {
		uc.Logger.Warn("Failed to marshal user", zap.Error(err))
		return
	}

	err = uc.RedisClient.Set(c.Request.Context(), strconv.Itoa(id), userJson, 10*time.Minute).Err()
	if err != nil {
		uc.Logger.Warn("Failed to cache user", zap.Error(err))
	}

	c.JSON(http.StatusOK, gin.H{"source": "database", "user": user})
}

type UpdateUserRequest struct {
	Username *string `json:"name,omitempty"`
	Email    *string `json:"email,omitempty"`
	Age      *int32  `json:"age,omitempty"`
}

// UpdateUser godoc
// @Summary Update a user's information
// @Description Update the user's details such as username, email, and age
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param user body UpdateUserRequest true "Updated User Data"
// @Success 200 {object} db.User "Updated User"
// @Failure 400 {object} gin.H "Bad Request"
// @Failure 500 {object} gin.H "Internal Server Error"
// @Router /users/{id} [put]
func (uc *UserController) UpdateUser(c *gin.Context) {
	userRequests.WithLabelValues("PUT", routeForSingleUser).Inc()
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existingUser, err := uc.Queries.GetUser(c.Request.Context(), int32(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	updateParams := db.UpdateUserParams{
		ID:       int32(id),
		Username: ifNotNil(req.Username, existingUser.Username),
		Email:    ifNotNil(req.Email, existingUser.Email),
		Age:      ifNotNilInt(req.Age, existingUser.Age),
	}

	user, err := uc.Queries.UpdateUser(c.Request.Context(), updateParams)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	userWithNoPassword := db.User{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Age:      user.Age,
	}

	c.JSON(http.StatusOK, userWithNoPassword)
}

// GetUsers godoc
// @Summary Get all users
// @Description Retrieve all users from the database
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {array} db.User "List of Users"
// @Failure 500 {object} gin.H "Internal Server Error"
// @Router /users [get]
func (uc *UserController) GetUsers(c *gin.Context) {
	userRequests.WithLabelValues("GET", "/users").Inc()
	users, err := uc.Queries.GetUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

func ifNotNil[T any](value *T, defaultValue T) T {
	if value != nil {
		return *value
	}
	return defaultValue
}

func ifNotNilInt(value *int32, defaultValue pgtype.Int4) pgtype.Int4 {
	if value != nil {
		return pgtype.Int4{Int32: *value, Valid: true}
	}
	return defaultValue
}

// func (uc *UserController) CheckCache(c *gin.Context, userId int) (db.User, string, error) {
// 	cacheChan := make(chan db.User, 1)
// 	dbChan := make(chan db.User, 1)

// 	// Start goroutine to check cache
// 	go func() {
// 		cachedUser, err := uc.RedisClient.Get(c.Request.Context(), strconv.Itoa(userId)).Result()

// 		if err != nil {
// 			uc.Logger.Info("There Is No Data In Redis Or There Is A Problem Fetching Data")
// 		}
// 		type CachedUser struct {
// 			ID        int       `json:"id"`
// 			Username  string    `json:"username"`
// 			Email     string    `json:"email"`
// 			Password  string    `json:"password"`
// 			Age       int       `json:"age"`
// 			CreatedAt time.Time `json:"created_at"`
// 		}

// 		var cached CachedUser
// 		if err := json.Unmarshal([]byte(cachedUser), &cached); err == nil {
// 			user := db.User{
// 				ID:        int32(cached.ID),
// 				Username:  cached.Username,
// 				Email:     cached.Email,
// 				Password:  cached.Password,
// 				Age:       pgtype.Int4{Int32: int32(cached.Age), Valid: cached.Age != 0},
// 				CreatedAt: pgtype.Timestamp{Time: cached.CreatedAt, Valid: true},
// 			}
// 			uc.Logger.Info("this is the redis finding the user", zap.Any("user", user))
// 			cacheChan <- user
// 		} else {
// 			cacheChan <- db.User{}
// 		}
// 	}()

// 	// Start goroutine to get user from DB
// 	go func() {
// 		user, err := uc.Queries.GetUser(c.Request.Context(), int32(userId))
// 		uc.Logger.Info("this is the database finding the user", zap.Any("user", user))
// 		if err == nil {
// 			dbChan <- user
// 		} else {
// 			c.JSON(http.StatusInternalServerError, err.Error())
// 			uc.Logger.Error("There Is A Problem In Getting Data From Database")
// 			dbChan <- db.User{}
// 		}
// 	}()

// 	// Select which source returns first
// 	select {
// 	case cachedUser := <-cacheChan:
// 		if cachedUser.ID != 0 {
// 			return cachedUser, "source: cache", nil
// 		}
// 	case dbUser := <-dbChan:
// 		if dbUser.ID != 0 {
// 			return dbUser, "source: database", nil
// 		}
// 	}

// 	return db.User{}, "source: none", errors.New("user not found")
// }
