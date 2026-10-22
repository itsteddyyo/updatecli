//go:build wasip1

package main

import (
	"encoding/json"
	"fmt"

	pdk "github.com/extism/go-pdk"
	"github.com/updatecli/plugins/autodiscovery/argocd-valuesobject/internal"
)

type wasmHostFunctions struct{}

//go:wasmimport extism:host/user generate_docker_source_spec
func generateDockerSourceSpec(uint64) uint64

//go:wasmimport extism:host/user versionfilter_greater_than_pattern
func versionFilterGreaterThanPattern(uint64) uint64

func (wasmHostFunctions) DockerSourceSpec(image, tag string) (*internal.DockerSourceSpec, error) {
	input, err := json.Marshal(struct {
		Image string `json:"image"`
		Tag   string `json:"tag"`
	}{Image: image, Tag: tag})
	if err != nil {
		return nil, fmt.Errorf("marshaling docker source spec input: %w", err)
	}

	memory := pdk.AllocateString(string(input))
	defer memory.Free()
	response := memoryResponse(generateDockerSourceSpec(memory.Offset()))

	var result internal.DockerSourceSpec
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("parsing docker source spec response: %w", err)
	}
	return &result, nil
}

func (wasmHostFunctions) VersionFilterGreaterThanPattern(filter *internal.VersionFilter, pattern string) error {
	input, err := json.Marshal(struct {
		VersionFilter internal.VersionFilter `json:"versionfilter"`
		Pattern       string                 `json:"pattern"`
	}{VersionFilter: *filter, Pattern: pattern})
	if err != nil {
		return fmt.Errorf("marshaling version filter input: %w", err)
	}

	memory := pdk.AllocateString(string(input))
	defer memory.Free()
	response := memoryResponse(versionFilterGreaterThanPattern(memory.Offset()))
	var result struct {
		Pattern string `json:"pattern"`
	}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return fmt.Errorf("parsing version filter response: %w", err)
	}
	filter.Pattern = result.Pattern
	return nil
}

func memoryResponse(pointer uint64) string {
	memory := pdk.FindMemory(pointer)
	return string(memory.ReadBytes())
}
