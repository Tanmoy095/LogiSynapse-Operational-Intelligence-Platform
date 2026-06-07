package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
)

const authJSONCodecName = "json"

type authJSONCodec struct{}

func init() {
	encoding.RegisterCodec(authJSONCodec{})
}

func (authJSONCodec) Name() string {
	return authJSONCodecName
}

func (authJSONCodec) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (authJSONCodec) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

type AuthClient struct {
	conn *grpc.ClientConn
}

type ValidateAccessTokenResponse struct {
	UserID       string `json:"userId"`
	Email        string `json:"email"`
	IsSuperAdmin bool   `json:"isSuperAdmin"`
	Role         string `json:"role"`
	TenantID     string `json:"tenantId"`
	Allowed      bool   `json:"allowed"`
}

func NewAuthClient(addr string) (*AuthClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(
		ctx,
		addr,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(authJSONCodec{})),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to authentication service: %v", err)
	}
	return &AuthClient{conn: conn}, nil
}

func (c *AuthClient) Close() error {
	return c.conn.Close()
}

func (c *AuthClient) ValidateAccessToken(ctx context.Context, bearerToken, tenantID string) (*ValidateAccessTokenResponse, error) {
	token := strings.TrimSpace(strings.TrimPrefix(bearerToken, "Bearer "))
	var resp ValidateAccessTokenResponse
	err := c.conn.Invoke(ctx, "/auth.v1.AuthService/ValidateAccessToken", map[string]string{
		"accessToken": token,
		"tenantId":    tenantID,
	}, &resp, grpc.ForceCodec(authJSONCodec{}))
	if err != nil {
		return nil, handleGRPCError(err, "authentication")
	}
	return &resp, nil
}
