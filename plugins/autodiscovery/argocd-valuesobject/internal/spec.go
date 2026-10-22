package internal

type Input struct {
	ScmID    string `json:"scmid"`
	ActionID string `json:"actionid"`
	RootDir  string `json:"rootdir"`
	Spec     Spec   `json:"spec"`
}

type Output struct {
	Manifests []string `json:"manifests"`
}

type Spec struct {
	RootDir       string        `json:"rootdir,omitempty"`
	Files         []string      `json:"files,omitempty"`
	Digest        *bool         `json:"digest,omitempty"`
	VersionFilter VersionFilter `json:"versionfilter,omitempty"`
}

type VersionFilter struct {
	Kind    string `json:"kind,omitempty"`
	Pattern string `json:"pattern,omitempty"`
	Regex   string `json:"regex,omitempty"`
}

func (f VersionFilter) IsZero() bool {
	return f.Kind == "" && f.Pattern == "" && f.Regex == ""
}

type DockerSourceSpec struct {
	TagFilter     string        `json:"tagfilter,omitempty"`
	VersionFilter VersionFilter `json:"versionfilter,omitempty"`
}

type HostFunctions interface {
	DockerSourceSpec(image, tag string) (*DockerSourceSpec, error)
	VersionFilterGreaterThanPattern(filter *VersionFilter, pattern string) error
}
