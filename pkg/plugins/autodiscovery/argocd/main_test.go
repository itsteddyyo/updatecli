package argocd

import (
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverManifests(t *testing.T) {
	testdata := []struct {
		name              string
		rootDir           string
		actionID          string
		scmID             string
		expectedPipelines []string
		spec              Spec
	}{
		{
			name:    "Expected no pipelines due to ignore rules",
			rootDir: "testdata/multi-release",
			spec: Spec{
				Only: MatchingRules{
					{
						Path: "donotexist",
					},
				},
			},
			expectedPipelines: []string{},
		},
		{
			name:    "ArgoCD manifests discovered with multiple documents",
			rootDir: "testdata/multi-release",
			expectedPipelines: []string{
				`name: 'deps(helm): bump Helm chart "nginx" in ArgoCD manifest "manifest.yaml"'
sources:
  nginx:
    name: 'Get latest "nginx" Helm chart version'
    kind: 'helmchart'
    spec:
      name: 'nginx'
      url: 'oci://registry-1.docker.io/bitnamicharts'
      versionfilter:
        kind: 'semver'
        pattern: '*'
conditions:
  nginx-name:
    name: 'Ensure Helm chart name nginx is specified'
    kind: 'yaml'
    disablesourceinput: true
    spec:
      file: 'manifest.yaml'
      key: '$.spec.source.chart'
      documentindex: 1
      value: 'nginx'
  nginx-repository:
    name: 'Ensure Helm chart repository oci://registry-1.docker.io/bitnamicharts is specified'
    kind: 'yaml'
    disablesourceinput: true
    spec:
      file: 'manifest.yaml'
      key: '$.spec.source.repoURL'
      documentindex: 1
      value: 'oci://registry-1.docker.io/bitnamicharts'
targets:
  nginx:
    name: 'deps(helm): update Helm chart "nginx" to {{ source "nginx" }}'
    kind: 'yaml'
    spec:
      file: 'manifest.yaml'
      key: '$.spec.source.targetRevision'
      documentindex: 1
    sourceid: 'nginx'
`,
				`name: 'deps(helm): bump Helm chart "nginx" in ArgoCD manifest "manifest.yaml"'
sources:
  nginx:
    name: 'Get latest "nginx" Helm chart version'
    kind: 'helmchart'
    spec:
      name: 'nginx'
      url: 'oci://registry-1.docker.io/bitnamicharts'
      versionfilter:
        kind: 'semver'
        pattern: '*'
conditions:
  nginx-name:
    name: 'Ensure Helm chart name nginx is specified'
    kind: 'yaml'
    disablesourceinput: true
    spec:
      file: 'manifest.yaml'
      key: '$.spec.template.spec.source.chart'
      documentindex: 2
      value: 'nginx'
  nginx-repository:
    name: 'Ensure Helm chart repository oci://registry-1.docker.io/bitnamicharts is specified'
    kind: 'yaml'
    disablesourceinput: true
    spec:
      file: 'manifest.yaml'
      key: '$.spec.template.spec.source.repoURL'
      documentindex: 2
      value: 'oci://registry-1.docker.io/bitnamicharts'
targets:
  nginx:
    name: 'deps(helm): update Helm chart "nginx" to {{ source "nginx" }}'
    kind: 'yaml'
    spec:
      file: 'manifest.yaml'
      key: '$.spec.template.spec.source.targetRevision'
      documentindex: 2
    sourceid: 'nginx'
`,
			},
		},
		{
			name:              "ArgoCD manifests discovered no source",
			rootDir:           "testdata/empty",
			expectedPipelines: []string{},
		},
		{
			name:    "ArgoCD manifests discovery with a single source and auths",
			rootDir: "testdata/sealed-secrets",
			spec: Spec{
				Auths: map[string]auth{
					"bitnami-labs.github.io": {
						Token: "token",
					},
				},
			},
			expectedPipelines: []string{`name: 'deps(helm): bump Helm chart "sealed-secrets" in ArgoCD manifest "manifest.yaml"'
sources:
  sealed-secrets:
    name: 'Get latest "sealed-secrets" Helm chart version'
    kind: 'helmchart'
    spec:
      name: 'sealed-secrets'
      url: 'https://bitnami-labs.github.io/sealed-secrets'
      token: 'token'
      versionfilter:
        kind: 'semver'
        pattern: '*'
conditions:
  sealed-secrets-name:
    name: 'Ensure Helm chart name sealed-secrets is specified'
    kind: 'yaml'
    disablesourceinput: true
    spec:
      file: 'manifest.yaml'
      key: '$.spec.source.chart'
      documentindex: 0
      value: 'sealed-secrets'
  sealed-secrets-repository:
    name: 'Ensure Helm chart repository https://bitnami-labs.github.io/sealed-secrets is specified'
    kind: 'yaml'
    disablesourceinput: true
    spec:
      file: 'manifest.yaml'
      key: '$.spec.source.repoURL'
      documentindex: 0
      value: 'https://bitnami-labs.github.io/sealed-secrets'
targets:
  sealed-secrets:
    name: 'deps(helm): update Helm chart "sealed-secrets" to {{ source "sealed-secrets" }}'
    kind: 'yaml'
    spec:
      file: 'manifest.yaml'
      key: '$.spec.source.targetRevision'
      documentindex: 0
    sourceid: 'sealed-secrets'
`},
		},
		{
			name:    "ArgoCD manifests discovery with a single source",
			rootDir: "testdata/sealed-secrets",
			expectedPipelines: []string{`name: 'deps(helm): bump Helm chart "sealed-secrets" in ArgoCD manifest "manifest.yaml"'
sources:
  sealed-secrets:
    name: 'Get latest "sealed-secrets" Helm chart version'
    kind: 'helmchart'
    spec:
      name: 'sealed-secrets'
      url: 'https://bitnami-labs.github.io/sealed-secrets'
      versionfilter:
        kind: 'semver'
        pattern: '*'
conditions:
  sealed-secrets-name:
    name: 'Ensure Helm chart name sealed-secrets is specified'
    kind: 'yaml'
    disablesourceinput: true
    spec:
      file: 'manifest.yaml'
      key: '$.spec.source.chart'
      documentindex: 0
      value: 'sealed-secrets'
  sealed-secrets-repository:
    name: 'Ensure Helm chart repository https://bitnami-labs.github.io/sealed-secrets is specified'
    kind: 'yaml'
    disablesourceinput: true
    spec:
      file: 'manifest.yaml'
      key: '$.spec.source.repoURL'
      documentindex: 0
      value: 'https://bitnami-labs.github.io/sealed-secrets'
targets:
  sealed-secrets:
    name: 'deps(helm): update Helm chart "sealed-secrets" to {{ source "sealed-secrets" }}'
    kind: 'yaml'
    spec:
      file: 'manifest.yaml'
      key: '$.spec.source.targetRevision'
      documentindex: 0
    sourceid: 'sealed-secrets'
`},
		},
		{
			name:    "ArgoCD manifests discovery with several sources",
			rootDir: "testdata/sealed-secrets_sources",
			expectedPipelines: []string{`name: 'deps(helm): bump Helm chart "sealed-secrets" in ArgoCD manifest "manifest.yaml"'
sources:
  sealed-secrets:
    name: 'Get latest "sealed-secrets" Helm chart version'
    kind: 'helmchart'
    spec:
      name: 'sealed-secrets'
      url: 'https://bitnami-labs.github.io/sealed-secrets'
      versionfilter:
        kind: 'semver'
        pattern: '*'
conditions:
  sealed-secrets-name:
    name: 'Ensure Helm chart name sealed-secrets is specified'
    kind: 'yaml'
    disablesourceinput: true
    spec:
      file: 'manifest.yaml'
      key: '$.spec.sources[0].chart'
      documentindex: 0
      value: 'sealed-secrets'
  sealed-secrets-repository:
    name: 'Ensure Helm chart repository https://bitnami-labs.github.io/sealed-secrets is specified'
    kind: 'yaml'
    disablesourceinput: true
    spec:
      file: 'manifest.yaml'
      key: '$.spec.sources[0].repoURL'
      documentindex: 0
      value: 'https://bitnami-labs.github.io/sealed-secrets'
targets:
  sealed-secrets:
    name: 'deps(helm): update Helm chart "sealed-secrets" to {{ source "sealed-secrets" }}'
    kind: 'yaml'
    spec:
      file: 'manifest.yaml'
      key: '$.spec.sources[0].targetRevision'
      documentindex: 0
    sourceid: 'sealed-secrets'
`},
		},
		{
			name:     "ArgoCD manifests discovery with several sources and both action and scm IDs",
			rootDir:  "testdata/sealed-secrets_sources",
			actionID: "argoRepo",
			scmID:    "scm123",
			expectedPipelines: []string{`name: 'deps(helm): bump Helm chart "sealed-secrets" in ArgoCD manifest "manifest.yaml"'
actions:
  argoRepo:
    title: 'deps(argocd): update Helm chart sealed-secrets to {{ source "sealed-secrets" }}'
sources:
  sealed-secrets:
    name: 'Get latest "sealed-secrets" Helm chart version'
    kind: 'helmchart'
    spec:
      name: 'sealed-secrets'
      url: 'https://bitnami-labs.github.io/sealed-secrets'
      versionfilter:
        kind: 'semver'
        pattern: '*'
conditions:
  sealed-secrets-name:
    name: 'Ensure Helm chart name sealed-secrets is specified'
    kind: 'yaml'
    disablesourceinput: true
    scmid: scm123
    spec:
      file: 'manifest.yaml'
      key: '$.spec.sources[0].chart'
      documentindex: 0
      value: 'sealed-secrets'
  sealed-secrets-repository:
    name: 'Ensure Helm chart repository https://bitnami-labs.github.io/sealed-secrets is specified'
    kind: 'yaml'
    disablesourceinput: true
    scmid: scm123
    spec:
      file: 'manifest.yaml'
      key: '$.spec.sources[0].repoURL'
      documentindex: 0
      value: 'https://bitnami-labs.github.io/sealed-secrets'
targets:
  sealed-secrets:
    name: 'deps(helm): update Helm chart "sealed-secrets" to {{ source "sealed-secrets" }}'
    kind: 'yaml'
    scmid: scm123
    spec:
      file: 'manifest.yaml'
      key: '$.spec.sources[0].targetRevision'
      documentindex: 0
    sourceid: 'sealed-secrets'
`},
		},
		{
			name:    "ArgoCD manifests discovery with OCI source",
			rootDir: "testdata/oci-helm-source",
			expectedPipelines: []string{`name: 'deps(helm): bump Helm chart "nginx" in ArgoCD manifest "manifest.yaml"'
sources:
  nginx:
    name: 'Get latest "nginx" Helm chart version'
    kind: 'helmchart'
    spec:
      name: 'nginx'
      url: 'oci://registry-1.docker.io/bitnamicharts'
      versionfilter:
        kind: 'semver'
        pattern: '*'
conditions:
  nginx-name:
    name: 'Ensure Helm chart name nginx is specified'
    kind: 'yaml'
    disablesourceinput: true
    spec:
      file: 'manifest.yaml'
      key: '$.spec.source.chart'
      documentindex: 0
      value: 'nginx'
  nginx-repository:
    name: 'Ensure Helm chart repository registry-1.docker.io/bitnamicharts is specified'
    kind: 'yaml'
    disablesourceinput: true
    spec:
      file: 'manifest.yaml'
      key: '$.spec.source.repoURL'
      documentindex: 0
      value: 'registry-1.docker.io/bitnamicharts'
targets:
  nginx:
    name: 'deps(helm): update Helm chart "nginx" to {{ source "nginx" }}'
    kind: 'yaml'
    spec:
      file: 'manifest.yaml'
      key: '$.spec.source.targetRevision'
      documentindex: 0
    sourceid: 'nginx'
`},
		},
		{
			name:    "ArgoCD ApplicationSet manifests discovery with several sources",
			rootDir: "testdata/appset-sealed-secrets_sources",
			expectedPipelines: []string{`name: 'deps(helm): bump Helm chart "sealed-secrets" in ArgoCD manifest "manifest.yaml"'
sources:
  sealed-secrets:
    name: 'Get latest "sealed-secrets" Helm chart version'
    kind: 'helmchart'
    spec:
      name: 'sealed-secrets'
      url: 'https://bitnami-labs.github.io/sealed-secrets'
      versionfilter:
        kind: 'semver'
        pattern: '*'
conditions:
  sealed-secrets-name:
    name: 'Ensure Helm chart name sealed-secrets is specified'
    kind: 'yaml'
    disablesourceinput: true
    spec:
      file: 'manifest.yaml'
      key: '$.spec.template.spec.sources[0].chart'
      documentindex: 0
      value: 'sealed-secrets'
  sealed-secrets-repository:
    name: 'Ensure Helm chart repository https://bitnami-labs.github.io/sealed-secrets is specified'
    kind: 'yaml'
    disablesourceinput: true
    spec:
      file: 'manifest.yaml'
      key: '$.spec.template.spec.sources[0].repoURL'
      documentindex: 0
      value: 'https://bitnami-labs.github.io/sealed-secrets'
targets:
  sealed-secrets:
    name: 'deps(helm): update Helm chart "sealed-secrets" to {{ source "sealed-secrets" }}'
    kind: 'yaml'
    spec:
      file: 'manifest.yaml'
      key: '$.spec.template.spec.sources[0].targetRevision'
      documentindex: 0
    sourceid: 'sealed-secrets'
`},
		},
	}

	for _, tt := range testdata {

		t.Run(tt.name, func(t *testing.T) {
			argocd, err := New(
				tt.spec, tt.rootDir, tt.scmID, tt.actionID)

			require.NoError(t, err)

			var pipelines []string
			rawPipelines, err := argocd.DiscoverManifests()
			require.NoError(t, err)

			// Sort pipelines by name to ensure consistent order for assertion
			sort.Slice(rawPipelines, func(i, j int) bool {
				return string(rawPipelines[i]) < string(rawPipelines[j])
			})

			require.Equal(t, len(tt.expectedPipelines), len(rawPipelines), "number of discovered pipelines does not match expected")

			for i := range rawPipelines {
				// We expect manifest generated by the autodiscovery to use the yaml syntax
				pipelines = append(pipelines, string(rawPipelines[i]))
				assert.Equal(t, tt.expectedPipelines[i], pipelines[i])
			}
		})
	}
}

func TestDiscoverContainerImagesFromValuesObject(t *testing.T) {
	argocd, err := New(Spec{}, "testdata/values-object", "", "")
	require.NoError(t, err)

	manifests, err := argocd.DiscoverManifests()
	require.NoError(t, err)

	var imageManifests []string
	for _, manifest := range manifests {
		if strings.Contains(string(manifest), "kind: 'dockerimage'") {
			imageManifests = append(imageManifests, string(manifest))
		}
	}

	require.Len(t, imageManifests, 2)
	assert.Contains(t, imageManifests[0]+imageManifests[1], "image: 'ghcr.io/org/frontend'")
	assert.Contains(t, imageManifests[0]+imageManifests[1], "key: '$.spec.source.helm.valuesObject.workload.container.image.tag'")
	assert.Contains(t, imageManifests[0]+imageManifests[1], "kind: 'dockerdigest'")
	assert.Contains(t, imageManifests[0]+imageManifests[1], "hidetag: true")
	assert.Contains(t, imageManifests[0]+imageManifests[1], "- trimprefix: '@'")
	assert.Contains(t, imageManifests[0]+imageManifests[1], "key: '$.spec.source.helm.valuesObject.workload.container.image.digest'")
	assert.Contains(t, imageManifests[0]+imageManifests[1], "image: 'docker.io/org/worker'")
	assert.Contains(t, imageManifests[0]+imageManifests[1], "key: '$.spec.sources[0].helm.valuesObject.images.worker.tag'")
	assert.Contains(t, imageManifests[0]+imageManifests[1], "key: '$.spec.sources[0].helm.valuesObject.images.worker.digest'")
	assert.NotContains(t, imageManifests[0]+imageManifests[1], "parameter-only")
	assert.NotContains(t, imageManifests[0]+imageManifests[1], "non-chart")
}

func TestDiscoverContainerImagesWithInlineDigest(t *testing.T) {
	argocd, err := New(Spec{}, "testdata/values-object", "", "")
	require.NoError(t, err)

	source := ApplicationSourceSpec{
		RepoURL:        "https://charts.example.com",
		TargetRevision: "1.0.0",
		Chart:          "frontend",
	}
	source.Helm.ValuesObject = map[string]interface{}{
		"image": map[string]interface{}{
			"repository": "org/frontend@sha256:old",
			"tag":        "v1.2.3@sha256:old",
		},
	}

	manifests, err := argocd.generateContainerImageManifests(source, "manifest.yaml", "$.spec.source", 0)
	require.NoError(t, err)
	require.Len(t, manifests, 1)

	manifest := string(manifests[0])
	assert.Contains(t, manifest, "image: 'org/frontend'")
	assert.Contains(t, manifest, "pattern: '>=v1.2.3'")
	assert.NotContains(t, manifest, "@sha256:old")
}

func TestDisableContainerImageDigests(t *testing.T) {
	digest := false
	argocd, err := New(Spec{Digest: &digest}, "testdata/values-object", "", "")
	require.NoError(t, err)

	manifests, err := argocd.DiscoverManifests()
	require.NoError(t, err)

	for _, manifest := range manifests {
		assert.NotContains(t, string(manifest), "kind: 'dockerdigest'")
	}
}

func TestIgnoreContainerImages(t *testing.T) {
	argocd, err := New(Spec{IgnoreContainer: true}, "testdata/values-object", "", "")
	require.NoError(t, err)

	manifests, err := argocd.DiscoverManifests()
	require.NoError(t, err)
	require.Len(t, manifests, 2)

	for _, manifest := range manifests {
		assert.NotContains(t, string(manifest), "kind: 'dockerimage'")
		assert.Contains(t, string(manifest), "kind: 'helmchart'")
	}
}
