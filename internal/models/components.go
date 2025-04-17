package models

// Project represents the entire project structure
type Project struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Config      Config      `json:"config"`
	Components  []Component `json:"components"`
}

// Config represents the configuration for a project
type Config struct {
	Header   ComponentConfig `json:"header"`
	Main     ComponentConfig `json:"main"`
	BlogPost ComponentConfig `json:"blogPost"`
	Footer   ComponentConfig `json:"footer"`
}

// ComponentConfig represents the style configuration for a component
type ComponentConfig struct {
	ClassNames string `json:"classNames"`
}

// Component represents a component in the project
type Component struct {
	Type    string                 `json:"type"`
	Content map[string]interface{} `json:"content"`
}
