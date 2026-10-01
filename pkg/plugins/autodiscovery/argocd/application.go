package argocd

import (
	"bytes"
	"fmt"
	"maps"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/sirupsen/logrus"
	"github.com/updatecli/updatecli/pkg/plugins/resources/dockerimage"
)

var (
	// ArgoCDFilePatterns specifies accepted Helm chart metadata file name
	ArgoCDFilePatterns [2]string = [2]string{"*.yaml", "*.yml"}
)

// ArgocdApplicationSpec represents the subset of ArgoCD application manifest relevant for Updatecli autodiscovery
type ArgoCDApplicationSpec struct {
	ApiVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Spec       struct {
		Source   ApplicationSourceSpec
		Sources  []ApplicationSourceSpec
		Template struct {
			Spec struct {
				Source  ApplicationSourceSpec
				Sources []ApplicationSourceSpec
			}
		}
	}
}

type ApplicationSourceSpec struct {
	RepoURL        string `yaml:"repoURL"`
	TargetRevision string `yaml:"targetRevision"`
	Chart          string `yaml:"chart"`
	Ref            string `yaml:"ref"`
	Helm           struct {
		ValuesObject map[string]interface{} `yaml:"valuesObject"`
	} `yaml:"helm"`
}

type containerImage struct {
	Registry   string
	Repository string
	Tag        string
	Path       string
}

func findContainerImages(values map[string]interface{}) []containerImage {
	var images []containerImage

	var visit func(interface{}, string)
	visit = func(value interface{}, valuePath string) {
		switch typedValue := value.(type) {
		case map[string]interface{}:
			if repository, ok := typedValue["repository"].(string); ok && repository != "" {
				image := containerImage{Repository: repository, Path: valuePath}
				image.Registry, _ = typedValue["registry"].(string)
				image.Tag, _ = typedValue["tag"].(string)
				if image.Tag == "" {
					image.Tag = "latest"
				}
				images = append(images, image)
			}

			for key, child := range typedValue {
				keyPath := key
				if valuePath != "" {
					keyPath = valuePath + "." + key
				}
				visit(child, keyPath)
			}
		case []interface{}:
			for index, child := range typedValue {
				visit(child, fmt.Sprintf("%s[%d]", valuePath, index))
			}
		}
	}

	visit(values, "")
	return images
}

// IsZero checks if the ApplicationSourceSpec is empty in the context of Updatecli
// We prefer ignoring when chart and targetRevision are empty
func (a ApplicationSourceSpec) IsZero() bool {
	return a.RepoURL == "" || a.TargetRevision == "" || a.Chart == ""
}

func determineChartRepository(RepoURL string) (string, error) {
	parsedURL, err := url.Parse(RepoURL)
	if err != nil {
		return "", err
	}

	if parsedURL.Scheme == "" {
		// Combine "oci://" and the RepoURL
		return fmt.Sprintf("oci://%s", RepoURL), nil
	}

	return RepoURL, nil
}

