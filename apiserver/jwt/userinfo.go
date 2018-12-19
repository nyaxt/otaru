package jwt

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type userInfoKey struct{}

type UserInfo struct {
	Role
	User string
}

func NewUserInfo(rolestr, user string) (*UserInfo, error) {
	role, ok := strToRole[rolestr]
	if !ok {
		return nil, fmt.Errorf("Invalid role %q.", rolestr)
	}

	return &UserInfo{Role: role, User: user}, nil
}

var AnonymousUserInfo *UserInfo
var NoauthUserInfo *UserInfo

func init() {
	var err error
	AnonymousUserInfo, err = NewUserInfo("anonymous", "anonymous")
	if err != nil {
		panic("AnonymousUserInfo")
	}
	NoauthUserInfo, err = NewUserInfo("admin", "auth-disabled")
	if err != nil {
		panic("NoauthUserInfo")
	}
}

func ContextWithUserInfo(ctx context.Context, ui *UserInfo) context.Context {
	return context.WithValue(ctx, userInfoKey{}, ui)
}

func UserInfoFromContext(ctx context.Context) *UserInfo {
	ui, ok := ctx.Value(userInfoKey{}).(*UserInfo)
	if !ok {
		return AnonymousUserInfo
	}
	return ui
}

func RequireRoleGRPC(ctx context.Context, req Role) error {
	ui := UserInfoFromContext(ctx)
	if ui.Role < req {
		return grpc.Errorf(codes.PermissionDenied, "")
	}

	return nil
}
