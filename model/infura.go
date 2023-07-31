package model

// InfuraError represents an error returned by Infura
type InfuraError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		See  string `json:"see"`
		Rate struct {
			CurrentRps     float64 `json:"current_rps"`
			AllowedRps     float64 `json:"allowed_rps"`
			BackoffSeconds float64 `json:"backoff_seconds"`
		}
	} `json:"data"`
}