func (f ArgoCD) discoverArgoCDManifests() ([][]byte, error) {

	var manifests [][]byte

	searchFromDir := f.rootDir
	// If the spec.RootDir is an absolute path, then it as already been set
	// correctly in the New function.
	if f.spec.RootDir != "" && !path.IsAbs(f.spec.RootDir) {
		searchFromDir = filepath.Join(f.rootDir, f.spec.RootDir)
	}

	foundFiles, err := searchArgoCDFiles(
		searchFromDir,
		ArgoCDFilePatterns[:])

	if err != nil {
		return nil, err
	}

	for _, foundFile := range foundFiles {
		logrus.Debugf("parsing file %q", foundFile)

		relativeFilepath, err := filepath.Rel(f.rootDir, foundFile)
		if err != nil {
			// Let's try the next chart if one fail
			logrus.Debugln(err)
			continue
		}

		// Retrieve chart dependencies for each chart
		d, err := readManifest(foundFile)
		if err != nil {
			logrus.Debugln(err)
			continue
		}

		for documentIndex := range maps.Keys(d) {
			data := d[documentIndex]
			if !data.Spec.Source.IsZero() {
				manifest, err := f.generateManifestBySource(
					data.Spec.Source,
					relativeFilepath,
					"$.spec.source",
					documentIndex,
				)
				if err != nil {
					logrus.Errorf("error discovering application source: %s", err)
					continue
				}

				if manifest != nil {
					manifests = append(manifests, manifest)
				}

				imageManifests, err := f.generateContainerImageManifests(
					data.Spec.Source,
					relativeFilepath,
					"$.spec.source",
					documentIndex,
				)
				if err != nil {
					logrus.Errorf("error discovering application source images: %s", err)
					continue
				}
				manifests = append(manifests, imageManifests...)
			}

			if !data.Spec.Template.Spec.Source.IsZero() {
				manifest, err := f.generateManifestBySource(
					data.Spec.Template.Spec.Source,
					relativeFilepath,
					"$.spec.template.spec.source",
					documentIndex,
				)
				if err != nil {
					logrus.Errorf("error discovering application source: %s", err)
					continue
				}

				if manifest != nil {
					manifests = append(manifests, manifest)
				}

				imageManifests, err := f.generateContainerImageManifests(
					data.Spec.Template.Spec.Source,
					relativeFilepath,
					"$.spec.template.spec.source",
					documentIndex,
				)
				if err != nil {
					logrus.Errorf("error discovering application source images: %s", err)
					continue
				}
				manifests = append(manifests, imageManifests...)
			}

			for i, source := range data.Spec.Sources {
				if source.IsZero() {
					continue
				}

				manifest, err := f.generateManifestBySource(
					source,
					relativeFilepath,
					fmt.Sprintf("$.spec.sources[%d]", i),
					documentIndex,
				)
				if err != nil {
					logrus.Errorf("error discovering application source: %s", err)
					continue
				}

				if manifest != nil {
					manifests = append(manifests, manifest)
				}

				imageManifests, err := f.generateContainerImageManifests(
					source,
					relativeFilepath,
					fmt.Sprintf("$.spec.sources[%d]", i),
					documentIndex,
				)
				if err != nil {
					logrus.Errorf("error discovering application source images: %s", err)
					continue
				}
				manifests = append(manifests, imageManifests...)
			}

			for i, source := range data.Spec.Template.Spec.Sources {
				if source.IsZero() {
					continue
				}

				manifest, err := f.generateManifestBySource(
					source,
					relativeFilepath,
					fmt.Sprintf("$.spec.template.spec.sources[%d]", i),
					documentIndex,
				)
				if err != nil {
					logrus.Errorf("error discovering application source: %s", err)
					continue
				}

				if manifest != nil {
					manifests = append(manifests, manifest)
				}

				imageManifests, err := f.generateContainerImageManifests(
					source,
					relativeFilepath,
					fmt.Sprintf("$.spec.template.spec.sources[%d]", i),
					documentIndex,
				)
				if err != nil {
					logrus.Errorf("error discovering application source images: %s", err)
					continue
				}
				manifests = append(manifests, imageManifests...)
			}
		}
	}

	return manifests, nil
}

func (f ArgoCD) generateContainerImageManifests(data ApplicationSourceSpec, file, sourcePath string, yamlDocument int) ([][]byte, error) {
	if f.spec.IgnoreContainer || data.IsZero() || len(data.Helm.ValuesObject) == 0 {
		return nil, nil
	}

	var manifests [][]byte
	for _, image := range findContainerImages(data.Helm.ValuesObject) {
		imageName := strings.Trim(image.Registry, "/")
		if imageName != "" {
			imageName += "/"
		}
		imageName += strings.Trim(image.Repository, "/")

		sourceSpec := dockerimage.NewDockerImageSpecFromImage(imageName, image.Tag, nil)
		if sourceSpec == nil {
			continue
		}

		versionFilterKind := sourceSpec.VersionFilter.Kind
		versionFilterPattern := sourceSpec.VersionFilter.Pattern
		versionFilterRegex := sourceSpec.VersionFilter.Regex
		tagFilter := sourceSpec.TagFilter
		if !f.spec.VersionFilter.IsZero() {
			versionFilterKind = f.versionFilter.Kind
			var err error
			versionFilterPattern, err = f.versionFilter.GreaterThanPattern(image.Tag)
			if err != nil {
				logrus.Debugf("building image version filter pattern: %s", err)
				versionFilterPattern = "*"
			}
			versionFilterRegex = f.versionFilter.Regex
			tagFilter = ""
		}

		tmpl, err := template.New("containerImageManifest").Parse(containerImageManifestTemplate)
		if err != nil {
			return nil, err
		}

		valuesPath := sourcePath + ".helm.valuesObject." + image.Path
		params := struct {
			ActionID                   string
			ImageName                  string
			ChartName                  string
			SourceID                   string
			SourceVersionFilterKind    string
			SourceVersionFilterPattern string
			SourceVersionFilterRegex   string
			SourceTagFilter            string
			Registry                   string
			Repository                 string
			RegistryKey                string
			RepositoryKey              string
			TagKey                     string
			File                       string
			ScmID                      string
			YamlDocument               int
		}{
			ActionID:                   f.actionID,
			ImageName:                  imageName,
			ChartName:                  data.Chart,
			SourceID:                   strings.ReplaceAll(imageName, "/", "_"),
			SourceVersionFilterKind:    versionFilterKind,
			SourceVersionFilterPattern: versionFilterPattern,
			SourceVersionFilterRegex:   versionFilterRegex,
			SourceTagFilter:            tagFilter,
			Registry:                   image.Registry,
			Repository:                 image.Repository,
			RegistryKey:                valuesPath + ".registry",
			RepositoryKey:              valuesPath + ".repository",
			TagKey:                     valuesPath + ".tag",
			File:                       file,
			ScmID:                      f.scmID,
			YamlDocument:               yamlDocument,
		}

		manifest := bytes.Buffer{}
		if err := tmpl.Execute(&manifest, params); err != nil {
			return nil, fmt.Errorf("executing container image manifest template: %w", err)
		}
		manifests = append(manifests, manifest.Bytes())
	}

	return manifests, nil
}

