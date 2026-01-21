# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Case-insensitive literal matching**: Per-terminal `'SELECT'i` suffix and global `parse.WithCaseInsensitive(true)` option. Uses `strings.EqualFold()` for ~60% faster matching than regex `(?i)` patterns.
- **Multiple transform function signatures**: Transform functions can now optionally receive `*Node` and/or `*TransformContext` parameters for access to source positions, parent nodes, and siblings. Old-style functions continue to work unchanged.
- **Multi-pass transformations**: New `TransformMultiPass()` function enables sequential transformation passes with automatic structure preservation between passes.
- **Top-down transformations**: New `TransformTopDown()` function processes parent nodes before children, useful for scoping and symbol tables.
- **Transform context**: New `TransformContext` type provides access to parent, siblings, tree, and position information within transforms.
- **Transform metadata**: New `TransformResult` type allows attaching metadata to transformed values.
- **Enhanced error reporting**: New `TransformError` type includes source position, visual source snippets with highlighting, and context path.
- **Node type preservation**: New `Node.TransformedValue` field preserves type information across multi-pass transformations.
- **Context helper methods**: Added `IsFirst()`, `IsLast()`, `NextSibling()`, `PrevSibling()`, and `SiblingCount()` to `TransformContext`.
- **Panic recovery**: Transform functions that panic are now caught and wrapped in informative error messages with stack traces.
- Documentation: Added `parse/transform.md`, `parse/transform_quick_start.md`, and `parse/error_api_usage.md`.
- Example: Added `examples/markdown/` demonstrating multi-pass Markdown-to-HTML transformation.

### Changed

- `Transform()` and `TransformNode()` now support multiple function signature styles, automatically detected via reflection.
- Error messages from transforms now include source positions and helpful signature mismatch hints.
- README updated with simplified transform examples and links to comprehensive documentation.

### Removed

- Removed non-functional `GetParentMetadata()` method from `TransformContext`. Documentation now explains using closures for cross-node state sharing.

### Backward Compatibility

All existing transform code continues to work without changes. New features are opt-in and detected automatically based on function signatures.
