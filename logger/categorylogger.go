package logger

type CategoryLogger struct {
	BE       Logger
	Category string
	Level
}

func (l *CategoryLogger) Log(lv Level, data map[string]interface{}) {
	if !l.WillAccept(lv) {
		return
	}
	data["category"] = l.Category
	l.BE.Log(lv, data)
}

func (l *CategoryLogger) WillAccept(lv Level) bool {
	return lv >= l.Level
}
