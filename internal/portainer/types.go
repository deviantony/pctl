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
	StackFile     string `json:"EntryPoint"` // Portainer API uses EntryPoint, not StackFile
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

// StackDetails represents detailed stack information from Portainer
type StackDetails struct {
	ID            int    `json:"Id"`
	Name          string `json:"Name"`
	Status        int    `json:"Status"`
	EnvironmentID int    `json:"EndpointId"`
	CreatedAt     int64  `json:"creationDate"`
	UpdatedAt     int64  `json:"updateDate"`
	CreatedBy     string `json:"createdBy"`
	UpdatedBy     string `json:"updatedBy"`
	ProjectPath   string `json:"projectPath"`
	EntryPoint    string `json:"EntryPoint"`
}

// Container represents a Docker container
type Container struct {
	ID      string            `json:"Id"`
	Names   []string          `json:"Names"`
	Image   string            `json:"Image"`
	Status  string            `json:"Status"`
	State   string            `json:"State"`
	Created int64             `json:"Created"`
	Labels  map[string]string `json:"Labels"`
	Ports   []Port            `json:"Ports"`
}

// Port represents container port mapping
type Port struct {
	PrivatePort int    `json:"PrivatePort"`
	PublicPort  int    `json:"PublicPort"`
	Type        string `json:"Type"`
	IP          string `json:"IP"`
}

// APIError represents an error response from the Portainer API
type APIError struct {
	Message string `json:"message"`
	Details string `json:"details"`
}
