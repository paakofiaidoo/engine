package api

import (
	"context"
	"strings"

	"juki-engine/pkg/data/models"
	enginev1 "juki-engine/pkg/gen/juki/engine/v1"
	"juki-engine/pkg/services"
	"juki-engine/pkg/services/marketplace"

	"connectrpc.com/connect"
)

// MarketplaceServer implements MarketplaceServiceHandler — the GitHub-as-DB
// registry for templates, components, plugins and icon packs.
type MarketplaceServer struct {
	registry *marketplace.Service
	service  services.Service
}

func NewMarketplaceServer(registry *marketplace.Service, service services.Service) *MarketplaceServer {
	return &MarketplaceServer{registry: registry, service: service}
}

var entryTypeToString = map[enginev1.EntryType]string{
	enginev1.EntryType_ENTRY_TYPE_UNSPECIFIED: "",
	enginev1.EntryType_ENTRY_TYPE_TEMPLATE:    "template",
	enginev1.EntryType_ENTRY_TYPE_COMPONENT:   "component",
	enginev1.EntryType_ENTRY_TYPE_PLUGIN:      "plugin",
	enginev1.EntryType_ENTRY_TYPE_ICON_PACK:   "icon_pack",
}

var stringToEntryType = map[string]enginev1.EntryType{
	"template":  enginev1.EntryType_ENTRY_TYPE_TEMPLATE,
	"component": enginev1.EntryType_ENTRY_TYPE_COMPONENT,
	"plugin":    enginev1.EntryType_ENTRY_TYPE_PLUGIN,
	"icon_pack": enginev1.EntryType_ENTRY_TYPE_ICON_PACK,
}

var stringToInstallType = map[string]enginev1.InstallType{
	"copy":  enginev1.InstallType_INSTALL_TYPE_COPY,
	"npm":   enginev1.InstallType_INSTALL_TYPE_NPM,
	"patch": enginev1.InstallType_INSTALL_TYPE_PATCH,
}

func entryToProto(e marketplace.Entry) *enginev1.MarketplaceEntry {
	return &enginev1.MarketplaceEntry{
		Id:          e.ID,
		Name:        e.Name,
		Description: e.Description,
		Version:     e.Version,
		Type:        stringToEntryType[e.Type],
		InstallType: stringToInstallType[e.InstallType],
		Source:      e.Source,
		Package:     e.Package,
		PreviewUrl:  e.Preview,
		Tags:        e.Tags,
	}
}

func (s *MarketplaceServer) RefreshRegistry(
	ctx context.Context,
	req *connect.Request[enginev1.RefreshRegistryRequest],
) (*connect.Response[enginev1.RefreshRegistryResponse], error) {
	count, fetchedAt, fromCache, err := s.registry.Refresh(req.Msg.Force)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}

	return connect.NewResponse(&enginev1.RefreshRegistryResponse{
		EntryCount:  int32(count),
		FetchedAtMs: fetchedAt,
		FromCache:   fromCache,
	}), nil
}

func (s *MarketplaceServer) ListEntries(
	ctx context.Context,
	req *connect.Request[enginev1.ListEntriesRequest],
) (*connect.Response[enginev1.ListEntriesResponse], error) {
	typeFilter := strings.ToLower(entryTypeToString[req.Msg.Type])

	entries := s.registry.List(typeFilter, req.Msg.Query)

	out := make([]*enginev1.MarketplaceEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, entryToProto(e))
	}

	return connect.NewResponse(&enginev1.ListEntriesResponse{Entries: out}), nil
}

func (s *MarketplaceServer) GetEntry(
	ctx context.Context,
	req *connect.Request[enginev1.GetEntryRequest],
) (*connect.Response[enginev1.GetEntryResponse], error) {
	entry, found := s.registry.Get(req.Msg.Id)
	if !found {
		return connect.NewResponse(&enginev1.GetEntryResponse{Found: false}), nil
	}

	return connect.NewResponse(&enginev1.GetEntryResponse{
		Found: true,
		Entry: entryToProto(*entry),
	}), nil
}

func (s *MarketplaceServer) InstallEntry(
	ctx context.Context,
	req *connect.Request[enginev1.InstallEntryRequest],
) (*connect.Response[enginev1.InstallEntryResponse], error) {
	entry, found := s.registry.Get(req.Msg.EntryId)
	if !found {
		return connect.NewResponse(&enginev1.InstallEntryResponse{
			Success: false,
			Message: "entry not found in registry: " + req.Msg.EntryId,
		}), nil
	}

	ref, err := s.service.InstallMarketplaceEntry(req.Msg.ProjectId, entry)
	if err != nil {
		return connect.NewResponse(&enginev1.InstallEntryResponse{
			Success: false,
			Message: err.Error(),
		}), nil
	}

	return connect.NewResponse(&enginev1.InstallEntryResponse{
		Success:      true,
		Message:      "installed " + entry.Name,
		InstalledRef: ref,
	}), nil
}

func (s *MarketplaceServer) ListInstalled(
	ctx context.Context,
	req *connect.Request[enginev1.ListInstalledRequest],
) (*connect.Response[enginev1.ListInstalledResponse], error) {
	installs, err := s.service.ListInstalledMarketplaceEntries(req.Msg.ProjectId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := make([]*enginev1.InstalledEntry, 0, len(installs))
	for _, i := range installs {
		out = append(out, installedToProto(i))
	}

	return connect.NewResponse(&enginev1.ListInstalledResponse{Entries: out}), nil
}

func installedToProto(i *models.MarketplaceInstall) *enginev1.InstalledEntry {
	return &enginev1.InstalledEntry{
		EntryId:       i.EntryID,
		Name:          i.Name,
		Type:          stringToEntryType[i.Type],
		InstalledRef:  i.InstalledRef,
		InstalledAtMs: i.CreatedAt.UnixMilli(),
	}
}
