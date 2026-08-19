package runmode

import "fmt"

type Mode string

const (
	Local       Mode = "local"
	LocalDocker Mode = "localDocker"
	Server      Mode = "server"
)

func Parse(raw string) (Mode, error) {
	m := Mode(raw)
	switch m {
	case Local, LocalDocker, Server:
		return m, nil
	default:
		return "", fmt.Errorf("invalid runMode %q (want local, localDocker, or server)", raw)
	}
}

func Default() Mode {
	return LocalDocker
}
