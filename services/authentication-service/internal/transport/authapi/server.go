package authapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	appauth "github.com/Tanmoy095/LogiSynapse/services/authentication-service/internal/app/auth"
	"github.com/Tanmoy095/LogiSynapse/services/authentication-service/internal/app/commands"
	domainErr "github.com/Tanmoy095/LogiSynapse/services/authentication-service/internal/domain/errors"
	"github.com/Tanmoy095/LogiSynapse/services/authentication-service/internal/domain/membership"
	"github.com/Tanmoy095/LogiSynapse/services/authentication-service/internal/domain/policy"
	"github.com/Tanmoy095/LogiSynapse/services/authentication-service/internal/domain/session"
	"github.com/Tanmoy095/LogiSynapse/services/authentication-service/internal/ports/crypto"
	"github.com/Tanmoy095/LogiSynapse/services/authentication-service/internal/ports/repository"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Server struct {
	register       *commands.RegisterUserHandler
	login          *commands.LoginUserHandler
	createTenant   *commands.CreateTenantCmdByPlatform
	addMembership  *commands.AddMembershipCmd
	tokenVerifier  *crypto.HMACTokenSigner
	userRepo       repository.UserStore
	tokenRepo      repository.RefreshTokenStore
	tenantRepo     repository.TenantStore
	membershipRepo repository.MemberShipStore
}

func NewServer(
	register *commands.RegisterUserHandler,
	login *commands.LoginUserHandler,
	createTenant *commands.CreateTenantCmdByPlatform,
	addMembership *commands.AddMembershipCmd,
	tokenVerifier *crypto.HMACTokenSigner,
	userRepo repository.UserStore,
	tokenRepo repository.RefreshTokenStore,
	tenantRepo repository.TenantStore,
	membershipRepo repository.MemberShipStore,
) *Server {
	return &Server{register: register, login: login, createTenant: createTenant, addMembership: addMembership, tokenVerifier: tokenVerifier, userRepo: userRepo, tokenRepo: tokenRepo, tenantRepo: tenantRepo, membershipRepo: membershipRepo}
}

func RegisterGRPCServer(s grpc.ServiceRegistrar, srv *Server) {
	s.RegisterService(&grpc.ServiceDesc{
		ServiceName: "auth.v1.AuthService",
		HandlerType: (*AuthServiceServer)(nil),
		Methods: []grpc.MethodDesc{
			{MethodName: "RegisterUser", Handler: unaryHandler(srv.RegisterUser)},
			{MethodName: "LoginUser", Handler: unaryHandler(srv.LoginUser)},
			{MethodName: "RefreshSession", Handler: unaryHandler(srv.RefreshSession)},
			{MethodName: "LogoutUser", Handler: unaryHandler(srv.LogoutUser)},
			{MethodName: "CreateTenant", Handler: unaryHandler(srv.CreateTenant)},
			{MethodName: "InviteMember", Handler: unaryHandler(srv.InviteMember)},
			{MethodName: "ValidateAccessToken", Handler: unaryHandler(srv.ValidateAccessToken)},
			{MethodName: "Health", Handler: unaryHandler(srv.Health)},
		},
	}, srv)
}

type AuthServiceServer interface {
	RegisterUser(context.Context, *RegisterUserRequest) (*RegisterUserResponse, error)
	LoginUser(context.Context, *LoginUserRequest) (*LoginUserResponse, error)
	RefreshSession(context.Context, *RefreshSessionRequest) (*LoginUserResponse, error)
	LogoutUser(context.Context, *LogoutUserRequest) (*LogoutUserResponse, error)
	CreateTenant(context.Context, *CreateTenantRequest) (*CreateTenantResponse, error)
	InviteMember(context.Context, *InviteMemberRequest) (*InviteMemberResponse, error)
	ValidateAccessToken(context.Context, *ValidateAccessTokenRequest) (*ValidateAccessTokenResponse, error)
	Health(context.Context, *HealthRequest) (*HealthResponse, error)
}

