---
title: "hugo completion fish"
slug: hugo_completion_fish
url: /commands/hugo_completion_fish/
---
## hugo completion fish

generate the autocompletion script for fish

### Synopsis


Generate the autocompletion script for the fish shell.

To load completions in your current shell session:
$ hugo completion fish | source

To load completions for every new session, execute once:
$ hugo completion fish > ~/.config/fish/completions/hugo.fish

You will need to start a new shell for this setup to take effect.


```
hugo completion fish [flags]
```

### Options

```
  -h, --help              help for fish
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
      --config string              config file (default is path/config.yaml|json|toml)
      --configDir string           config dir (default "config")
      --debug                      debug output
  -e, --environment string         build environment
      --ignoreVendor               ignores any _vendor directory
      --ignoreVendorPaths string   ignores any _vendor for module paths matching the given Glob pattern
      --log                        enable Logging
      --logFile string             log File path (if set, logging enabled automatically)
      --quiet                      build in quiet mode
  -s, --source string              filesystem path to read files relative from
      --themesDir string           filesystem path to themes directory
  -v, --verbose                    verbose output
      --verboseLog                 verbose logging
```

### SEE ALSO

* [hugo completion](/commands/hugo_completion/)	 - generate the autocompletion script for the specified shell

