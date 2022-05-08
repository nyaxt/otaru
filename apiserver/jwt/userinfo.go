package jwt

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type UserInfo struct {
	Role
	User string
}

func (ui UserInfo) String() string {
	return fmt.Sprintf("%s [role=%v]", ui.User, ui.Role)
}

var (
	AnonymousUserInfo = UserInfo{Role: RoleAnonymous, User: "anonymous"}
	NoauthUserInfo    = UserInfo{Role: RoleAdmin, User: "auth-disabled"}
)

type userInfoKey struct{}

func ContextWithUserInfo(ctx context.Context, ui UserInfo) context.Context {
	return context.WithValue(ctx, userInfoKey{}, ui)
}

func UserInfoFromContext(ctx context.Context) UserInfo {
	ui, ok := ctx.Value(userInfoKey{}).(UserInfo)
	if !ok {
		return AnonymousUserInfo
	}
	return ui
}

func RequireRoleGRPC(ctx context.Context, req Role) error {
	ui := UserInfoFromContext(ctx)
	if ui.Role < req {
		return grpc.Errorf(codes.PermissionDenied, "Action requires role %v, but you are %v", req, ui)
	}

	return nil
}
