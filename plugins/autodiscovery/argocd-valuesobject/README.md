# Argo CD valuesObject image autodiscovery

This WASM autodiscovery plugin scans Argo CD `Application` and `ApplicationSet` YAML files and creates Updatecli manifests for container images declared as Helm `valuesObject` maps containing a `repository` field and an optional `registry` and `tag`.

It handles `spec.source`, `spec.sources`, `spec.template.spec.source`, and `spec.template.spec.sources`, including YAML files with multiple documents. It intentionally does not generate Helm chart updates. Image settings in Helm parameters and external `valueFiles` are not scanned.

## Build and test

Run `make test` to run the unit tests and build `argocd-valuesobject.wasm` for WASI.

## Configuration

```yaml
autodiscovery:
  crawlers:
    "argocd-valuesobject.wasm":
      spec:
        rootdir: manifests
        files:
          - "*.yaml"
          - "*.yml"
        digest: true
        versionfilter:
          kind: semver
          pattern: minor
```

`rootdir` is relative to the plugin's mounted workspace and defaults to `.`. When `files` is omitted, all `.yaml` and `.yml` files below `rootdir` are scanned. `files` entries are glob patterns matched against paths relative to `rootdir` or against the basename. Digest updates are enabled by default; set `digest: false` to update tags only. The version filter is optional and defaults to the filter inferred from the current image tag.