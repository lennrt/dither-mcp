# Palette library design

## Context

The image engine already resolves a stable identifier or custom colors into an ordered opaque sRGB palette. Discovery currently returns the entire small preset list. The expanded catalog needs useful browsing. Calls should not need to return all entries, and prior recipes must remain valid.

## Goals and non-goals

The design has these goals:

- Provide 256 distinct and useful curated choices.
- Preserve original palette IDs and colors.
- Provide clear metadata and provenance.
- Bound local search and keep page traversal stable.
- Give MCP and CLI requests identical semantics.

The design excludes these behaviors:

- Download remote palettes or issue runtime web requests.
- Manage user accounts.
- Silently replace historical palette approximations with new values.
- Claim perceptual uniqueness or exact rendering matches with another product.

## Decisions

1. Keep palette resolution in `engine`. Put query validation, filtering, and pagination in `internal/app`. Discovery is read-only and runs under the existing admission and request limits.
2. Preserve each original palette's ID and ordered color array. Metadata can gain detail without changing the colors that a stored recipe resolves to.
3. Validate uniqueness at two levels: unique IDs and unique normalized color-set signatures. A permutation of another palette is not a new catalog entry. Order still determines how one palette resolves equal-distance ties.
4. Use explicit lowercase six-digit hex colors and complete metadata. Original artistic palettes identify themselves as original. Externally sourced palettes cite an appropriate source. Historical and display approximations state that status. A source URL is information, never a runtime fetch.
5. Combine all supplied filters with logical AND. Filter before pagination. Sort palette IDs lexicographically. The response reports `total`, `matched`, `count`, effective `limit` and `offset`, optional `next_offset`, and whole-catalog category IDs/counts. Limit defaults to 32 and cannot exceed 256. An offset beyond the result returns an empty page. Queries use case-insensitive substring terms combined with AND across ID/name/description/category/tags/origin/colors. The query limit is 256 UTF-8 bytes. The service trims and lowercases category slugs. Unknown slugs fail. Color-count bounds default from zero to 2/256.
6. A query result is an ephemeral view over a static versioned catalog. Offsets remain stable for an unchanged query and catalog version. They are not durable cursors across catalog changes. A caller changing a filter should restart at offset zero.
7. Model six abstract catalog entries in Quint. Exercise filtering, stable slicing, concatenated pages, termination at the matched count, and rejection of invalid bounds. Go tests independently validate the real 256 entries, query semantics, compatibility, and transport schemas.

## Risks and tradeoffs

Large libraries can accumulate look-alike palettes, arbitrary labels, or misleading historical attribution. Tests can enforce unique color sets. Aesthetic usefulness and truthful descriptions also require review. Related families are appropriate when each has explicit design intent and a distinct color set. Count alone does not establish quality.

Bounded default pages change the old full-list discovery behavior. Keep the `palettes` field. Return clear pagination metadata. Update tool descriptions and examples. Test enumeration through the last page. The runtime rendering contract for existing named IDs remains unchanged.

## Validation plan

Strictly validate OpenSpec. Typecheck and execute the finite Quint discovery model, including empty matches, filtered nonempty pages, over-end offsets, and complete traversal. Compare the original 22 palette arrays to a fixed compatibility fixture. Check all catalog IDs, colors, metadata, source fields, and order-independent set signatures. Test every discovery filter alone and in combination, boundaries, repeated pages, concatenation, MCP schema conformance, and CLI parity.
