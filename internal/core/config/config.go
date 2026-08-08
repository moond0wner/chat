package config

type Config struct {
	TCP_PORT       string
	LimitOnChannel int
}

func NewConfig() Config {
	return Config{
		TCP_PORT:       ":8080",
		LimitOnChannel: 10,
	}
}
