//go:build wasip1

package main

import (
	"fmt"

	pdk "github.com/extism/go-pdk"
	"github.com/updatecli/plugins/autodiscovery/argocd-valuesobject/internal"
)

//go:wasmexport autodiscovery
func autodiscovery() int32 {
	input := internal.Input{}
	if err := pdk.InputJSON(&input); err != nil {
		pdk.SetError(fmt.Errorf("unable to parse plugin input: %w", err))
		return -1
	}

	output, err := internal.Run(input, wasmHostFunctions{})
	if err != nil {
		pdk.SetError(fmt.Errorf("running plugin: %w", err))
		return -1
	}

	if output == nil {
		output = &internal.Output{}
	}
	if err := pdk.OutputJSON(output); err != nil {
		pdk.SetError(err)
		return -1
	}
	return 0
}

func main() {
	autodiscovery()
}
