package jwt

type Role int

const (
	RoleAnonymous Role = iota
	RoleReadOnly
	RoleAdmin
)

var strToRole = map[string]Role{
	"anonymous": RoleAnonymous,
	"readonly":  RoleReadOnly,
	"admin":     RoleAdmin,
}

var roleToStr = map[Role]string{
	RoleAnonymous: "anonymous",
	RoleReadOnly:  "readonly",
	RoleAdmin:     "admin",
}

func IsValidRoleStr(s string) bool {
	_, ok := strToRole[s]
	return ok
}

func RoleFromStr(s string) Role {
	if r, ok := strToRole[s]; ok {
		return r
	}
	return RoleAnonymous
}

func (r Role) String() string {
	return roleToStr[r]
}
