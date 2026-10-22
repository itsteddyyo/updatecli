package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type testHostFunctions struct{}

func (testHostFunctions) DockerSourceSpec(_, _ string) (*DockerSourceSpec, error) {
	return &DockerSourceSpec{
		TagFilter: "^v[0-9]+\\.[0-9]+\\.[0-9]+$",
		VersionFilter: VersionFilter{
			Kind:    "semver",
			Pattern: "*",
		},
	}, nil
}

func (testHostFunctions) VersionFilterGreaterThanPattern(filter *VersionFilter, tag string) error {
	filter.Pattern = ">=" + tag
	return nil
}

func TestRunDiscoversValuesObjectImagesAcrossAllSources(t *testing.T) {
	rootDir := t.TempDir()
	manifestPath := filepath.Join(rootDir, "applications.yaml")
	manifest := `---
{"apiVersion":"argoproj.io/v1alpha1","kind":"Application","spec":{"source":{"repoURL":"https://charts.example.com","chart":"frontend","targetRevision":"1.0.0","helm":{"valuesObject":{"workload":{"containers":[{"image":{"registry":"ghcr.io","repository":"org/frontend","tag":"v1.2.3"}}]}},"parameters":[{"name":"image.repository","value":"ignored-by-plugin"}],"valueFiles":["values.yaml"]}},"sources":[{"chart":"worker","repoURL":"https://charts.example.com","targetRevision":"2.0.0","helm":{"valuesObject":{"image":{"repository":"org/worker","tag":"2.3.4"}}}},{"ref":"values","helm":{"valuesObject":{"image":{"repository":"ignored-non-chart-source","tag":"1.0.0"}}}}]}}
---
{"apiVersion":"argoproj.io/v1alpha1","kind":"ApplicationSet","spec":{"template":{"spec":{"sources":[{"chart":"api","repoURL":"https://charts.example.com","targetRevision":"3.0.0","helm":{"valuesObject":{"images":{"api":{"registry":"docker.io","repository":"org/api","tag":"4.5.6"}}}}}]}}}}
`
	require.NoError(t, os.WriteFile(manifestPath, []byte(manifest), 0600))
	documents, err := readDocuments(manifestPath)
	require.NoError(t, err)
	require.Len(t, documents, 3)
	require.Len(t, applicationSources(documents[1]), 2)
	require.Len(t, findContainerImages(nestedMap(applicationSources(documents[1])[0].value, "helm", "valuesObject")), 1)

	output, err := Run(Input{Spec: Spec{RootDir: rootDir}}, testHostFunctions{})
	require.NoError(t, err)
	require.Len(t, output.Manifests, 3)

	allManifests := strings.Join(output.Manifests, "\n---\n")
	require.Contains(t, allManifests, "image: 'ghcr.io/org/frontend'")
	require.Contains(t, allManifests, "documentindex: 1")
	require.Contains(t, allManifests, "key: '$.spec.source.helm.valuesObject.workload.containers[0].image.tag'")
	require.Contains(t, allManifests, "image: 'org/worker'")
	require.Contains(t, allManifests, "key: '$.spec.sources[0].helm.valuesObject.image.tag'")
	require.Contains(t, allManifests, "image: 'docker.io/org/api'")
	require.Contains(t, allManifests, "documentindex: 2")
	require.Contains(t, allManifests, "key: '$.spec.template.spec.sources[0].helm.valuesObject.images.api.tag'")
	require.Contains(t, allManifests, "file: 'applications.yaml'")
	require.NotContains(t, allManifests, "helmchart")
	require.NotContains(t, allManifests, "ignored-by-plugin")
	require.NotContains(t, allManifests, "ignored-non-chart-source")
}

func TestRunAppliesVersionFilterAndDigestSettings(t *testing.T) {
	rootDir := t.TempDir()
	manifestPath := filepath.Join(rootDir, "app.yaml")
	manifest := `{"apiVersion":"argoproj.io/v1alpha1","kind":"Application","spec":{"source":{"repoURL":"https://charts.example.com","chart":"service","targetRevision":"1.0.0","helm":{"valuesObject":{"image":{"repository":"org/service","tag":"v1.2.3"}}}}}}
`
	require.NoError(t, os.WriteFile(manifestPath, []byte(manifest), 0600))
	digest := false
	output, err := Run(Input{
		Spec: Spec{
			RootDir: rootDir,
			Digest:  &digest,
			VersionFilter: VersionFilter{
				Kind:    "semver",
				Pattern: "minor",
			},
		},
	}, testHostFunctions{})
	require.NoError(t, err)
	require.Len(t, output.Manifests, 1)
	require.Contains(t, output.Manifests[0], "pattern: '>=v1.2.3'")
	require.NotContains(t, output.Manifests[0], "dockerdigest")
}

func TestFindFilesFiltersByPattern(t *testing.T) {
	rootDir := t.TempDir()
	for _, file := range []string{"apps/one.yaml", "apps/two.yml", "other.yaml", "ignored.txt"} {
		path := filepath.Join(rootDir, file)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0700))
		require.NoError(t, os.WriteFile(path, []byte("kind: Application\n"), 0600))
	}

	files, err := findFiles(rootDir, []string{"apps/*.yaml"})
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.Equal(t, filepath.Join(rootDir, "apps/one.yaml"), files[0])
}
