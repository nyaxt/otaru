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