func (s *Server) RegisterUser(ctx context.Context, req *RegisterUserRequest) (*RegisterUserResponse, error) {
	userID, err := s.register.Handle(ctx, commands.RegisterUserParams{
		Email: strings.ToLower(strings.TrimSpace(req.Email)), Password: req.Password, FirstName: strings.TrimSpace(req.FirstName), LastName: strings.TrimSpace(req.LastName),
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &RegisterUserResponse{UserID: userID.String()}, nil
}

func (s *Server) LoginUser(ctx context.Context, req *LoginUserRequest) (*LoginUserResponse, error) {
	result, err := s.login.Handler(ctx, commands.LoginParams{
		Email: strings.ToLower(strings.TrimSpace(req.Email)), Password: req.Password, DeviceFingerprint: req.DeviceFingerprint,
	})
	if err != nil {
		return nil, appauth.MapLoginError(err)
	}
	return &LoginUserResponse{AccessToken: result.AccessToken, RefreshToken: result.RefreshToken, ExpiresIn: result.ExpiresIn, TokenType: result.TokenType}, nil
}

func (s *Server) RefreshSession(ctx context.Context, req *RefreshSessionRequest) (*LoginUserResponse, error) {
	oldHash := refreshHash(req.RefreshToken)
	oldToken, err := s.tokenRepo.GetTokenByHash(ctx, oldHash)
	if err != nil || oldToken.RevokedAt != nil || oldToken.ExpiresAt.Before(time.Now().UTC()) {
		return nil, status.Error(codes.Unauthenticated, "invalid refresh token")
	}
	if oldToken.ReplacedBy != nil {
		_ = s.tokenRepo.RevokeTokenFamily(ctx, oldToken.FamilyID)
		return nil, status.Error(codes.Unauthenticated, "invalid refresh token")
	}
	u, err := s.userRepo.GetUserByID(ctx, oldToken.UserID)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid refresh token")
	}
	accessToken, expiresIn, err := s.tokenVerifier.SignAccessToken(ctx, crypto.AccessClaims{UserID: u.UserID, UserEmail: u.UserEmail, IsSuperAdmin: u.IsSuperAdmin})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to sign access token")
	}
	rawRefresh, err := randomToken()
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create refresh token")
	}
	newToken := &session.RefreshToken{
		TokenID: uuid.New(), UserID: oldToken.UserID, TenantID: oldToken.TenantID, TokenHash: refreshHash(rawRefresh), FamilyID: oldToken.FamilyID,
		IssuedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(30 * 24 * time.Hour), DeviceMetadata: req.DeviceFingerprint,
	}
	if err := s.tokenRepo.RotateToken(ctx, oldToken.TokenID, newToken); err != nil {
		return nil, status.Error(codes.Internal, "failed to rotate refresh token")
	}
	return &LoginUserResponse{AccessToken: accessToken, RefreshToken: rawRefresh, ExpiresIn: int64(expiresIn.Seconds()), TokenType: "Bearer"}, nil
}

func (s *Server) LogoutUser(ctx context.Context, req *LogoutUserRequest) (*LogoutUserResponse, error) {
	token, err := s.tokenRepo.GetTokenByHash(ctx, refreshHash(req.RefreshToken))
	if err != nil {
		return &LogoutUserResponse{Success: true}, nil
	}
	_ = s.tokenRepo.RevokeTokenFamily(ctx, token.FamilyID)
	return &LogoutUserResponse{Success: true}, nil
}

