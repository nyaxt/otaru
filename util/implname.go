package util

type ImplNamed interface {
	ImplName() string
}

func TryGetImplName(v interface{}) string {
	named, ok := v.(ImplNamed)
	if !ok {
		return "<unknown>"
	}
	return named.ImplName()
}

func Describe(v interface{}) string {
	type stringifier interface {
		String() string
	}
	if sf, ok := v.(stringifier); ok {
		return sf.String()
	}
	return TryGetImplName(v)
}
