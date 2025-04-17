package models

import (
	"encoding/json"
	"fmt"
)

// BaseComponent represents the common properties shared by all components
type BaseComponent struct {
	Type       string   `json:"type"`
	ClassNames []string `json:"classNames"`
}

// HeaderComponent represents the header component
type HeaderComponent struct {
	BaseComponent
	Title string `json:"title"`
}

// FooterComponent represents the footer component
type FooterComponent struct {
	BaseComponent
	CompanyName string `json:"companyName"`
}

// BlogPostComponent represents the blog post component
type BlogPostComponent struct {
	BaseComponent
	ImageURL    string `json:"imageUrl,omitempty"`
	Title       string `json:"title"`
	CreatedDate string `json:"createdDate"`
	Author      string `json:"author,omitempty"`
	ReadTime    string `json:"readTime,omitempty"`
	Content     string `json:"content"`
}

// Project represents the entire project structure
type Project struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	CSS         string      `json:"css,omitempty"`
	Components  []Component `json:"components"`
}

// Component is an interface that all components must implement
type Component interface {
	GetType() string
}

// GetType implements Component interface for BaseComponent
func (b BaseComponent) GetType() string {
	return b.Type
}

// UnmarshalJSON implements custom unmarshaling for Project
func (p *Project) UnmarshalJSON(data []byte) error {
	type Alias Project
	aux := &struct {
		Components []json.RawMessage `json:"components"`
		*Alias
	}{
		Alias: (*Alias)(p),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	p.Components = make([]Component, len(aux.Components))
	for i, raw := range aux.Components {
		var base BaseComponent
		if err := json.Unmarshal(raw, &base); err != nil {
			return err
		}

		switch base.Type {
		case "header":
			var header HeaderComponent
			if err := json.Unmarshal(raw, &header); err != nil {
				return err
			}
			p.Components[i] = header
		case "blogPost":
			var blogPost BlogPostComponent
			if err := json.Unmarshal(raw, &blogPost); err != nil {
				return err
			}
			p.Components[i] = blogPost
		case "footer":
			var footer FooterComponent
			if err := json.Unmarshal(raw, &footer); err != nil {
				return err
			}
			p.Components[i] = footer
		default:
			return fmt.Errorf("unknown component type: %s", base.Type)
		}
	}

	return nil
}
