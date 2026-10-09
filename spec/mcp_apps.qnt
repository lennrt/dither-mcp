// Revisions abstract complete source, palette, options, and mask settings.
// Requests may finish after cancellation or an edit. Only current results count.
// Publication details belong to publication.qnt and concrete service tests.
module mcp_apps {
  pure val Requests = 0.to(2)
  type Authorization = {
    current: int, preview: int, recipe: int,
    explicit: bool, output: bool, tools: bool
  }
  type State = {
    revision: int, connected: bool, tools: bool, output: bool,
    next: int, pending: int, inFlight: Set[int], requested: int -> int,
    accepted: int, acceptedRevision: int, recipe: int,
    saving: bool, authorizations: List[Authorization], writes: int,
    discarded: int, cancellations: int, failures: int
  }
  var s: State
  action init = s' = {
    revision: 0, connected: false, tools: false, output: false,
    next: 0, pending: -1, inFlight: Set(),
    requested: Map(0 -> -1, 1 -> -1, 2 -> -1),
    accepted: -1, acceptedRevision: -1, recipe: -1,
    saving: false, authorizations: List(), writes: 0,
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
      s.inFlight.contains(id),
      s' = {
        inFlight: s.inFlight.exclude(Set(id)),
        pending: if (id == s.pending) -1 else s.pending,
        accepted: if (accept) id else s.accepted,
        acceptedRevision: if (accept) s.requested.get(id) else s.acceptedRevision,
        recipe: if (accept) s.requested.get(id) else s.recipe,
        discarded: s.discarded + (if (not(current)) 1 else 0),
        failures: s.failures + (if (current and not(success)) 1 else 0),
        ...s
      }
    }
  }
  action cancel = all {
    s.pending >= 0,
    s' = { pending: -1, accepted: -1, acceptedRevision: -1, recipe: -1,
      cancellations: s.cancellations + 1, ...s }
  }
  val canSave = all {
    s.connected, s.tools, s.output, not(s.saving), s.pending == -1,
    s.authorizations.length() < 3,
    s.accepted >= 0, s.acceptedRevision == s.revision
  }
  action save(explicit) = all {
    canSave, explicit,
    s' = {
      saving: true,
      authorizations: s.authorizations.append({
        current: s.revision, preview: s.acceptedRevision, recipe: s.recipe,
        explicit: explicit, output: s.output, tools: s.tools
      }), ...s
    }
  }
  action finishSave(success) = all {
    s.saving,
    s' = { saving: false, writes: s.writes + (if (success) 1 else 0), ...s }
  }
  action stutter = s' = s
  action step = {
    nondet id = Requests.oneOf()
    nondet flag = Set(false, true).oneOf()
    any {
      connect(flag), edit, enterOutput(flag), preview, receive(id, flag),
      cancel, save(flag), finishSave(flag), stutter
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

  val requestProvenance = s.accepted >= 0 implies all {
    s.accepted < s.next,
    s.acceptedRevision == s.requested.get(s.accepted),
    s.recipe == s.acceptedRevision
  }
  val explicitCurrentSaves = s.authorizations.indices().forall(i => {
    val a = s.authorizations.nth(i)
    all { a.current == a.preview, a.recipe == a.preview,
      a.explicit, a.output, a.tools }
  })
  val noPreviewWrites = s.writes <= s.authorizations.length()
  val dirtyCannotSave = s.acceptedRevision != s.revision implies not(canSave)
  val safety = all {
    s.revision >= 0, s.revision <= 3, s.next >= 0, s.next <= 3,
    s.authorizations.length() <= 3,
    s.inFlight.subseteq(Requests),
    s.pending >= 0 implies s.inFlight.contains(s.pending),
    requestProvenance, explicitCurrentSaves, noPreviewWrites, dirtyCannotSave,
    s.saving implies s.authorizations.length() > s.writes
  }
  val previewAccepted = s.accepted >= 0
  val staleDiscarded = s.discarded > 0
  val canceledRequest = s.cancellations > 0
  val savedCurrent = s.writes > 0

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
}
