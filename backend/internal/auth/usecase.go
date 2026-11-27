package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/handokodenih/umkm_pos/backend/internal/model"
	pkghash "github.com/handokodenih/umkm_pos/backend/pkg/hash"
	pkgjwt "github.com/handokodenih/umkm_pos/backend/pkg/jwt"
)

var (
	ErrUserAlreadyExists = errors.New("username or email is already registered in this tenant")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserInactive       = errors.New("user account is currently inactive")
	ErrRoleNotFound       = errors.New("requested user role is not recognized")
	ErrTenantNotFound     = errors.New("specified business tenant was not found")
	ErrInvalidPIN         = errors.New("invalid 6-digit security PIN")
)

// Usecase defines business logic methods for authentication.
type Usecase interface {
	Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error)
	Login(ctx context.Context, req LoginRequest) (*AuthResponse, error)
	PinLogin(ctx context.Context, req PinLoginRequest) (*AuthResponse, error)
	GetMe(ctx context.Context, userID uint64) (*AuthResponse, error)
}

type usecase struct {
	repo       Repository
	jwtService *pkgjwt.Service
}

// NewUsecase instantiates an Auth usecase.
func NewUsecase(repo Repository, jwtService *pkgjwt.Service) Usecase {
	return &usecase{
		repo:       repo,
		jwtService: jwtService,
	}
}

func (u *usecase) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	// 1. Resolve Tenant
	var tenant *model.Tenant
	var err error

	if req.TenantCode != "" {
		tenant, err = u.repo.FindTenantByCode(ctx, req.TenantCode)
		if err != nil {
			// Auto create tenant if tenant_name was provided
			if req.TenantName != "" {
				tenant = &model.Tenant{
					Code:             req.TenantCode,
					Name:             req.TenantName,
					Status:           "ACTIVE",
					SubscriptionPlan: "STANDARD",
				}
				if err := u.repo.CreateTenant(ctx, tenant); err != nil {
					return nil, fmt.Errorf("failed to create tenant: %w", err)
				}
			} else {
				return nil, ErrTenantNotFound
			}
		}
	} else {
		// Single-tenant default bootstrap
		var defaultOutlet *model.Outlet
		tenant, defaultOutlet, err = u.repo.EnsureDefaultTenantAndOutlet(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to ensure default tenant: %w", err)
		}
		if len(req.OutletIDs) == 0 && defaultOutlet != nil {
			req.OutletIDs = []uint64{defaultOutlet.ID}
		}
	}

	// 2. Check if username already exists within tenant
	existingUser, _ := u.repo.FindUserByUsernameOrEmail(ctx, tenant.ID, req.Username)
	if existingUser != nil {
		return nil, ErrUserAlreadyExists
	}

	// 3. Resolve Role
	roleName := strings.ToUpper(strings.TrimSpace(req.RoleName))
	role, err := u.repo.FindRoleByName(ctx, &tenant.ID, roleName)
	if err != nil {
		return nil, ErrRoleNotFound
	}

	// 4. Hash Password
	hashedPassword, err := pkghash.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// 5. Hash PIN if provided
	var hashedPin string
	if req.Pin != "" {
		hashedPin = pkghash.HashPIN(req.Pin, req.Username)
	}

	// 6. Create User record
	newUser := &model.User{
		TenantID:     tenant.ID,
		RoleID:       role.ID,
		Username:     req.Username,
		Email:        req.Email,
		Phone:        req.Phone,
		PasswordHash: hashedPassword,
		FullName:     req.FullName,
		PinHash:      hashedPin,
		IsActive:     true,
	}

	if err := u.repo.CreateUser(ctx, newUser, req.OutletIDs); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// 7. Fetch permissions and outlets for response & JWT
	permissions, _ := u.repo.GetRolePermissions(ctx, role.ID)
	outlets, _ := u.repo.GetUserOutlets(ctx, newUser.ID)

	var outletIDs []uint64
	var outletSummaries []OutletSummaryDTO
	for _, o := range outlets {
		outletIDs = append(outletIDs, o.ID)
		outletSummaries = append(outletSummaries, OutletSummaryDTO{
			ID:           o.ID,
			Code:         o.Code,
			Name:         o.Name,
			IsMainBranch: o.IsMainBranch,
		})
	}

	// 8. Generate JWT Token Pair
	tokenPair, err := u.jwtService.GenerateTokenPair(
		newUser.ID,
		newUser.UUID,
		tenant.ID,
		tenant.UUID,
		newUser.Username,
		role.Name,
		permissions,
		outletIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth tokens: %w", err)
	}

	return &AuthResponse{
		Token: *tokenPair,
		User: UserSummaryDTO{
			ID:        newUser.ID,
			UUID:      newUser.UUID,
			TenantID:  newUser.TenantID,
			Username:  newUser.Username,
			FullName:  newUser.FullName,
			Email:     newUser.Email,
			Phone:     newUser.Phone,
			Role:      role.Name,
			IsActive:  newUser.IsActive,
			CreatedAt: newUser.CreatedAt,
		},
		Tenant: TenantSummaryDTO{
			ID:   tenant.ID,
			UUID: tenant.UUID,
			Code: tenant.Code,
			Name: tenant.Name,
		},
		Permissions: permissions,
		Outlets:     outletSummaries,
	}, nil
}