// generateManifestBySource generates an ArgoCD manifest for Updatecli based on the application source input
func (f ArgoCD) generateManifestBySource(data ApplicationSourceSpec, file string, targetKey string, yamlDocument int) ([]byte, error) {

	var err error

	if data.IsZero() {
		return nil, nil
	}

	if len(f.spec.Ignore) > 0 {
		if f.spec.Ignore.isMatchingRules(f.rootDir, file, data.RepoURL, data.Chart, data.TargetRevision) {
			logrus.Debugf("Ignoring Helm chart %q from %q, as matching ignore rule(s)\n", data.Chart, file)
			return nil, nil
		}
	}

	if len(f.spec.Only) > 0 {
		if !f.spec.Only.isMatchingRules(f.rootDir, file, data.RepoURL, data.Chart, data.TargetRevision) {
			logrus.Debugf("Ignoring Helm chart %q from %q, as not matching only rule(s)\n", data.Chart, file)
			return nil, nil
		}
	}

	sourceVersionFilterKind := "semver"
	sourceVersionFilterPattern := "*"
	sourceVersionFilterRegex := "*"

	if !f.spec.VersionFilter.IsZero() {
		sourceVersionFilterKind = f.versionFilter.Kind
		sourceVersionFilterPattern, err = f.versionFilter.GreaterThanPattern(data.TargetRevision)
		sourceVersionFilterRegex = f.versionFilter.Regex
		if err != nil {
			logrus.Debugf("building version filter pattern: %s", err)
			sourceVersionFilterPattern = "*"
		}
	}

	username := ""
	password := ""
	token := ""
	repoURL, err := url.Parse(data.RepoURL) // to validate URL format
	switch err {
	case nil:
		if _, ok := f.spec.Auths[repoURL.Host]; ok {
			username = f.spec.Auths[repoURL.Host].Username
			password = f.spec.Auths[repoURL.Host].Password
			token = f.spec.Auths[repoURL.Host].Token
			logrus.Debugf("found credentials for repository %q", data.RepoURL)
		}
	default:
		logrus.Debugf("Ignoring auth configuration due to invalid Helm repository URL: %s", err)
	}

	sourceChartRepository, err := determineChartRepository(data.RepoURL)
	if err != nil {
		logrus.Debugf("invalid URL: %s", err)
		return nil, nil
	}

	tmpl, err := template.New("manifest").Parse(manifestTemplate)
	if err != nil {
		logrus.Debugln(err)
		return nil, nil
	}

	params := struct {
		ActionID                   string
		ManifestName               string
		ImageName                  string
		ChartName                  string
		ChartRepository            string
		ManifestFile               string
		ConditionID                string
		SourceChartRepository      string
		SourceID                   string
		SourceName                 string
		SourceKind                 string
		SourceVersionFilterKind    string
		SourceVersionFilterPattern string
		SourceVersionFilterRegex   string
		TargetID                   string
		TargetKey                  string
		TargetYamlDocument         int
		Username                   string
		Password                   string
		Token                      string
		File                       string
		ScmID                      string
	}{
		ActionID:                   f.actionID,
		ManifestName:               fmt.Sprintf("deps(helm): bump Helm chart %q in ArgoCD manifest %q", data.Chart, file),
		ChartName:                  data.Chart,
		ChartRepository:            data.RepoURL,
		ConditionID:                data.Chart,
		ManifestFile:               file,
		SourceChartRepository:      sourceChartRepository,
		SourceID:                   data.Chart,
		SourceName:                 fmt.Sprintf("Get latest %q Helm chart version", data.Chart),
		SourceKind:                 "helmchart",
		SourceVersionFilterKind:    sourceVersionFilterKind,
		SourceVersionFilterPattern: sourceVersionFilterPattern,
		SourceVersionFilterRegex:   sourceVersionFilterRegex,
		TargetID:                   data.Chart,
		TargetKey:                  targetKey,
		TargetYamlDocument:         yamlDocument,
		File:                       file,
		ScmID:                      f.scmID,
		Username:                   username,
		Password:                   password,
		Token:                      token,
	}

	manifest := bytes.Buffer{}
	if err = tmpl.Execute(&manifest, params); err != nil {
		return nil, fmt.Errorf("error executing template: %w", err)
	}

	return manifest.Bytes(), nil
}
