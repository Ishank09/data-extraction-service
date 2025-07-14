package server

type Config struct {
	Server struct {
		Port int64
	}
}

const (
	PortEnvVar = "PORT"
)
