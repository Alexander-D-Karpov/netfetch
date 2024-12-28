package cli

type Options struct {
	ExePath     string
	ConfigPath  string
	DefaultPort int
}

type Command interface {
	Execute() error
}