func (u *usecase) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	// 1. Resolve Tenant
	var tenant *model.Tenant
	var err error

	if req.TenantCode != "" {
		tenant, err = u.repo.FindTenantByCode(ctx, req.TenantCode)
		if err != nil {
			return nil, ErrTenantNotFound
		}
	} else {
		// Single-tenant fallback
		tenant, _, err = u.repo.EnsureDefaultTenantAndOutlet(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve tenant: %w", err)
		}
	}

	// 2. Find User by username or email
	user, err := u.repo.FindUserByUsernameOrEmail(ctx, tenant.ID, req.Username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// 3. Verify Account Status
	if !user.IsActive {
		return nil, ErrUserInactive
	}

	// 4. Verify Password
	if !pkghash.CheckPasswordHash(req.Password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	// 5. Fetch Role Permissions and Outlets
	permissions, _ := u.repo.GetRolePermissions(ctx, user.RoleID)
	outlets, _ := u.repo.GetUserOutlets(ctx, user.ID)

	var outletIDs []uint64
	var outletSummaries []OutletSummaryDTO
	for _, o := range outlets {
		outletIDs = append(outletIDs, o.ID)
		outletSummaries = append(outletSummaries, OutletSummaryDTO{
			ID:           o.ID,
			Code:         o.Code,
			Name:         o.Name,
			IsMainBranch: o.IsMainBranch,
		})
	}

	// 6. Generate JWT Token Pair
	tokenPair, err := u.jwtService.GenerateTokenPair(
		user.ID,
		user.UUID,
		tenant.ID,
		tenant.UUID,
		user.Username,
		user.Role.Name,
		permissions,
		outletIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth tokens: %w", err)
	}

	return &AuthResponse{
		Token: *tokenPair,
		User: UserSummaryDTO{
			ID:        user.ID,
			UUID:      user.UUID,
			TenantID:  user.TenantID,
			Username:  user.Username,
			FullName:  user.FullName,
			Email:     user.Email,
			Phone:     user.Phone,
			Role:      user.Role.Name,
			IsActive:  user.IsActive,
			CreatedAt: user.CreatedAt,
		},
		Tenant: TenantSummaryDTO{
			ID:   tenant.ID,
			UUID: tenant.UUID,
			Code: tenant.Code,
			Name: tenant.Name,
		},
		Permissions: permissions,
		Outlets:     outletSummaries,
	}, nil
}

func (u *usecase) PinLogin(ctx context.Context, req PinLoginRequest) (*AuthResponse, error) {
	// Single-tenant or resolved tenant
	tenant, _, err := u.repo.EnsureDefaultTenantAndOutlet(ctx)
	if err != nil {
		return nil, err
	}

	user, err := u.repo.FindUserByUsernameOrEmail(ctx, tenant.ID, req.Username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, ErrUserInactive
	}

	if user.PinHash == "" {
		return nil, errors.New("user does not have a quick PIN configured")
	}

	if !pkghash.CheckPINHash(req.Pin, user.Username, user.PinHash) {
		return nil, ErrInvalidPIN
	}

	permissions, _ := u.repo.GetRolePermissions(ctx, user.RoleID)
	outlets, _ := u.repo.GetUserOutlets(ctx, user.ID)

	var outletIDs []uint64
	var outletSummaries []OutletSummaryDTO
	for _, o := range outlets {
		outletIDs = append(outletIDs, o.ID)
		outletSummaries = append(outletSummaries, OutletSummaryDTO{
			ID:           o.ID,
			Code:         o.Code,
			Name:         o.Name,
			IsMainBranch: o.IsMainBranch,
		})
	}

	tokenPair, err := u.jwtService.GenerateTokenPair(
		user.ID,
		user.UUID,
		tenant.ID,
		tenant.UUID,
		user.Username,
		user.Role.Name,
		permissions,
		outletIDs,
	)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		Token: *tokenPair,
		User: UserSummaryDTO{
			ID:        user.ID,
			UUID:      user.UUID,
			TenantID:  user.TenantID,
			Username:  user.Username,
			FullName:  user.FullName,
			Email:     user.Email,
			Phone:     user.Phone,
			Role:      user.Role.Name,
			IsActive:  user.IsActive,
			CreatedAt: user.CreatedAt,
		},
		Tenant: TenantSummaryDTO{
			ID:   tenant.ID,
			UUID: tenant.UUID,
			Code: tenant.Code,
			Name: tenant.Name,
		},
		Permissions: permissions,
		Outlets:     outletSummaries,
	}, nil
}

func (u *usecase) GetMe(ctx context.Context, userID uint64) (*AuthResponse, error) {
	user, err := u.repo.FindUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	permissions, _ := u.repo.GetRolePermissions(ctx, user.RoleID)
	outlets, _ := u.repo.GetUserOutlets(ctx, user.ID)

	var outletIDs []uint64
	var outletSummaries []OutletSummaryDTO
	for _, o := range outlets {
		outletIDs = append(outletIDs, o.ID)
		outletSummaries = append(outletSummaries, OutletSummaryDTO{
			ID:           o.ID,
			Code:         o.Code,
			Name:         o.Name,
			IsMainBranch: o.IsMainBranch,
		})
	}

	return &AuthResponse{
		User: UserSummaryDTO{
			ID:        user.ID,
			UUID:      user.UUID,
			TenantID:  user.TenantID,
			Username:  user.Username,
			FullName:  user.FullName,
			Email:     user.Email,
			Phone:     user.Phone,
			Role:      user.Role.Name,
			IsActive:  user.IsActive,
			CreatedAt: user.CreatedAt,
		},
		Tenant: TenantSummaryDTO{
			ID:   user.Tenant.ID,
			UUID: user.Tenant.UUID,
			Code: user.Tenant.Code,
			Name: user.Tenant.Name,
		},
		Permissions: permissions,
		Outlets:     outletSummaries,
	}, nil
}
