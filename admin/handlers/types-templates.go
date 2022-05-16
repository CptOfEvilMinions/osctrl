package handlers

import (
	"github.com/jmpsec/osctrl/carves"
	"github.com/jmpsec/osctrl/environments"
	"github.com/jmpsec/osctrl/nodes"
	"github.com/jmpsec/osctrl/queries"
	"github.com/jmpsec/osctrl/settings"
	"github.com/jmpsec/osctrl/tags"
	"github.com/jmpsec/osctrl/types"
	"github.com/jmpsec/osctrl/users"
)

// LoginTemplateData for passing data to the login template
type LoginTemplateData struct {
	Title   string
	Project string
}

// TemplateMetadata to pass some metadata to templates
type TemplateMetadata struct {
	Username       string
	Level          string
	Service        string
	Version        string
	TLSDebug       bool
	AdminDebug     bool
	APIDebug       bool
	AdminDebugHTTP bool
	APIDebugHTTP   bool
	CSRFToken      string
}

// TableTemplateData for passing data to the table template
type TableTemplateData struct {
	Title        string
	Selector     string
	SelectorName string
	Target       string
	Tags         []tags.AdminTag
	Environments []environments.TLSEnvironment
	Platforms    []string
	Metadata     TemplateMetadata
}

// ConfTemplateData for passing data to the conf template
type ConfTemplateData struct {
	Title        string
	Environment  environments.TLSEnvironment
	Environments []environments.TLSEnvironment
	Platforms    []string
	Metadata     TemplateMetadata
}

// EnrollTemplateData for passing data to the conf template
type EnrollTemplateData struct {
	Title                 string
	EnvName               string
	OnelinerExpiration    bool
	EnrollExpiry          string
	EnrollExpired         bool
	RemoveExpiry          string
	RemoveExpired         bool
	QuickAddShell         string
	QuickRemoveShell      string
	QuickAddPowershell    string
	QuickRemovePowershell string
	Secret                string
	Flags                 string
	Certificate           string
	Environments          []environments.TLSEnvironment
	Platforms             []string
	Metadata              TemplateMetadata
}

// QueryRunTemplateData for passing data to the query run template
type QueryRunTemplateData struct {
	Title         string
	Environments  []environments.TLSEnvironment
	Platforms     []string
	UUIDs         []string
	Hosts         []string
	Tables        []types.OsqueryTable
	TablesVersion string
	Metadata      TemplateMetadata
}

// CarvesRunTemplateData for passing data to the carves run template
type CarvesRunTemplateData QueryRunTemplateData

// GenericTableTemplateData for passing data to a table template
type GenericTableTemplateData struct {
	Title        string
	Environments []environments.TLSEnvironment
	Platforms    []string
	Target       string
	Metadata     TemplateMetadata
}

// QueryTableTemplateData for passing data to the query template
type QueryTableTemplateData GenericTableTemplateData

// SavedQueriesTemplateData for passing data to the saved queries
type SavedQueriesTemplateData GenericTableTemplateData

// CarvesTableTemplateData for passing data to the carves template
type CarvesTableTemplateData GenericTableTemplateData

// CarvesDetailsTemplateData for passing data to the carves details
type CarvesDetailsTemplateData struct {
	Title        string
	Environments []environments.TLSEnvironment
	Platforms    []string
	Query        queries.DistributedQuery
	QueryTargets []queries.DistributedQueryTarget
	Carves       []carves.CarvedFile
	CarveBlocks  map[string][]carves.CarvedBlock
	Metadata     TemplateMetadata
}

// QueryLogsTemplateData for passing data to the query template
type QueryLogsTemplateData struct {
	Title        string
	Environments []environments.TLSEnvironment
	Platforms    []string
	Query        queries.DistributedQuery
	QueryTargets []queries.DistributedQueryTarget
	Metadata     TemplateMetadata
}

// EnvironmentsTemplateData for passing data to the environments template
type EnvironmentsTemplateData struct {
	Title        string
	Environments []environments.TLSEnvironment
	Platforms    []string
	Metadata     TemplateMetadata
}

// SettingsTemplateData for passing data to the settings template
type SettingsTemplateData struct {
	Title           string
	Service         string
	Environments    []environments.TLSEnvironment
	Platforms       []string
	CurrentSettings []settings.SettingValue
	ServiceConfig   types.JSONConfigurationService
	Metadata        TemplateMetadata
}

// UsersTemplateData for passing data to the users template
type UsersTemplateData struct {
	Title        string
	Environments []environments.TLSEnvironment
	Platforms    []string
	CurrentUsers []users.AdminUser
	Metadata     TemplateMetadata
}

// ProfileTemplateData for passing data to the users profile template
type ProfileTemplateData struct {
	Title        string
	Environments []environments.TLSEnvironment
	Platforms    []string
	CurrentUser  users.AdminUser
	Metadata     TemplateMetadata
}

// TagsTemplateData for passing data to the tags template
type TagsTemplateData struct {
	Title        string
	Environments []environments.TLSEnvironment
	Platforms    []string
	Tags         []tags.AdminTag
	Metadata     TemplateMetadata
}

// NodeTemplateData for passing data to the query template
type NodeTemplateData struct {
	Title        string
	Node         nodes.OsqueryNode
	NodeTags     []tags.AdminTag
	TagsForNode  []tags.AdminTagForNode
	Environments []environments.TLSEnvironment
	Platforms    []string
	Metadata     TemplateMetadata
	Dashboard    bool
	Schedule     environments.ScheduleConf
	Packs        environments.PacksEntries
}
