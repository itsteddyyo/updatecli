package internal

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type containerImage struct {
	Registry   string
	Repository string
	Tag        string
	Path       string
}

func (image containerImage) Name() string {
	name := strings.Trim(image.Registry, "/")
	if name != "" {
		name += "/"
	}
	return name + strings.Trim(image.Repository, "/")
}

func Run(input Input, host HostFunctions) (*Output, error) {
	rootDir := input.Spec.RootDir
	if rootDir == "" {
		rootDir = "."
	}

	files, err := findFiles(rootDir, input.Spec.Files, input.Spec.ExcludeFiles)
	if err != nil {
		return nil, err
	}

	output := &Output{}
	for _, file := range files {
		relativeFile, err := filepath.Rel(rootDir, file)
		if err != nil {
			return nil, fmt.Errorf("getting path for %q: %w", file, err)
		}
		targetFile := filepath.ToSlash(relativeFile)
		if input.Spec.RootDir != "" && !filepath.IsAbs(input.Spec.RootDir) {
			targetFile = filepath.ToSlash(filepath.Join(input.Spec.RootDir, relativeFile))
		}

		documents, err := readDocuments(file)
		if err != nil {
			continue
		}
		for documentIndex, document := range documents {
			for _, source := range applicationSources(document) {
				valuesObject := nestedMap(source.value, "helm", "valuesObject")
				for _, image := range findContainerImages(valuesObject) {
					if matchesAnyPattern(image.Name(), input.Spec.ExcludeImages) {
						continue
					}
					manifest, err := generateManifest(input, host, targetFile, source.path, documentIndex, image)
					if err != nil {
						return nil, fmt.Errorf("generating image manifest for %q in %q: %w", image.Repository, targetFile, err)
					}
					if manifest != "" {
						output.Manifests = append(output.Manifests, manifest)
					}
				}
			}
		}
	}

	return output, nil
}

func findFiles(rootDir string, patterns, excludePatterns []string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(rootDir, func(file string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		extension := strings.ToLower(filepath.Ext(file))
		if extension != ".yaml" && extension != ".yml" {
			return nil
		}
		relativeFile, err := filepath.Rel(rootDir, file)
		if err != nil {
			return err
		}
		relativePath := filepath.ToSlash(relativeFile)
		baseName := filepath.Base(file)
		if len(patterns) > 0 && !matchesAnyPattern(relativePath, patterns) && !matchesAnyPattern(baseName, patterns) {
			return nil
		}
		if matchesAnyPattern(relativePath, excludePatterns) || matchesAnyPattern(baseName, excludePatterns) {
			return nil
		}
		files = append(files, file)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("searching yaml files in %q: %w", rootDir, err)
	}
	sort.Strings(files)
	return files, nil
}

func matchesAnyPattern(file string, patterns []string) bool {
	for _, pattern := range patterns {
		if matched, err := filepath.Match(filepath.FromSlash(pattern), file); err == nil && matched {
			return true
		}
	}
	return false
}

func readDocuments(file string) ([]map[string]interface{}, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	decoder := yaml.NewDecoder(strings.NewReader(string(content)))
	var documents []map[string]interface{}
	for {
		var document map[string]interface{}
		if err := decoder.Decode(&document); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		documents = append(documents, document)
	}
	return documents, nil
}

type sourceReference struct {
	path  string
	value map[string]interface{}
}

func applicationSources(document map[string]interface{}) []sourceReference {
	var sources []sourceReference
	spec := nestedMap(document, "spec")
	appendSources := func(spec map[string]interface{}, prefix string) {
		if source := nestedMap(spec, "source"); isChartSource(source) {
			sources = append(sources, sourceReference{path: prefix + ".source", value: source})
		}
		if sourceList, ok := spec["sources"].([]interface{}); ok {
			for index, item := range sourceList {
				if source, ok := item.(map[string]interface{}); ok && isChartSource(source) {
					sources = append(sources, sourceReference{path: fmt.Sprintf("%s.sources[%d]", prefix, index), value: source})
				}
			}
		}
	}
	appendSources(spec, "$.spec")
	if templateSpec := nestedMap(nestedMap(spec, "template"), "spec"); len(templateSpec) > 0 {
		appendSources(templateSpec, "$.spec.template.spec")
	}
	return sources
}

func isChartSource(source map[string]interface{}) bool {
	if len(source) == 0 {
		return false
	}
	repoURL, hasRepository := source["repoURL"].(string)
	revision, hasRevision := source["targetRevision"].(string)
	chart, hasChart := source["chart"].(string)
	return hasRepository && repoURL != "" && hasRevision && revision != "" && hasChart && chart != ""
}

func nestedMap(value map[string]interface{}, keys ...string) map[string]interface{} {
	for _, key := range keys {
		child, ok := value[key].(map[string]interface{})
		if !ok {
			return nil
		}
		value = child
	}
	return value
}

func findContainerImages(values map[string]interface{}) []containerImage {
	var images []containerImage
	var visit func(interface{}, string)
	visit = func(value interface{}, valuePath string) {
		switch typedValue := value.(type) {
		case map[string]interface{}:
			if repository, ok := typedValue["repository"].(string); ok && repository != "" {
				image := containerImage{
					Repository: strings.SplitN(repository, "@", 2)[0],
					Path:       valuePath,
				}
				image.Registry, _ = typedValue["registry"].(string)
				image.Tag, _ = typedValue["tag"].(string)
				image.Tag = strings.SplitN(image.Tag, "@", 2)[0]
				if image.Tag == "" {
					image.Tag = "latest"
				}
				images = append(images, image)
			}
			keys := make([]string, 0, len(typedValue))
			for key := range typedValue {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				child := typedValue[key]
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
