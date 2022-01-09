package golpa

type Option func(*Model) error

func WithLogger(logger Logger) Option {
	return func(m *Model) error {
		m.logger = logger

		return nil
	}
}
