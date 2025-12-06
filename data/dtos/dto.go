package dtos

type PageDto struct {
	ID          string `json:"id"`
	Name        string `json:"name" docs:"page name"`
	Route       string `json:"route"`
	Content     string `json:"content"`
	RawContent  string `json:"raw_content"`
	CustomTheme bool   `json:"custom_theme"`
}

type Project struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	Path         string              `json:"path"`
	Framework    string              `json:"framework"`
	Description  string              `json:"description"`
	GlobalCSS    string              `json:"global_css"`
	ApiKey       string              `json:"api_key"`
	Pages        []PageDto           `json:"pages"`
	NextJSConfig NextJSProjectConfig `json:"nextJSConfig" docs:"project next js config"`
}

type NextJSProjectConfig struct {
	UseTypeScript        bool
	UseESLint            bool
	UseTailwindCSS       bool
	UseSrcDirectory      bool
	UseAppRouter         bool
	UseTurbopack         bool
	CustomizeImportAlias bool
	ImportAlias          string
}
