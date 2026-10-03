# Writing style

Use the [Google developer documentation style guide](https://developers.google.com/style)
and the [ASD-STE100 skill](https://github.com/danyuchn/asd-ste100-skill/blob/master/SKILL.md)
in STE-flavored mode. This mode applies sentence structure rules and treats
vocabulary restrictions as guidance. It does not establish certified ASD-STE100
compliance.

Apply this style to documentation, code comments, CLI help, MCP descriptions,
palette descriptions, and website text.

## Write clearly

- Use active voice and identify the actor.
- Put conditions before instructions.
- Give one instruction per sentence.
- Keep instructions within 20 words and descriptions within 25 words where precision permits.
- Write complete sentences. Keep each paragraph on one topic.
- Use plain words, American spelling, and sentence case headings.
- Use consistent terms and serial commas.
- Replace prose semicolons with separate sentences.
- Describe capabilities positively. State technical limits directly.
- Preserve numbers, conditions, uncertainty, and requirement strength.

## Preserve technical meaning

Keep identifiers, commands, flags, and file paths exact. Format them as code in
Markdown and HTML. Retain necessary terms such as *quantization*, *alpha*, and
*error diffusion*. Explain unfamiliar terms where readers first need them.

Use `catalog` for the searchable registry. Use `palette` for an ordered set of
colors. Use `recipe` for versioned image processing settings. Use `artifact` for a file
that a tool creates.

Keep Go documentation comments attached to their identifiers. Preserve compiler
directives and required OpenSpec keywords. Keep license text and third-party
notices verbatim. Retain proper names and published titles.

## Review changes

Read each edit for meaning before you check its style. Treat automated findings
as review prompts. Tables, code, formulas, and proper names can produce false
positives. If a shorter sentence loses precision, keep the precise wording.

Edit generated text at its source. Regenerate palette documentation, MCP schemas,
website data, and affected demos after the source changes.
