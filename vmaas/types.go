package vmaas

import (
	"time"

	"github.com/redhatinsights/vmaas-lib/vmaas/utils"
)

type (
	RepoID       int
	PkgID        int
	NameID       int
	EvrID        int
	ArchID       int
	ErrataID     int
	ContentSetID int
	DefinitionID int
	CpeID        int
	CriteriaID   int
	TestID       int
	ModuleTestID int
	OvalStateID  int
)

type Request struct {
	Packages   []string       `json:"package_list"`
	Repos      *[]string      `json:"repository_list"`
	Modules    []ModuleStream `json:"modules_list"`
	Releasever *string        `json:"releasever"`
	Basearch   *string        `json:"basearch"`
	RepoPaths  []string       `json:"repository_paths"`

	ThirdParty   bool `json:"third_party"`
	Optimistic   bool `json:"optimistic_updates"`
	LatestOnly   bool `json:"latest_only"`
	SecurityOnly bool `json:"security_only"`

	Extended bool `json:"extended"`
}

type Update struct {
	Package    string `json:"package"`
	Erratum    string `json:"erratum"`
	Repository string `json:"repository"`
	Basearch   string `json:"basearch"`
	Releasever string `json:"releasever"`
}

type UpdateDetail struct {
	AvailableUpdates []Update `json:"available_updates,omitempty"`
}

type UpdateList map[string]UpdateDetail

type Updates struct {
	UpdateList UpdateList     `json:"update_list"`
	RepoList   *[]string      `json:"repository_list,omitempty"`
	RepoPaths  []string       `json:"repository_paths,omitempty"`
	ModuleList []ModuleStream `json:"modules_list,omitempty"`
	Releasever *string        `json:"releasever,omitempty"`
	Basearch   *string        `json:"basearch,omitempty"`
	LastChange time.Time      `json:"last_change"`
}

type Vulnerability string

type VulnerabilityDetail struct {
	CVE      string   `json:"cve"`
	Packages []string `json:"affected_packages"`
	Errata   []string `json:"errata"`
}

type Vulnerabilities struct {
	CVEs                []Vulnerability `json:"cve_list"`
	ManuallyFixableCVEs []Vulnerability `json:"manually_fixable_cve_list"`
	UnpatchedCVEs       []Vulnerability `json:"unpatched_cve_list"`
	LastChange          time.Time       `json:"last_change"`
}

type VulnerabilitiesExtended struct {
	CVEs                []VulnerabilityDetail `json:"cve_list"`
	ManuallyFixableCVEs []VulnerabilityDetail `json:"manually_fixable_cve_list"`
	UnpatchedCVEs       []VulnerabilityDetail `json:"unpatched_cve_list"`
	LastChange          time.Time             `json:"last_change"`
}

type NevraIDs struct {
	NameID NameID
	EvrIDs []int
	ArchID ArchID
}

type PackageDetail struct {
	NameID        NameID
	EvrID         EvrID
	ArchID        ArchID
	SummaryID     int
	DescriptionID int

	SrcPkgID   *PkgID
	Modified   *time.Time
	ModifiedID int
}

type Nevra struct {
	NameID NameID
	EvrID  EvrID
	ArchID ArchID
}

type RepoDetail struct {
	Label      string
	Name       string
	URL        string
	Basearch   string
	Releasever string
	Product    string
	ProductID  int
	Revision   *string
	ThirdParty bool
}

type CveDetail struct {
	RedHatURL     *string
	SecondaryURL  *string
	Cvss3Score    *string
	Cvss3Metrics  *string
	Impact        string
	PublishedDate *string
	ModifiedDate  *string
	Iava          *string
	Description   string
	Cvss2Score    *string
	Cvss2Metrics  *string
	Source        string

	CWEs      []string
	PkgIDs    []int
	ErrataIDs []int
}

type PkgErrata struct {
	PkgID    int
	ErrataID int
}

type Module struct {
	Name              string
	StreamID          int
	Stream            string
	Version           string
	Context           string
	PackageList       []string
	SourcePackageList []string
}

type ModuleStream struct {
	Module string `json:"module_name"`
	Stream string `json:"module_stream"`
}

type DBChange struct {
	ErrataChanges string `json:"errata_changes"`
	CveChanges    string `json:"cve_changes"`
	RepoChanges   string `json:"repository_changes"`
	LastChange    string `json:"last_change"`
	Exported      string `json:"exported"`
}

type ErrataDetail struct {
	ID             ErrataID
	Synopsis       string
	Summary        *string
	Type           string
	Severity       *string
	Description    *string
	CVEs           []string
	PkgIDs         []int
	ModulePkgIDs   []int
	Bugzillas      []string
	Refs           []string
	Modules        []Module
	Solution       *string
	Issued         *string
	Updated        *string
	URL            string
	ThirdParty     bool
	RequiresReboot bool
}

type DefinitionDetail struct {
	ID               DefinitionID
	DefinitionTypeID int
	CriteriaID       CriteriaID
}

type OvalTestDetail struct {
	PkgNameID      NameID
	CheckExistence int
}

type OvalModuleTestDetail struct {
	ModuleStream ModuleStream
}

type OvalState struct {
	ID           OvalStateID
	EvrID        EvrID
	OperationEvr int
}

type NameArch struct {
	Name string
	Arch string
}

type NevraString struct {
	Nevra utils.Nevra
	Pkg   string
}
