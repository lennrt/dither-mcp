# Implementation checklist

## 1. Contracts

- [x] 1.1 Write and strictly validate OpenSpec requirements and scenarios.
- [x] 1.2 Typecheck and execute Quint publication and quantization models.
- [x] 1.3 Document model assumptions, bounds, and implementation refinement tests.

## 2. Image engine

- [x] 2.1 Implement a discoverable, tested algorithm registry with distinct kernels/screens.
- [x] 2.2 Implement named/custom/extracted palettes and explicit numeric validation.
- [x] 2.3 Implement transforms, masks, creative effects, deterministic seeds, and cancellation.
- [x] 2.4 Implement documented still encoders and print artifacts with format validation.

## 3. Local service and transports

- [x] 3.1 Implement rooted paths, size/work limits, atomic publication, and immutable destinations.
- [x] 3.2 Implement still inspection/render, comparisons, batch, and versioned recipes.
- [x] 3.3 Implement GIF, generated animation, sprite sheet, and optional local FFmpeg workflows.
- [x] 3.4 Implement stdio MCP discovery/tools/resources/prompts and a matching CLI.
- [x] 3.5 Exercise a real initialized MCP session and cancellation/error paths.

## 4. Delivery

- [x] 4.1 Add clear install/use/MCP/algorithm/security/reference documentation.
- [x] 4.2 Generate original sample artwork and real output galleries.
- [x] 4.3 Record multiple reproducible VHS demos and verify the resulting media.
- [x] 4.4 Build and check a separate static GitHub Pages-compatible showcase.
- [x] 4.5 Run Go tests/race/vet, specification checks, demo smoke tests, and final consistency review.
