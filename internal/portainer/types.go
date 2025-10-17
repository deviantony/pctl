package portainer

// Environment represents a Portainer environment/endpoint
type Environment struct {
	ID   int    `json:"Id"`
	Name string `json:"Name"`
	URL  string `json:"URL"`
}

// Stack represents a Portainer stack
type Stack struct {
	ID            int    `json:"Id"`
	Name          string `json:"Name"`
	StackFile     string `json:"StackFile"`
	EnvironmentID int    `json:"EndpointId"`
	Status        int    `json:"Status"`
}

// CreateStackRequest represents the request payload for creating a stack
type CreateStackRequest struct {
	Name             string   `json:"Name"`
	StackFileContent string   `json:"StackFileContent"`
	Env              []EnvVar `json:"Env"`
}

// EnvVar represents an environment variable
type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// UpdateStackRequest represents the request payload for updating a stack
type UpdateStackRequest struct {
	StackFile string `json:"StackFile"`
	PullImage bool   `json:"PullImage"`
	Prune     bool   `json:"Prune"`
}

// APIError represents an error response from the Portainer API
type APIError struct {
	Message string `json:"message"`
	Details string `json:"details"`
}
