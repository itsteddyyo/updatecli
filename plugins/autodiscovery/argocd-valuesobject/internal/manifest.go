package internal

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

type manifestParams struct {
	ActionID             string
	Digest               bool
	ImageName            string
	ImageID              string
	Registry             string
	Repository           string
	RegistryKey          string
	RepositoryKey        string
	TagKey               string
	File                 string
	ScmID                string
	DocumentIndex        int
	TagFilter            string
	VersionFilterKind    string
	VersionFilterPattern string
	VersionFilterRegex   string
}

func generateManifest(input Input, host HostFunctions, file, sourcePath string, documentIndex int, image containerImage) (string, error) {
	imageName := strings.Trim(image.Registry, "/")
	if imageName != "" {
		imageName += "/"
	}
	imageName += strings.Trim(image.Repository, "/")
	if imageName == "" {
		return "", nil
	}

	versionFilter := VersionFilter{Kind: "semver", Pattern: "*"}
	tagFilter := ""
	if sourceSpec, err := host.DockerSourceSpec(imageName, image.Tag); err == nil && sourceSpec != nil {
		if !sourceSpec.VersionFilter.IsZero() {
			versionFilter = sourceSpec.VersionFilter
		}
		tagFilter = sourceSpec.TagFilter
	}
	if !input.Spec.VersionFilter.IsZero() {
		versionFilter = input.Spec.VersionFilter
		if versionFilter.Kind != "lex" {
			if err := host.VersionFilterGreaterThanPattern(&versionFilter, image.Tag); err != nil {
				if versionFilter.Kind == "semver" {
					return "", nil
				}
				return "", fmt.Errorf("building version filter pattern: %w", err)
			}
		}
		tagFilter = ""
	}

	digest := true
	if input.Spec.Digest != nil {
		digest = *input.Spec.Digest
	}
	imageID := strings.ReplaceAll(imageName, "/", "_")
	valuesPath := sourcePath + ".helm.valuesObject." + image.Path
	params := manifestParams{
		ActionID:             input.ActionID,
		Digest:               digest,
		ImageName:            imageName,
		ImageID:              imageID,
		Registry:             image.Registry,
		Repository:           image.Repository,
		RegistryKey:          valuesPath + ".registry",
		RepositoryKey:        valuesPath + ".repository",
		TagKey:               valuesPath + ".tag",
		File:                 file,
		ScmID:                input.ScmID,
		DocumentIndex:        documentIndex,
		TagFilter:            tagFilter,
		VersionFilterKind:    versionFilter.Kind,
		VersionFilterPattern: versionFilter.Pattern,
		VersionFilterRegex:   versionFilter.Regex,
	}

	tmpl, err := template.New("imageManifest").Parse(manifestTemplate)
	if err != nil {
		return "", err
	}
	var manifest bytes.Buffer
	if err := tmpl.Execute(&manifest, params); err != nil {
		return "", err
	}
	return manifest.String(), nil
}

const manifestTemplate = `name: 'deps(argocd): bump image "{{ .ImageName }}" tag{{ if .Digest }} and digest{{ end }}'
{{- if .ActionID }}
actions:
  {{ .ActionID }}:
    title: 'deps(argocd): update Docker image {{ .ImageName }} to {{ "{{" }} source "{{ .ImageID }}" {{ "}}" }}'
{{- end }}
sources:
  '{{ .ImageID }}':
    name: 'get latest image tag for "{{ .ImageName }}"'
    kind: 'dockerimage'
    spec:
      image: '{{ .ImageName }}'
{{- if .TagFilter }}
      tagfilter: '{{ .TagFilter }}'
{{- end }}
      versionfilter:
        kind: '{{ .VersionFilterKind }}'
        pattern: '{{ .VersionFilterPattern }}'
{{- if or (eq .VersionFilterKind "regex/semver") (eq .VersionFilterKind "regex/time") }}
        regex: '{{ .VersionFilterRegex }}'
{{- end }}
{{- if .Digest }}
  '{{ .ImageID }}-digest':
    name: 'get latest image digest for "{{ .ImageName }}"'
    kind: 'dockerdigest'
    spec:
      image: '{{ .ImageName }}'
      tag: '{{ "{{" }} source "{{ .ImageID }}" {{ "}}" }}'
    dependson:
      - '{{ .ImageID }}'
{{- end }}
conditions:
{{- if .Registry }}
  '{{ .ImageID }}-registry':
    name: 'Ensure container registry {{ .Registry }} is specified'
    kind: 'yaml'
    disablesourceinput: true
{{- if .ScmID }}
    scmid: '{{ .ScmID }}'
{{- end }}
    spec:
      file: '{{ .File }}'
      key: '{{ .RegistryKey }}'
      documentindex: {{ .DocumentIndex }}
      value: '{{ .Registry }}'
{{- end }}
  '{{ .ImageID }}-repository':
    name: 'Ensure container repository {{ .Repository }} is specified'
    kind: 'yaml'
    disablesourceinput: true
{{- if .ScmID }}
    scmid: '{{ .ScmID }}'
{{- end }}
    spec:
      file: '{{ .File }}'
      key: '{{ .RepositoryKey }}'
      documentindex: {{ .DocumentIndex }}
      value: '{{ .Repository }}'
targets:
  '{{ .ImageID }}':
    name: 'deps(argocd): bump image "{{ .ImageName }}" tag{{ if .Digest }} and digest{{ end }}'
    kind: 'yaml'
{{- if .ScmID }}
    scmid: '{{ .ScmID }}'
{{- end }}
    spec:
      file: '{{ .File }}'
      key: '{{ .TagKey }}'
      documentindex: {{ .DocumentIndex }}
{{- if .Digest }}
    sourceid: '{{ .ImageID }}-digest'
{{- else }}
    sourceid: '{{ .ImageID }}'
{{- end }}
`
