package argocd

const (
	// manifestTemplate is the Go template used to generate ArgoCD application manifests
	manifestTemplate string = `name: '{{ .ManifestName }}'
{{- if .ActionID }}
actions:
  {{ .ActionID }}:
    title: 'deps(argocd): update Helm chart {{ .ChartName }} to {{ "{{" }} source "{{ .SourceID }}" {{ "}}" }}'
{{- end }}
sources:
  {{ .SourceID }}:
    name: '{{ .SourceName }}'
    kind: '{{ .SourceKind }}'
    spec:
      name: '{{ .ChartName }}'
      url: '{{ .SourceChartRepository }}'
      {{- if .Username }}
      username: '{{ .Username }}'
      {{- end }}
      {{- if .Password }}
      password: '{{ .Password }}'
      {{- end }}
      {{- if .Token }}
      token: '{{ .Token }}'
      {{- end }}
      versionfilter:
        kind: '{{ .SourceVersionFilterKind }}'
        pattern: '{{ .SourceVersionFilterPattern }}'
        {{- if or (eq .SourceVersionFilterKind "regex/semver") (eq .SourceVersionFilterKind "regex/time") }}
        regex: '{{ .SourceVersionFilterRegex }}'
        {{- end }}
conditions:
  {{ .ConditionID }}-name:
    name: 'Ensure Helm chart name {{ .ChartName }} is specified'
    kind: 'yaml'
    disablesourceinput: true
{{- if .ScmID }}
    scmid: {{ .ScmID }}
{{- end }}
    spec:
      file: '{{ .File }}'
      key: '{{ .TargetKey }}.chart'
      documentindex: {{ .TargetYamlDocument }}
      value: '{{ .ChartName }}'
  {{ .ConditionID }}-repository:
    name: 'Ensure Helm chart repository {{ .ChartRepository }} is specified'
    kind: 'yaml'
    disablesourceinput: true
{{- if .ScmID }}
    scmid: {{ .ScmID }}
{{- end }}
    spec:
      file: '{{ .File }}'
      key: '{{ .TargetKey }}.repoURL'
      documentindex: {{ .TargetYamlDocument }}
      value: '{{ .ChartRepository }}'
targets:
  {{ .TargetID }}:
    name: 'deps(helm): update Helm chart "{{ .ChartName }}" to {{ "{{" }} source "{{ .SourceID }}" {{ "}}" }}'
    kind: 'yaml'
{{- if .ScmID }}
    scmid: {{ .ScmID }}
{{- end }}
    spec:
      file: '{{ .File }}'
      key: '{{ .TargetKey }}.targetRevision'
      documentindex: {{ .TargetYamlDocument }}
    sourceid: '{{ .SourceID }}'
`
	containerImageManifestTemplate string = `name: 'deps(argocd): bump image "{{ .ImageName }}" tag for chart "{{ .ChartName }}"'
{{- if .ActionID }}
actions:
  {{ .ActionID }}:
    title: 'deps(argocd): update Docker image {{ .ImageName }} to {{ "{{" }} source "{{ .SourceID }}" {{ "}}" }}'
{{- end }}
sources:
  {{ .SourceID }}:
    name: 'get latest image tag for "{{ .ImageName }}"'
    kind: 'dockerimage'
    spec:
      image: '{{ .ImageName }}'
      tagfilter: '{{ .SourceTagFilter }}'
      versionfilter:
        kind: '{{ .SourceVersionFilterKind }}'
        pattern: '{{ .SourceVersionFilterPattern }}'
{{- if or (eq .SourceVersionFilterKind "regex/semver") (eq .SourceVersionFilterKind "regex/time") }}
        regex: '{{ .SourceVersionFilterRegex }}'
{{- end }}
conditions:
{{- if .Registry }}
  {{ .SourceID }}-registry:
    name: 'Ensure container registry {{ .Registry }} is specified'
    kind: 'yaml'
    disablesourceinput: true
{{- if .ScmID }}
    scmid: '{{ .ScmID }}'
{{- end }}
    spec:
      file: '{{ .File }}'
      key: '{{ .RegistryKey }}'
      documentindex: {{ .YamlDocument }}
      value: '{{ .Registry }}'
{{- end }}
  {{ .SourceID }}-repository:
    name: 'Ensure container repository {{ .Repository }} is specified'
    kind: 'yaml'
    disablesourceinput: true
{{- if .ScmID }}
    scmid: '{{ .ScmID }}'
{{- end }}
    spec:
      file: '{{ .File }}'
      key: '{{ .RepositoryKey }}'
      documentindex: {{ .YamlDocument }}
      value: '{{ .Repository }}'
targets:
  {{ .SourceID }}:
    name: 'deps(argocd): bump image "{{ .ImageName }}" tag'
    kind: 'yaml'
{{- if .ScmID }}
    scmid: '{{ .ScmID }}'
{{- end }}
    spec:
      file: '{{ .File }}'
      key: '{{ .TagKey }}'
      documentindex: {{ .YamlDocument }}
    sourceid: '{{ .SourceID }}'
`
)
