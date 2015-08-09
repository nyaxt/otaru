package logger

type Mux struct {
	Ls []Logger
}

func (m *Mux) Log(lv Level, data map[string]interface{}) {
	for _, l := range m.Ls {
		l.Log(lv, data)
	}
}

func (m *Mux) WillAccept(lv Level) bool {
	for _, l := range m.Ls {
		if l.WillAccept(lv) {
			return true
		}
	}
	return false
}
