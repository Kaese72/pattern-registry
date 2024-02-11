package client

type Config struct {
	BaseUrl string `mapstructure:"baseurl"`
}

func (config Config) GetClient() Client {
	// FIXME These will diverge eventuall
	return Client(config)
}
