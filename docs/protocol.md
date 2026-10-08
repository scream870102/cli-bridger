# CLI Bridger protocol v1

A cooperating executable handles `--cli-bridger-describe` as its only argument. It exits successfully and writes exactly one UTF-8 JSON object to stdout. Diagnostics belong on stderr. Discovery must not prompt or modify files. CLI Bridger never guesses a schema from `--help`; tools without this handshake need an adapter.

```json
{
  "version": "1",
  "name": "example",
  "description": "Example CLI",
  "root": {
    "id": "root",
    "name": "",
    "commands": [{
      "id": "log",
      "name": "log",
      "description": "Show history",
      "parameters": [
        {"id": "oneline", "flag": "--oneline", "type": "bool", "default": true},
        {"id": "count", "flag": "--count", "type": "int", "required": true,
         "default": 10, "examples": [5, 20], "limits": {"min": 1, "max": 100}},
        {"id": "prefix", "flag": "--prefix", "type": "string", "default": "commit",
         "examples": ["change"], "limits": {"minLength": 1, "maxLength": 40},
         "dependsOn": {"id": "oneline", "value": true}}
      ]
    }]
  }
}
```

The example illustrates the protocol; it is not a descriptor accepted by stock Git.

## Fields

Descriptor requires `version` (the string `"1"`), `name`, and `root`; `description` is optional. Unknown fields are rejected so accidental misspellings fail visibly.

Each command has a globally unique `id`, a `name` (literal argv token), optional `description`, ordered `parameters`, and nested `commands`. The root's `name` is ignored and should be empty. Selecting a child reveals its parameters; ancestor parameters remain active. Command IDs are used in the GUI selection path; names are passed to the executable. IDs and non-root command names match `[A-Za-z][A-Za-z0-9_.-]*`.

| Parameter field | Meaning |
| --- | --- |
| `id` | Required, globally unique across commands and parameters |
| `name`, `description` | Optional display label and help |
| `flag` | Option spelling, e.g. `--count` or `-n`; absent/empty means positional |
| `type` | Required: `string`, `int`, `float`, `path`, or `bool` |
| `required` | Defaults to false; true activates automatically when dependencies match |
| `default` | Typed initial value used when active input is absent; null means no default |
| `examples` | Optional array of valid typed examples |
| `enum` | Optional array of allowed typed values |
| `limits` | Numeric `min`/`max` (inclusive), or string/path `minLength`/`maxLength` (Unicode characters) |
| `dependsOn` | `{ "id": "earlier-parameter-id", "value": typedValue }`; requires that parameter to be active with that value |
| `pathKind` | For paths only: `file`, `directory`, or `save`; omitted means file chooser |

Numeric limits only apply to int/float; length limits only apply to string/path. Both numeric endpoints enable a finite slider; one endpoint still constrains input. Integers must be exact JSON-safe integers (absolute value at most 9007199254740991). Path values must be nonblank; file existence is left to the CLI (save paths may not exist). Values and examples must satisfy types, limits, and enums. Parameter flags cannot repeat within a command and its ancestor chain.

Dependencies reference earlier parameters in the same command or any ancestor. This ordering prevents cycles and makes activation deterministic. If a dependency is inactive or does not match, its dependent is omitted even when required or holding stale input. Optional parameters require explicit activation; a default alone does not enable them. A boolean emits its flag only for true; false emits nothing. Version 1 does not represent a separate negated flag, repeated option, variadic positional, or multiple dependency expression.

## Argument ordering and execution

Arguments follow descriptor order: root parameters, selected command name and its parameters, then each nested command. Flags with values use `--flag=value` (or `-n=value`), so participating CLIs must accept equals syntax. Boolean flags use `--flag`. Positionals retain parameter order. Positionals are permitted only on leaf commands; required positionals cannot follow optional ones, and activation cannot leave gaps. Positional text/path values beginning with `-` are rejected to prevent option injection; use an explicit relative path such as `./-filename`.

Run the executable with an argv array, never through a shell. A value like `a; echo hello` is a single literal argument. Schema descriptions and example values are text and must never become HTML or executable code. The descriptor advertises available operations; it does not make an executable trustworthy. Selecting and running a CLI executes that program with the user's account permissions.

Build the cooperative demo with `go build -o demo.exe ./examples/demo`, select it in the app, choose `render`, and run. The demo uses carriage returns and ANSI color so the terminal output can be checked, and never modifies files.