func (s *Server) CreateTenant(ctx context.Context, req *CreateTenantRequest) (*CreateTenantResponse, error) {
	actor, err := claimsFromContext(ctx, s.tokenVerifier)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid bearer token")
	}
	ownerID, err := uuid.Parse(req.OwnerUserID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid owner user id")
	}
	tenantID, err := s.createTenant.Handle(ctx, commands.CreateTenantParams{
		TenantName: strings.TrimSpace(req.Name), ActorUserID: actor.UserID, IsActorSuperAdmin: actor.IsSuperAdmin, OwnerUserID: ownerID,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &CreateTenantResponse{TenantID: tenantID.String()}, nil
}

func (s *Server) InviteMember(ctx context.Context, req *InviteMemberRequest) (*InviteMemberResponse, error) {
	actor, err := claimsFromContext(ctx, s.tokenVerifier)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid bearer token")
	}
	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant id")
	}
	role := membership.Role(strings.ToLower(strings.TrimSpace(req.Role)))
	if role != membership.RoleAdmin && role != membership.RoleMember {
		return nil, status.Error(codes.InvalidArgument, "role must be admin or member")
	}
	err = s.addMembership.Handle(ctx, commands.AddMembershipParams{
		TenantID: tenantID, ActorUserID: actor.UserID, TargetUserEmail: strings.ToLower(strings.TrimSpace(req.Email)), Role: role,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &InviteMemberResponse{Success: true}, nil
}

func (s *Server) ValidateAccessToken(ctx context.Context, req *ValidateAccessTokenRequest) (*ValidateAccessTokenResponse, error) {
	claims, err := s.tokenVerifier.VerifyAccessToken(req.AccessToken)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid access token")
	}
	resp := &ValidateAccessTokenResponse{UserID: claims.UserID.String(), Email: claims.UserEmail, IsSuperAdmin: claims.IsSuperAdmin, Role: claims.Role, Allowed: true}
	if strings.TrimSpace(req.TenantID) == "" || claims.IsSuperAdmin {
		resp.TenantID = strings.TrimSpace(req.TenantID)
		return resp, nil
	}
	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant id")
	}
	t, err := s.tenantRepo.GetTenantByID(ctx, tenantID)
	if err != nil {
		return nil, mapError(err)
	}
	member, _ := s.membershipRepo.GetMember(ctx, claims.UserID, tenantID)
	role := policy.EffectiveRole(t.OwnerUserID, claims.UserID, member)
	if role == membership.RoleNone {
		return nil, status.Error(codes.PermissionDenied, "not a member of tenant")
	}
	resp.Role = string(role)
	resp.TenantID = tenantID.String()
	return resp, nil
}

func (s *Server) Health(ctx context.Context, req *HealthRequest) (*HealthResponse, error) {
	return &HealthResponse{Status: "OK"}, nil
}

func claimsFromContext(ctx context.Context, verifier *crypto.HMACTokenSigner) (*crypto.JWTClaims, error) {
	md, ok := grpcHeader(ctx, "authorization")
	if !ok {
		return nil, errors.New("missing authorization")
	}
	token := strings.TrimSpace(strings.TrimPrefix(md, "Bearer "))
	return verifier.VerifyAccessToken(token)
}

func grpcHeader(ctx context.Context, key string) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}
	values := md[key]
	if len(values) == 0 {
		return "", false
	}
	return values[0], true
}

func unaryHandler[Req any, Resp any](fn func(context.Context, *Req) (*Resp, error)) grpc.MethodHandler {
	return func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
		req := new(Req)
		if err := dec(req); err != nil {
			return nil, err
		}
		if interceptor == nil {
			return fn(ctx, req)
		}
		info := &grpc.UnaryServerInfo{Server: srv}
		return interceptor(ctx, req, info, func(ctx context.Context, request any) (any, error) {
			return fn(ctx, request.(*Req))
		})
	}
}

func mapError(err error) error {
	switch {
	case errors.Is(err, domainErr.ErrEmailAlreadyExists), errors.Is(err, domainErr.ErrDuplicateMembership):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domainErr.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domainErr.ErrUnauthorized), errors.Is(err, domainErr.ErrNotTenantAdmin), errors.Is(err, domainErr.ErrNotTenantOwner), errors.Is(err, domainErr.ErrUnauthorizedAction):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domainErr.ErrUserNotFound), errors.Is(err, domainErr.ErrTenantNotFound), errors.Is(err, domainErr.ErrMembershipNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}

func refreshHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
