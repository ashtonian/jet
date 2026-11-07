# Context-aware compilation

`Set.ParseContext` and `Set.GetTemplateContext` admit one compilation and its import/extends graph. Legacy `Parse` and `GetTemplate` keep their signatures and unlimited settings. Compilation is synchronous, including scanner activity and loader cleanup. These limits do not establish a restricted template sandbox or limit execution, output, arbitrary Go code, or process memory.

```go
tpl, err := set.GetTemplateContext(ctx, "views/body.jet", jet.ParseOptions{
    MaxSourceBytes:      64 << 10,
    MaxTotalSourceBytes: 256 << 10,
    MaxNodes:           20000,
    MaxSteps:           1000000,
    MaxDepth:           32,
    MaxTemplates:       32,
})
```

These values illustrate opt-in settings; they are not universal safe defaults. Nil context and negative signed limits return `ErrInvalidParseOptions`. Zero is unlimited for each dimension. Choose finite limits appropriate to the supported source size and composition. An arbitrarily large positive depth does not protect a process stack.

| ParseOptions field | Type | Default | Counts |
| --- | --- | --- | --- |
| MaxSourceBytes | int64 | 0 | Bytes in each individual source, checked before inline parsing and during named reads |
| MaxTotalSourceBytes | int64 | 0 | Unique canonical sources admitted in the operation; repeated versions of a name charge any size increase |
| MaxNodes | uint64 | 0 | AST construction; development-mode reparsing charges each construction; a cached graph charges each immutable AST once |
| MaxSteps | uint64 | 0 | Scanning, tokens, AST construction, block merging and dependency/cache visits; exact counts are implementation details |
| MaxDepth | int | 0 | Active template/parser frames and constructed expression nesting; checked before recursive descent and expression construction |
| MaxTemplates | uint64 | 0 | Unique canonical source names in the operation |

Source and template allowances count shared diamonds once. Traversals and repeated parsing still consume work, so a development-mode dependency DAG cannot do unlimited repeated work under a finite step allowance. Each top-level call starts an independent admission. Runtime-triggered loading needs an execution-owned admission to share allowances; the standalone compilation API alone does not supply that bridge.

Cached templates retain immutable source/node/depth costs and dependencies. A stricter caller revalidates the graph; warming a cache under unlimited options does not bypass later limits. Context, counters and active dependency chains are cleared before publication. Templates from a different Set are treated as cache misses even if a caller supplies the same Cache instance. Do not mutate parsed ASTs or the Set configuration after construction; exported AST fields are intended for inspection and are not an admission extension point.

The scanner runs on the parsing goroutine. Cancellation is checked within rune scans, fixed-size text/comment scans, token consumption and parser loops. Diagnostics truncate source tokens and template names, and line lookup advances incrementally. `errors.Is` reaches context cancellation, loader failures, `ErrTemplateCycle` and the specific `ErrParse*Limit` sentinel. `errors.As` can inspect `ParseLimitError.Kind` and `.Limit`. A malformed dependency preserves its cause through parent template errors.

Loader Exists/Open/Read/Close calls are owned synchronously. Readers close before return on success, syntax error, limit and cancellation; a close failure is returned when no earlier error takes precedence. Jet does not launch a goroutine and abandon it to simulate interruption. An optional `ContextLoader` can observe the supplied context in ExistsContext/OpenContext and its returned reader. Cooperative behavior is the loader's responsibility: arbitrary blocking loader or reader code still prevents return. Use bounded process-owned sources when a stronger capability contract is needed.

No external service or new module dependency is required. The existing Loader/Cache interfaces and custom delimiters, comments, relative lookup, extension lookup and development mode remain available. Focused regressions are `go test -run 'TestCompilation|TestLexer' .`; run the repository's normal and race compatibility suites when integrating changes.
