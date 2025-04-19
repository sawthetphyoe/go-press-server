package models

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
