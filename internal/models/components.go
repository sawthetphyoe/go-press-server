package models

// Project represents the entire project structure
type Project struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	GlobalConfig GlobalConfig `json:"globalConfig"`
	Pages        []Page       `json:"pages"`
	BlogPosts    []BlogPost   `json:"blogPosts"`
}

// GlobalConfig represents the global theme configuration
type GlobalConfig struct {
	Theme Theme `json:"theme"`
}

// Theme represents the design system
type Theme struct {
	Colors     Colors     `json:"colors"`
	Typography Typography `json:"typography"`
	Spacing    Spacing    `json:"spacing"`
}

// Colors represents the color palette
type Colors struct {
	Primary    string `json:"primary"`
	Secondary  string `json:"secondary"`
	Background string `json:"background"`
	Text       string `json:"text"`
}

// Typography represents the typography system
type Typography struct {
	FontFamily string    `json:"fontFamily"`
	FontSizes  FontSizes `json:"fontSizes"`
}

// FontSizes represents the typography scale
type FontSizes struct {
	Small   string `json:"sm"`
	Base    string `json:"base"`
	Large   string `json:"lg"`
	XLarge  string `json:"xl"`
	XXLarge string `json:"2xl"`
}

// Spacing represents the spacing system
type Spacing struct {
	Small  string `json:"sm"`
	Medium string `json:"md"`
	Large  string `json:"lg"`
	XLarge string `json:"xl"`
}

// Page represents a page in the project
type Page struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Path       string      `json:"path"`
	Layout     string      `json:"layout"`
	Config     PageConfig  `json:"config"`
	Components []Component `json:"components"`
}

// PageConfig represents the configuration for a page
type PageConfig struct {
	Header ComponentConfig `json:"header"`
	Main   ComponentConfig `json:"main"`
	Footer ComponentConfig `json:"footer"`
}

// ComponentConfig represents the style configuration for a component
type ComponentConfig struct {
	ClassNames string `json:"classNames"`
}

// Component represents a component in the project
type Component struct {
	Type    string                 `json:"type"`
	Content map[string]interface{} `json:"content,omitempty"`
	Text    string                 `json:"text,omitempty"`
	URL     string                 `json:"url,omitempty"`
	Label   string                 `json:"label,omitempty"`
}

// BlogPost represents a blog post
type BlogPost struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	CreatedDate string         `json:"createdDate"`
	Author      string         `json:"author"`
	ReadTime    string         `json:"readTime"`
	Preview     string         `json:"preview"`
	Content     []ContentBlock `json:"content"`
}

// ContentBlock represents a block of content in a blog post
type ContentBlock struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	Label string `json:"label,omitempty"`
	URL   string `json:"url,omitempty"`
}
