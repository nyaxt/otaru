package logger

type CategoryLogger struct {
	BE       Logger
	Category string
}

func (l *CategoryLogger) Log(lv Level, data map[string]interface{}) {
	data["category"] = l.Category
	l.BE.Log(lv, data)
}

func (l *CategoryLogger) WillAccept(lv Level) bool { return true }
