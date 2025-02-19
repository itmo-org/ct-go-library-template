package config

type (
	Config struct {
		GRPC
	}

	GRPC struct {
		Port        string `env:"GRPC_PORT"`
		GatewayPort string `env:"GRPC_GATEWAY_PORT"`
	}
)

func NewConfig() (*Config, error) {
	cfg := &Config{}

	return cfg, nil
}
