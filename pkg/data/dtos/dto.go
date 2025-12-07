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
	Port         int                 `json:"port"`
	Pages        []PageDto           `json:"pages"`
	RootRoute    RouteNode           `json:"root_route"`
	NextJSConfig NextJSProjectConfig `json:"nextJSConfig" docs:"project next js config"`
}

type RouteNode struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Segment  string      `json:"segment"`
	FullPath string      `json:"full_path"`
	Type     string      `json:"type"` // STATIC, DYNAMIC
	PageID   string      `json:"page_id,omitempty"`
	LayoutID string      `json:"layout_id,omitempty"`
	Children []RouteNode `json:"children"`
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
