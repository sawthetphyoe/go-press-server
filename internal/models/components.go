package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// Project represents the entire project structure
type Project struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Description  string           `json:"description"`
	GlobalConfig GlobalConfig     `json:"globalConfig"`
	Pages        []Page           `json:"pages"`
	Header       ComponentWrapper `json:"header"`
	Footer       ComponentWrapper `json:"footer"`
}

// Page represents a page in the CMS
type Page struct {
	ID         string             `json:"id"`
	Title      string             `json:"title"`
	Slug       string             `json:"slug"`
	Components []ComponentWrapper `json:"components"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

// Base Component Interface
type BaseComponent struct {
	Type       string             `json:"type"`
	ID         string             `json:"id"`
	ClassNames string             `json:"classNames,omitempty"`
	Children   []ComponentWrapper `json:"children,omitempty"`
	Content    string             `json:"content,omitempty"`
}

// Text Component
type TextComponent struct {
	BaseComponent
	Variant string `json:"variant"`
}

// Image Component
type ImageComponent struct {
	BaseComponent
	Src                 string `json:"src"`
	Alt                 string `json:"alt"`
	Width               string `json:"width,omitempty"`
	Height              string `json:"height,omitempty"`
	ContainerClassNames string `json:"containerClassNames,omitempty"`
	LabelClassNames     string `json:"labelClassNames,omitempty"`
	Caption             string `json:"caption,omitempty"`
	CaptionClassNames   string `json:"captionClassNames,omitempty"`
	Loading             string `json:"loading,omitempty"`
}

// Link Component
type LinkComponent struct {
	BaseComponent
	Href   string `json:"href"`
	Target string `json:"target,omitempty"`
	Rel    string `json:"rel,omitempty"`
	Title  string `json:"title,omitempty"`
}

// Block Component
type BlockComponent struct {
	BaseComponent
}

// Header Component
type HeaderComponent struct {
	BaseComponent
}

// Article Component
type ArticleComponent struct {
	BaseComponent
}

// Input Component
type InputComponent struct {
	BaseComponent
	Type                string `json:"type,omitempty"`
	Name                string `json:"name,omitempty"`
	Placeholder         string `json:"placeholder,omitempty"`
	Label               string `json:"label,omitempty"`
	Required            bool   `json:"required,omitempty"`
	Value               string `json:"value,omitempty"`
	MaxLength           int    `json:"maxLength,omitempty"`
	Min                 int    `json:"min,omitempty"`
	Max                 int    `json:"max,omitempty"`
	Pattern             string `json:"pattern,omitempty"`
	Disabled            bool   `json:"disabled,omitempty"`
	ContainerClassNames string `json:"containerClassNames,omitempty"`
	LabelClassNames     string `json:"labelClassNames,omitempty"`
}

// TextArea Component
type TextAreaComponent struct {
	BaseComponent
	Name                string `json:"name,omitempty"`
	Placeholder         string `json:"placeholder,omitempty"`
	Label               string `json:"label,omitempty"`
	Required            bool   `json:"required,omitempty"`
	Value               string `json:"value,omitempty"`
	Rows                int    `json:"rows,omitempty"`
	ContainerClassNames string `json:"containerClassNames,omitempty"`
	LabelClassNames     string `json:"labelClassNames,omitempty"`
}

type ButtonComponent struct {
	BaseComponent
	OnClick  string `json:"onClick,omitempty"`
	Disabled bool   `json:"disabled,omitempty"`
}

type FooterComponent struct {
	BaseComponent
}

// Component interface for all component types
type Component interface {
	GetType() string
	GetID() string
	GetClassNames() string
	GetChildren() []Component
	GetContent() string
}

// Implement Component interface for each type
func (c BaseComponent) GetType() string       { return c.Type }
func (c BaseComponent) GetID() string         { return c.ID }
func (c BaseComponent) GetClassNames() string { return c.ClassNames }
func (c BaseComponent) GetChildren() []Component {
	children := make([]Component, len(c.Children))
	for i, child := range c.Children {
		children[i] = child.Component
	}
	return children
}
func (c BaseComponent) GetContent() string { return c.Content }

// ComponentWrapper is a wrapper type for Component interface to implement custom unmarshaler
type ComponentWrapper struct {
	Component
}

// UnmarshalJSON implements json.Unmarshaler for ComponentWrapper
func (cw *ComponentWrapper) UnmarshalJSON(data []byte) error {
	var base BaseComponent
	if err := json.Unmarshal(data, &base); err != nil {
		return err
	}

	switch base.Type {
	case "text":
		var text TextComponent
		if err := json.Unmarshal(data, &text); err != nil {
			return err
		}
		cw.Component = &text
	case "image":
		var image ImageComponent
		if err := json.Unmarshal(data, &image); err != nil {
			return err
		}
		cw.Component = &image
	case "link":
		var link LinkComponent
		if err := json.Unmarshal(data, &link); err != nil {
			return err
		}
		cw.Component = &link
	case "block":
		var block BlockComponent
		if err := json.Unmarshal(data, &block); err != nil {
			return err
		}
		cw.Component = &block
	case "header":
		var header HeaderComponent
		if err := json.Unmarshal(data, &header); err != nil {
			return err
		}
		cw.Component = &header
	case "footer":
		var footer FooterComponent
		if err := json.Unmarshal(data, &footer); err != nil {
			return err
		}
		cw.Component = &footer
	case "article":
		var article ArticleComponent
		if err := json.Unmarshal(data, &article); err != nil {
			return err
		}
		cw.Component = &article
	case "input":
		var input InputComponent
		if err := json.Unmarshal(data, &input); err != nil {
			return err
		}
		cw.Component = &input
	case "textarea":
		var textarea TextAreaComponent
		if err := json.Unmarshal(data, &textarea); err != nil {
			return err
		}
		cw.Component = &textarea
	case "button":
		var button ButtonComponent
		if err := json.Unmarshal(data, &button); err != nil {
			return err
		}
		cw.Component = &button
	default:
		return fmt.Errorf("unknown component type: %s", base.Type)
	}

	return nil
}
