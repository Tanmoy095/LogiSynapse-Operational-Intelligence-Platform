package authapi

type RegisterUserRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

type RegisterUserResponse struct {
	UserID string `json:"userId"`
}

type LoginUserRequest struct {
	Email             string `json:"email"`
	Password          string `json:"password"`
	DeviceFingerprint string `json:"deviceFingerprint"`
}

type LoginUserResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int64  `json:"expiresIn"`
	TokenType    string `json:"tokenType"`
}

type RefreshSessionRequest struct {
	RefreshToken      string `json:"refreshToken"`
	DeviceFingerprint string `json:"deviceFingerprint"`
}

type LogoutUserRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type LogoutUserResponse struct {
	Success bool `json:"success"`
}

type CreateTenantRequest struct {
	Name        string `json:"name"`
	OwnerUserID string `json:"ownerUserId"`
}

type CreateTenantResponse struct {
	TenantID string `json:"tenantId"`
}

type InviteMemberRequest struct {
	Email    string `json:"email"`
	TenantID string `json:"tenantId"`
	Role     string `json:"role"`
}

type InviteMemberResponse struct {
	Success bool `json:"success"`
}

type ValidateAccessTokenRequest struct {
	AccessToken string `json:"accessToken"`
	TenantID    string `json:"tenantId"`
}

type ValidateAccessTokenResponse struct {
	UserID       string `json:"userId"`
	Email        string `json:"email"`
	IsSuperAdmin bool   `json:"isSuperAdmin"`
	Role         string `json:"role"`
	TenantID     string `json:"tenantId"`
	Allowed      bool   `json:"allowed"`
}

type HealthRequest struct{}

type HealthResponse struct {
	Status string `json:"status"`
}
