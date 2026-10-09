// A separate finite abstraction of guarded export admission. Request identity,
// dirty settings, host capability, and cancellation belong to mcp_apps.qnt.
// Tokens represent encoded byte identities, assuming collision-free digests.
// This does not implement SHA-256, codecs, filesystem reads, or actual pixels.
module studio_export {
  type Export = {
    mode: str, width: int, height: int, selectedWidth: int, selectedHeight: int,
    recipe: int, acceptedRecipe: int,
    source: int, mask: int, masked: bool, explicit: bool
  }
  type Publication = { request: Export, source: int, mask: int }
  type State = {
    source: int, mask: int, accepted: bool, masked: bool,
    expectedSource: int, expectedMask: int, recipe: int,
    mode: str, customWidth: int, customHeight: int,
    phase: str, request: Export, readSource: int, readMask: int,
    decodedSource: int, decodedMask: int,
    writes: List[Publication], mismatches: int, previews: int
  }
  var s: State
  pure val PreviewWidth = 2
  pure val PreviewHeight = 2
  pure val SourceWidth = 6
  pure val SourceHeight = 4
  pure val MaxAxis = 8
  pure val MaxPixels = 48
  pure def bounded(width, height) = all {
    width > 0, height > 0, width <= MaxAxis, height <= MaxAxis,
    width * height <= MaxPixels
  }
  val exportWidth = if (s.mode == "preview") PreviewWidth
    else if (s.mode == "source") SourceWidth else s.customWidth
  val exportHeight = if (s.mode == "preview") PreviewHeight
    else if (s.mode == "source") SourceHeight else s.customHeight
  val canSave = all {
    s.phase == "idle", s.accepted, s.writes.length() < 3,
    bounded(exportWidth, exportHeight)
  }
  action init = s' = {
    source: 0, mask: 0, accepted: false, masked: false,
    expectedSource: -1, expectedMask: -1, recipe: 0,
    mode: "preview", customWidth: 0, customHeight: 0,
    phase: "idle", request: {
      mode: "preview", width: 0, height: 0, selectedWidth: 0, selectedHeight: 0,
      recipe: -1, acceptedRecipe: -1,
      source: -1, mask: -1, masked: false, explicit: false
    }, readSource: -1, readMask: -1, decodedSource: -1, decodedMask: -1,
    writes: List(), mismatches: 0, previews: 0
  }
  // Independent disk mutations can happen before, during, or after loading.
  action changeSource = all { s.source < 2, s' = { source: s.source + 1, ...s } }
  action changeMask = all { s.mask < 2, s' = { mask: s.mask + 1, ...s } }
  action preview(masked, recipe) = all {
    s.phase == "idle", s.previews < 3,
    s' = {
      accepted: true, masked: masked,
      expectedSource: s.source, expectedMask: if (masked) s.mask else -1,
      recipe: recipe, previews: s.previews + 1, ...s
    }
  }
  action selectExport(mode, width, height) = all {
    s.phase == "idle", Set("preview", "source", "custom").contains(mode),
    s' = { mode: mode, customWidth: width, customHeight: height, ...s }
  }
  action authorize(explicit) = all {
    canSave, explicit,
    s' = { phase: "authorized", request: {
      mode: s.mode, width: exportWidth, height: exportHeight,
      selectedWidth: exportWidth, selectedHeight: exportHeight,
      recipe: s.recipe, acceptedRecipe: s.recipe,
      source: s.expectedSource, mask: s.expectedMask, masked: s.masked,
      explicit: explicit
    }, ...s }
  }
  // Each token is an immutable captured buffer. No claim of a simultaneous
  // multi-file filesystem snapshot follows from this abstract action.
  action loadBuffers = all {
    s.phase == "authorized",
    s' = { phase: "loaded", readSource: s.source,
      readMask: if (s.request.masked) s.mask else -1, ...s }
  }
  action checkBuffers = {
    val matched = s.readSource == s.request.source and s.readMask == s.request.mask
    all {
      s.phase == "loaded",
      s' = {
        phase: if (matched) "checked" else "idle",
        accepted: if (matched) s.accepted else false,
        mismatches: s.mismatches + (if (matched) 0 else 1), ...s
      }
    }
  }
  action decodeBuffers = all {
    s.phase == "checked",
    s' = { phase: "decoded", decodedSource: s.readSource,
      decodedMask: s.readMask, ...s }
  }
  action publish = all {
    s.phase == "decoded",
    s' = { phase: "idle", writes: s.writes.append({ request: s.request,
      source: s.decodedSource, mask: s.decodedMask }), ...s }
  }
  action failSave = all {
    s.phase != "idle", s' = { phase: "idle", ...s }
  }
  action stutter = s' = s
  action step = {
    nondet masked = Set(false, true).oneOf()
    nondet recipe = 0.to(2).oneOf()
    nondet mode = Set("preview", "source", "custom").oneOf()
    nondet width = 0.to(9).oneOf()
    nondet height = 0.to(9).oneOf()
    any {
      changeSource, changeMask, preview(masked, recipe),
      selectExport(mode, width, height), authorize(masked), loadBuffers,
      checkBuffers, decodeBuffers, publish, failSave, stutter
    }
  }
  // Deliberately reach a source-size export with a disk change after loading.
  action initExport = init.then(preview(true, 1)).then(selectExport("source", 0, 0))
  action exportStep = any {
    all { s.writes.length() == 0, s.phase == "idle", authorize(true) },
    loadBuffers,
    all { s.phase == "loaded", s.source == 0, changeSource },
    all { s.source == 1, checkBuffers },
    decodeBuffers, publish, stutter
  }
  pure def preserved(a) = all {
    a.explicit, a.recipe == a.acceptedRecipe, bounded(a.width, a.height),
    a.width == a.selectedWidth, a.height == a.selectedHeight,
    a.mode == "preview" implies a.width == PreviewWidth and a.height == PreviewHeight,
    a.mode == "source" implies a.width == SourceWidth and a.height == SourceHeight,
    a.masked implies a.mask >= 0,
    not(a.masked) implies a.mask == -1
  }
  val capturedGuard = s.phase == "checked" or s.phase == "decoded" implies all {
    s.readSource == s.request.source, s.readMask == s.request.mask
  }
  val decodedGuard = s.phase == "decoded" implies all {
    s.decodedSource == s.readSource, s.decodedMask == s.readMask
  }
  val publicationsMatch = s.writes.indices().forall(i => {
    val p = s.writes.nth(i)
    all { preserved(p.request), p.source == p.request.source,
      p.mask == p.request.mask }
  })
  val safety = all {
    s.source >= 0, s.source <= 2, s.mask >= 0, s.mask <= 2,
    s.previews <= 3, s.writes.length() <= 3,
    s.phase != "idle" implies preserved(s.request),
    capturedGuard, decodedGuard, publicationsMatch,
    not(s.accepted) implies not(canSave)
  }
  val sourceSizedExport = s.writes.indices().exists(i => s.writes.nth(i).request.mode == "source")
  val customSizedExport = s.writes.indices().exists(i => s.writes.nth(i).request.mode == "custom")
  val changedInputRejected = s.mismatches > 0
  val capturedBeforeDiskChange = s.writes.indices().exists(i => s.writes.nth(i).source < s.source)

  run defaultReplaysPreviewSizeTest = init.then(preview(false, 2)).then(authorize(true))
    .then(loadBuffers).then(checkBuffers).then(decodeBuffers).then(publish)
    .expect(safety and s.writes.nth(0).request.width == 2 and s.writes.nth(0).request.height == 2)
  run sourceSizePreservesRecipeTest = initExport.then(authorize(true)).then(loadBuffers)
    .then(checkBuffers).then(decodeBuffers).then(publish)
    .expect(safety and sourceSizedExport and s.writes.nth(0).request.recipe == 1)
  run customSizePreservesRecipeTest = init.then(preview(true, 2))
    .then(selectExport("custom", 8, 6)).then(authorize(true)).then(loadBuffers)
    .then(checkBuffers).then(decodeBuffers).then(publish)
    .expect(safety and customSizedExport and s.writes.nth(0).request.recipe == 2
      and s.writes.nth(0).request.width == 8 and s.writes.nth(0).request.height == 6)
  run exportChoiceNeedsNoNewPreviewTest = init.then(preview(false, 1))
    .then(selectExport("source", 0, 0)).expect(safety and canSave and s.previews == 1)
  run missingCustomSizeRejectedTest = init.then(preview(false, 0))
    .then(selectExport("custom", 0, 2)).then(authorize(true).fail())
  run oversizedAxisRejectedTest = init.then(preview(false, 0))
    .then(selectExport("custom", 9, 1)).then(authorize(true).fail())
  run oversizedAreaRejectedTest = init.then(preview(false, 0))
    .then(selectExport("custom", 8, 8)).then(authorize(true).fail())
  run exportRequiresExplicitActionTest = initExport.then(authorize(false).fail())
  run sourceChangeRejectsAndInvalidatesTest = initExport.then(changeSource)
    .then(authorize(true)).then(loadBuffers).then(checkBuffers)
    .expect(safety and changedInputRejected and not(s.accepted) and s.writes.length() == 0)
    .then(authorize(true).fail())
  run maskChangeRejectsAndInvalidatesTest = initExport.then(changeMask)
    .then(authorize(true)).then(loadBuffers).then(checkBuffers)
    .expect(safety and changedInputRejected and not(s.accepted) and s.writes.length() == 0)
    .then(authorize(true).fail())
  run unselectedMaskChangeIsIrrelevantTest = init.then(preview(false, 0)).then(changeMask)
    .then(authorize(true)).then(loadBuffers).then(checkBuffers).then(decodeBuffers).then(publish)
    .expect(safety and s.writes.length() == 1 and s.mismatches == 0)
  run loadedSourceRemainsStableTest = initExport.then(authorize(true)).then(loadBuffers)
    .then(changeSource).then(checkBuffers).then(decodeBuffers).then(publish)
    .expect(safety and capturedBeforeDiskChange and s.writes.nth(0).source == 0)
  run checkedMaskRemainsStableTest = initExport.then(authorize(true)).then(loadBuffers)
    .then(checkBuffers).then(changeMask).then(decodeBuffers).then(publish)
    .expect(safety and s.writes.nth(0).mask == 0 and s.mask == 1)
  run freshPreviewRecoversTest = initExport.then(changeSource).then(authorize(true))
    .then(loadBuffers).then(checkBuffers).then(preview(true, 1))
    .then(authorize(true)).then(loadBuffers).then(checkBuffers).then(decodeBuffers).then(publish)
    .expect(safety and s.writes.length() == 1 and s.writes.nth(0).source == 1)
  run cannotSkipBufferCheckTest = initExport.then(authorize(true)).then(loadBuffers)
    .then(decodeBuffers.fail())
  run cannotPublishBeforeDecodeTest = initExport.then(authorize(true)).then(loadBuffers)
    .then(checkBuffers).then(publish.fail())
  run previewsCannotPublishTest = init.then(preview(true, 1)).expect(safety and s.writes.length() == 0)
    .then(publish.fail())
}
