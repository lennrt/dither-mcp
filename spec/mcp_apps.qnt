// Revisions abstract user edits to source, palette, options, and mask settings.
// Generated style identity is recorded separately in accepted save authority.
// Requests may finish after cancellation or an edit. Only current results count.
// Publication details belong to publication.qnt and concrete service tests.
// A recipe may be written despite a malformed receipt; success reporting is separate.
// Extraction success abstracts a valid palette with at least two unique colors.
// It changes only the draft palette; it cannot accept a preview or publish.
// Random generation and its preview share an operation identity; only random
// may supersede pending reads. Selection values are abstracted by that identity.
module mcp_apps {
  pure val Requests = 0.to(2)
  type Authorization = {
    current: int, preview: int, recipe: int,
    explicit: bool, output: bool, tools: bool, kind: str, width: int,
    random: bool, previewId: int, style: int
  }
  type State = {
    revision: int, connected: bool, tools: bool, output: bool, recipeOutput: bool, exportWidth: int,
    next: int, pending: int, inFlight: Set[int], requested: int -> int,
    extractions: Set[int], palette: int, styles: Set[int], generated: Set[int], style: int,
    accepted: int, acceptedRevision: int, recipe: int,
    saving: bool, saveKind: str, authorizations: List[Authorization], writes: int, recipeWrites: int, recipeReports: int,
    discarded: int, cancellations: int, failures: int
  }
  var s: State
  action init = s' = {
    revision: 0, connected: false, tools: false, output: false, recipeOutput: false, exportWidth: 2,
    next: 0, pending: -1, inFlight: Set(),
    extractions: Set(), palette: -1, styles: Set(), generated: Set(), style: -1,
    requested: Map(0 -> -1, 1 -> -1, 2 -> -1),
    accepted: -1, acceptedRevision: -1, recipe: -1,
    saving: false, saveKind: "", authorizations: List(), writes: 0, recipeWrites: 0, recipeReports: 0,
    discarded: 0, cancellations: 0, failures: 0
  }
  action connect(tools) = all {
    not(s.connected), s' = { connected: true, tools: tools, ...s }
  }
  action edit = all {
    not(s.saving), s.revision < 3,
    s' = { revision: s.revision + 1, ...s }
  }
  action enterOutput(present) = all {
    not(s.saving), s' = { output: present, ...s }
  }
  action enterRecipeOutput(present) = all {
    not(s.saving), s' = { recipeOutput: present, ...s }
  }
  action chooseExportWidth(width) = all {
    not(s.saving), width >= 1, width <= 3, s' = { exportWidth: width, ...s }
  }
  action preview = all {
    s.connected, s.tools, s.pending == -1, not(s.saving), s.next < 3,
    s' = {
      pending: s.next, next: s.next + 1,
      inFlight: s.inFlight.union(Set(s.next)),
      requested: s.requested.set(s.next, s.revision),
      accepted: -1, acceptedRevision: -1, recipe: -1, ...s
    }
  }
  action receive(id, success) = {
    val current = id == s.pending and s.requested.get(id) == s.revision
    val accept = current and success
    all {
      s.inFlight.contains(id), not(s.extractions.contains(id)),
      not(s.styles.contains(id)) or s.generated.contains(id),
      s' = {
        inFlight: s.inFlight.exclude(Set(id)),
        pending: if (id == s.pending) -1 else s.pending,
        accepted: if (accept) id else s.accepted,
        acceptedRevision: if (accept) s.requested.get(id) else s.acceptedRevision,
        recipe: if (accept) s.requested.get(id) else s.recipe,
        style: if (accept and s.styles.contains(id)) id else s.style,
        discarded: s.discarded + (if (not(current)) 1 else 0),
        failures: s.failures + (if (current and not(success)) 1 else 0),
        ...s
      }
    }
  }
  action extract = all {
    s.connected, s.tools, s.pending == -1, not(s.saving), s.next < 3,
    s.revision < 3,
    s' = {
      pending: s.next, next: s.next + 1,
      inFlight: s.inFlight.union(Set(s.next)),
      extractions: s.extractions.union(Set(s.next)),
      requested: s.requested.set(s.next, s.revision),
      accepted: -1, acceptedRevision: -1, recipe: -1, ...s
    }
  }
  action receiveExtraction(id, success) = {
    val extractionCurrent = id == s.pending and s.requested.get(id) == s.revision
    val adopt = extractionCurrent and success
    all {
      s.inFlight.contains(id), s.extractions.contains(id),
      s' = {
        inFlight: s.inFlight.exclude(Set(id)),
        pending: if (id == s.pending) -1 else s.pending,
        revision: if (adopt) s.revision + 1 else s.revision,
        palette: if (adopt) id else s.palette,
        discarded: s.discarded + (if (not(extractionCurrent)) 1 else 0),
        failures: s.failures + (if (extractionCurrent and not(success)) 1 else 0),
        ...s
      }
    }
  }
  action random = all {
    s.connected, s.tools, not(s.saving), s.next < 3,
    s' = {
      pending: s.next, next: s.next + 1,
      inFlight: s.inFlight.union(Set(s.next)),
      styles: s.styles.union(Set(s.next)),
      requested: s.requested.set(s.next, s.revision),
      accepted: -1, acceptedRevision: -1, recipe: -1, ...s
    }
  }
  action receiveStyle(id, success) = {
    val styleCurrent = id == s.pending and s.requested.get(id) == s.revision
    val ready = styleCurrent and success
    all {
      s.inFlight.contains(id), s.styles.contains(id), not(s.generated.contains(id)),
      s' = {
        inFlight: if (ready) s.inFlight else s.inFlight.exclude(Set(id)),
        pending: if (id == s.pending and not(ready)) -1 else s.pending,
        generated: if (ready) s.generated.union(Set(id)) else s.generated,
        discarded: s.discarded + (if (not(styleCurrent)) 1 else 0),
        failures: s.failures + (if (styleCurrent and not(success)) 1 else 0), ...s
      }
    }
  }
  action cancel = all {
    s.pending >= 0,
    s' = { pending: -1, accepted: -1, acceptedRevision: -1, recipe: -1,
      cancellations: s.cancellations + 1, ...s }
  }
  val saveSettingsReady = all {
    s.connected, s.tools, not(s.saving), s.pending == -1,
    s.authorizations.length() < 3,
    s.accepted >= 0, s.acceptedRevision == s.revision
  }
  val canSave = s.output and saveSettingsReady
  val canSaveRecipe = s.recipeOutput and saveSettingsReady
  action save(explicit) = all {
    canSave, explicit,
    s' = {
      saving: true, saveKind: "image",
      authorizations: s.authorizations.append({
        current: s.revision, preview: s.acceptedRevision, recipe: s.recipe,
        random: s.styles.contains(s.accepted), previewId: s.accepted, style: s.style,
        explicit: explicit, output: s.output, tools: s.tools, kind: "image", width: s.exportWidth
      }), ...s
    }
  }
  action saveRecipe(explicit) = all {
    canSaveRecipe, explicit,
    s' = {
      saving: true, saveKind: "recipe",
      authorizations: s.authorizations.append({
        current: s.revision, preview: s.acceptedRevision, recipe: s.recipe,
        random: s.styles.contains(s.accepted), previewId: s.accepted, style: s.style,
        explicit: explicit, output: s.recipeOutput, tools: s.tools, kind: "recipe", width: 2
      }), ...s
    }
  }
  action finishRecipeSave(success, receiptMatches) = all {
    s.saving, s.saveKind == "recipe",
    s' = { saving: false,
      writes: s.writes + (if (success) 1 else 0),
      recipeWrites: s.recipeWrites + (if (success) 1 else 0),
      recipeReports: s.recipeReports + (if (success and receiptMatches) 1 else 0), ...s }
  }
  action finishSave(success) = all {
    s.saving, s.saveKind == "image",
    s' = { saving: false, writes: s.writes + (if (success) 1 else 0), ...s }
  }
  action stutter = s' = s
  action step = {
    nondet id = Requests.oneOf()
    nondet flag = Set(false, true).oneOf()
    any {
      connect(flag), edit, enterOutput(flag), enterRecipeOutput(flag), chooseExportWidth(id + 1), preview, receive(id, flag),
      extract, receiveExtraction(id, flag), random, receiveStyle(id, flag),
      cancel, save(flag), saveRecipe(flag), finishSave(flag), finishRecipeSave(flag, true), finishRecipeSave(flag, false), stutter
    }
  }

  // This schedule deliberately reaches a preview and a successful save.
  action initSave = init.then(connect(true)).then(enterOutput(true))
  action saveStep = any {
    all { s.next == 0, preview },
    all { s.pending == 0, receive(0, true) },
    all { s.authorizations.length() == 0, save(true) },
    finishSave(true), stutter
  }

  action initRecipeSave = init.then(connect(true)).then(enterRecipeOutput(true))
  action recipeSaveStep = any {
    all { s.next == 0, preview },
    all { s.pending == 0, receive(0, true) },
    all { s.authorizations.length() == 0, saveRecipe(true) },
    finishRecipeSave(true, true), stutter
  }

  val requestProvenance = s.accepted >= 0 implies all {
    s.accepted < s.next,
    not(s.extractions.contains(s.accepted)),
    s.styles.contains(s.accepted) implies (s.generated.contains(s.accepted) and s.style == s.accepted),
    s.acceptedRevision == s.requested.get(s.accepted),
    s.recipe == s.acceptedRevision
  }
  val explicitCurrentSaves = s.authorizations.indices().forall(i => {
    val a = s.authorizations.nth(i)
    all { a.current == a.preview, a.recipe == a.preview,
      a.explicit, a.output, a.tools,
      a.kind == "recipe" implies a.width == 2,
      a.random implies a.style == a.previewId }
  })
  val noPreviewWrites = s.writes <= s.authorizations.length()
  val dirtyCannotSave = s.acceptedRevision != s.revision implies (not(canSave) and not(canSaveRecipe))
  val safety = all {
    s.revision >= 0, s.revision <= 3, s.next >= 0, s.next <= 3,
    s.authorizations.length() <= 3, s.recipeReports <= s.recipeWrites, s.recipeWrites <= s.writes,
    s.inFlight.subseteq(Requests),
    s.pending >= 0 implies s.inFlight.contains(s.pending),
    requestProvenance, explicitCurrentSaves, noPreviewWrites, dirtyCannotSave,
    s.saving implies s.authorizations.length() > s.writes
  }
  val previewAccepted = s.accepted >= 0
  val staleDiscarded = s.discarded > 0
  val canceledRequest = s.cancellations > 0
  val savedCurrent = s.writes > 0
  val recipeSaved = s.recipeReports > 0
  val paletteExtracted = s.palette >= 0
  val randomAccepted = s.style >= 0

  run previewDoesNotPublishTest = init.then(connect(true)).then(preview).then(receive(0, true))
    .expect(safety and previewAccepted and s.writes == 0 and s.authorizations.length() == 0)
  run explicitSaveUsesAcceptedRecipeTest = init.then(connect(true)).then(edit).then(preview)
    .then(receive(0, true)).then(enterOutput(true)).then(save(true)).then(finishSave(true))
    .expect(safety and savedCurrent and s.authorizations.nth(0).recipe == 1)
  run cannotSaveAutomaticallyTest = initSave.then(preview).then(receive(0, true)).then(save(false).fail())
  run cannotSaveWithoutDestinationTest = init.then(connect(true)).then(preview).then(receive(0, true))
    .then(save(true).fail())
  run cannotSaveBeforePreviewTest = initSave.then(save(true).fail())
  run cannotSaveDirtyControlsTest = initSave.then(preview).then(receive(0, true)).then(edit)
    .expect(safety and not(canSave)).then(save(true).fail())
  run editDuringPreviewDiscardsResultTest = initSave.then(preview).then(edit).then(receive(0, true))
    .expect(safety and staleDiscarded and s.accepted == -1 and not(canSave))
  run canceledPreviewCannotBecomeCurrentTest = initSave.then(preview).then(cancel).then(receive(0, true))
    .expect(safety and canceledRequest and staleDiscarded and s.accepted == -1)
  run lateResultCannotReplaceNewPreviewTest = initSave.then(preview).then(cancel).then(edit)
    .then(preview).then(receive(1, true)).then(receive(0, true))
    .expect(safety and s.accepted == 1 and s.recipe == 1 and canSave)
  run failureDoesNotEnableSaveTest = initSave.then(preview).then(receive(0, false))
    .expect(safety and s.failures == 1 and not(canSave) and s.writes == 0)
  run failedReplacementInvalidatesOldPreviewTest = initSave.then(preview).then(receive(0, true))
    .then(preview).then(receive(1, false)).expect(safety and s.accepted == -1 and not(canSave))
  run cannotPreviewWithoutHostToolsTest = init.then(connect(false)).then(preview.fail())
  run cannotPreviewBeforeConnectTest = init.then(preview.fail())
  run cannotSaveWhilePreviewPendingTest = initSave.then(preview).then(save(true).fail())
  run cannotSaveTwiceWhileBusyTest = initSave.then(preview).then(receive(0, true))
    .then(save(true)).then(save(true).fail())
  run failedSaveDoesNotPublishTest = initSave.then(preview).then(receive(0, true))
    .then(save(true)).then(finishSave(false)).expect(safety and s.writes == 0)
  run extractionRequiresPreviewTest = initSave.then(extract).then(receiveExtraction(0, true))
    .expect(safety and paletteExtracted and not(canSave) and s.writes == 0)
    .then(preview).then(receive(1, true))
    .then(save(true)).then(finishSave(true)).expect(safety and savedCurrent)
  run cannotSaveExtractedPaletteWithoutPreviewTest = initSave.then(extract)
    .then(receiveExtraction(0, true)).then(save(true).fail())
  run pendingExtractionCannotSaveTest = initSave.then(preview).then(receive(0, true))
    .then(extract).then(save(true).fail())
  run failedExtractionKeepsPaletteTest = initSave.then(extract).then(receiveExtraction(0, false))
    .expect(safety and s.palette == -1 and s.revision == 0 and not(canSave))
  run editedExtractionCannotChangePaletteTest = initSave.then(extract).then(edit)
    .then(receiveExtraction(0, true)).expect(safety and s.palette == -1 and s.revision == 1 and staleDiscarded)
  run cancelledExtractionCannotChangePaletteTest = initSave.then(extract).then(cancel)
    .then(receiveExtraction(0, true)).expect(safety and s.palette == -1 and s.revision == 0 and not(canSave))
  run staleExtractionCannotReplaceNewPaletteTest = initSave.then(extract).then(cancel)
    .then(extract).then(receiveExtraction(1, true)).then(receiveExtraction(0, true))
    .expect(safety and s.palette == 1 and s.revision == 1 and not(canSave))
  run cannotExtractWithoutHostToolsTest = init.then(connect(false)).then(extract.fail())
  run explicitRecipeUsesPreviewDimensionsTest = initRecipeSave.then(edit).then(preview)
    .then(receive(0, true)).then(chooseExportWidth(3)).then(saveRecipe(true))
    .then(finishRecipeSave(true, true))
    .expect(safety and recipeSaved and s.authorizations.nth(0).recipe == 1 and s.authorizations.nth(0).width == 2)
  run recipeRequiresExplicitActionTest = initRecipeSave.then(preview).then(receive(0, true))
    .then(saveRecipe(false).fail())
  run recipeRequiresOwnDestinationTest = initSave.then(preview).then(receive(0, true))
    .then(saveRecipe(true).fail())
  run recipeCannotSaveDirtySettingsTest = initRecipeSave.then(preview).then(receive(0, true)).then(edit)
    .then(saveRecipe(true).fail())
  run recipeCannotSaveDuringExtractionTest = initRecipeSave.then(preview).then(receive(0, true)).then(extract)
    .then(saveRecipe(true).fail())
  run recipeCannotSaveWithoutToolsTest = init.then(connect(false)).then(enterRecipeOutput(true))
    .then(saveRecipe(true).fail())
  run recipeReceiptMismatchCannotReportSuccessTest = initRecipeSave.then(preview).then(receive(0, true))
    .then(saveRecipe(true)).then(finishRecipeSave(true, false))
    .expect(safety and not(recipeSaved) and s.recipeWrites == 1 and canSaveRecipe)
  run failedRecipeCanRetryTest = initRecipeSave.then(preview).then(receive(0, true))
    .then(saveRecipe(true)).then(finishRecipeSave(false, true)).then(saveRecipe(true))
    .then(finishRecipeSave(true, true)).expect(safety and recipeSaved and s.writes == 1)
  run extractionNeedsPreviewBeforeRecipeTest = initRecipeSave.then(extract).then(receiveExtraction(0, true))
    .expect(safety and not(canSaveRecipe)).then(preview).then(receive(1, true)).then(saveRecipe(true))
    .then(finishRecipeSave(true, true)).expect(safety and recipeSaved)


  run randomRejectsPreviewBeforeGenerationTest = initRecipeSave.then(random).then(receive(0, true).fail())
  run randomNeedsBothStagesTest = initRecipeSave.then(random).then(receiveStyle(0, true)).expect(safety and not(canSaveRecipe) and s.style == -1)
    .then(receive(0, true)).expect(safety and canSaveRecipe and randomAccepted and s.writes == 0)
  run randomSupersedesGenerationTest = initRecipeSave.then(random).then(random)
    .then(receiveStyle(1, true)).then(receive(1, true)).then(receiveStyle(0, true))
    .expect(safety and s.style == 1 and s.accepted == 1 and not(s.generated.contains(0)) and canSaveRecipe)
  run randomRecipeUsesNewestStyleTest = initRecipeSave.then(random).then(receiveStyle(0, true))
    .then(receive(0, true)).then(random).then(receiveStyle(1, true)).then(receive(1, true))
    .then(saveRecipe(true)).then(finishRecipeSave(true, true))
    .expect(safety and recipeSaved and s.authorizations.nth(0).style == 1)
  run randomSupersedesPreviewTest = initRecipeSave.then(random).then(receiveStyle(0, true)).then(random)
    .then(receiveStyle(1, true)).then(receive(1, true)).then(receive(0, true))
    .expect(safety and s.style == 1 and s.accepted == 1 and canSaveRecipe)
  run randomSupersedesExtractionTest = initRecipeSave.then(extract).then(random)
    .then(receiveStyle(1, true)).then(receive(1, true)).then(receiveExtraction(0, true))
    .expect(safety and s.style == 1 and s.palette == -1 and canSaveRecipe)
  run canceledRandomGenerationCannotAdoptTest = initRecipeSave.then(random).then(cancel)
    .then(receiveStyle(0, true)).expect(safety and s.style == -1 and not(canSaveRecipe))
  run editedRandomPreviewCannotAdoptTest = initRecipeSave.then(random).then(receiveStyle(0, true))
    .then(edit).then(receive(0, true)).expect(safety and s.style == -1 and not(canSaveRecipe))
  run failedRandomPreviewKeepsStyleTest = initRecipeSave.then(random).then(receiveStyle(0, true))
    .then(receive(0, true)).then(random).then(receiveStyle(1, true)).then(receive(1, false))
    .expect(safety and s.style == 0 and s.failures == 1 and not(canSaveRecipe))
  run randomCannotSupersedeRecipeWriteTest = initRecipeSave.then(preview).then(receive(0, true))
    .then(saveRecipe(true)).expect(safety).then(random.fail())
  run randomCannotSupersedeImageWriteTest = initSave.then(preview).then(receive(0, true))
    .then(save(true)).expect(safety).then(random.fail())

}
