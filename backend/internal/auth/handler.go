package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/handokodenih/umkm_pos/backend/internal/middleware"
	"github.com/handokodenih/umkm_pos/backend/pkg/response"
)

// Handler handles HTTP requests for the Auth domain.
type Handler struct {
	usecase Usecase
}

// NewHandler instantiates an Auth HTTP handler.
func NewHandler(usecase Usecase) *Handler {
	return &Handler{usecase: usecase}
}

// RegisterRoutes registers auth endpoints into the Gin router group.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	authGroup := r.Group("/auth")
	{
		authGroup.POST("/register", h.Register)
		authGroup.POST("/login", h.Login)
		authGroup.POST("/login/pin", h.PinLogin)

		// Protected endpoints
		protected := authGroup.Group("")
		protected.Use(authMiddleware)
		{
			protected.GET("/me", h.GetMe)
		}
	}
}

// Register creates a new user account with role assignment.
// @Summary User registration
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Register Payload"
// @Success 201 {object} response.Response
// @Failure 400 {object} response.Response
// @Router /api/v1/auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid registration input data", err.Error())
		return
	}

	result, err := h.usecase.Register(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrUserAlreadyExists) || errors.Is(err, ErrRoleNotFound) || errors.Is(err, ErrTenantNotFound) {
			response.BadRequest(c, err.Error(), nil)
			return
		}
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, http.StatusCreated, "User registered successfully", result)
}

// Login authenticates a user with username/email and password.
// @Summary User login
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login Payload"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Username and password are required", err.Error())
		return
	}

	result, err := h.usecase.Login(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			response.Unauthorized(c, "Invalid username or password")
			return
		}
		if errors.Is(err, ErrUserInactive) {
			response.Forbidden(c, "Account is inactive, please contact administrator")
			return
		}
		if errors.Is(err, ErrTenantNotFound) {
			response.BadRequest(c, "Business tenant code is invalid", nil)
			return
		}
		response.InternalServerError(c, "Login processing failed")
		return
	}

	response.Success(c, http.StatusOK, "Login successful", result)
}

// PinLogin authenticates a user using a 6-digit PIN for quick cashier switching.
// @Summary Quick PIN Login
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body PinLoginRequest true "PIN Login Payload"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/auth/login/pin [post]
func (h *Handler) PinLogin(c *gin.Context) {
	var req PinLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Username and 6-digit PIN are required", err.Error())
		return
	}

	result, err := h.usecase.PinLogin(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidPIN) || errors.Is(err, ErrInvalidCredentials) {
			response.Unauthorized(c, "Invalid username or security PIN")
			return
		}
		if errors.Is(err, ErrUserInactive) {
			response.Forbidden(c, "Account is inactive")
			return
		}
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Quick PIN login successful", result)
}

// GetMe returns identity and permissions for the currently authenticated user.
// @Summary Get Current Profile
// @Tags Auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/auth/me [get]
func (h *Handler) GetMe(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	result, err := h.usecase.GetMe(c.Request.Context(), userID)
	if err != nil {
		response.InternalServerError(c, "Failed to retrieve user profile")
		return
	}

	response.Success(c, http.StatusOK, "User profile retrieved successfully", result)
}
