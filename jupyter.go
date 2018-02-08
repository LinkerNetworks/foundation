package config

type JupyterConfig struct {
	BaseURL string `json:"baseUrl"`

	// the address that the jupyter notebook will bind to "0.0.0.0"
	Address string `json:"bind"`

	// the default jupyternotebook docker image name
	DefaultImage string `json:"defaultImage"`

	// the working dir of the jupyter notebook process
	WorkingDir string `json:"workingDir"`

	// the cache configuration
	Cache *JupyterCacheConfig `json:"cache"`

	// proxy configuration that will be used for development mode.
	Dev *DevProxyConfig `json:"dev"`
}

type JupyterCacheConfig struct {
	Prefix string `json:"prefix"`
	Age    int    `json:"age"`
}

type DevProxyConfig struct {
	BaseURL     string `json:"baseUrl"`
	HostAddress string `json:"hostUrl"`
}
