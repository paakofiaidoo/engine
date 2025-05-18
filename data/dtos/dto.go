package dtos

type PageDto struct {
	Body struct {
		Name        string `json:"name" docs:"page name"`
		Route       string `json:"route"`
		CustomTheme bool   `json:"custom_theme"`
	}
}

type Project struct {
	Name         string              `json:"name" docs:"project name"`
	Description  string              `json:"description" docs:"project description"`
	Framework    string              `json:"framework" docs:"project framework"`
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