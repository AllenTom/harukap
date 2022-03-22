package register

type RegisterConfig struct {
	Enable    bool     `json:"enable"`
	Endpoints []string `json:"endpoints"`
	RegPath   string   `json:"config"`
}
