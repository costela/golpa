package golpa

type Logger interface {
	Print(v ...interface{})
}

type noopLogger struct{}

func (noopLogger) Print(v ...interface{}) {}
