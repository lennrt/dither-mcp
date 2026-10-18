// Names, exact recipes and source inputs are opaque finite identities.
// Filesystem publication is abstract atomic no-clobber; see publication.qnt.
module favorites {
  pure val Names = Set(0, 1)
  pure val Recipes = Set(0, 1, 2)
  type Request = { kind: str, name: int, recipe: int, revision: int, epoch: int }
  type State = {
    disk: int -> int, original: int -> int,
    draft: int, accepted: int, fresh: bool, revision: int, epoch: int,
    pending: int, next: int, requests: int -> Request, phases: int -> str,
    reports: int, badReports: int, badAdoptions: int,
    listed: int, loaded: int, rejected: int, restarted: bool
  }
  var s: State
  action init = s' = {
    disk: Names.mapBy(_ => -1), original: Names.mapBy(_ => -1),
    draft: 0, accepted: 0, fresh: true, revision: 0, epoch: 0,
    pending: -1, next: 0, requests: 0.to(5).mapBy(_ => { kind: "", name: 0, recipe: -1, revision: -1, epoch: -1 }), phases: 0.to(5).mapBy(_ => "new"),
    reports: 0, badReports: 0, badAdoptions: 0,
    listed: 0, loaded: 0, rejected: 0, restarted: false
  }
  val writing = s.pending != -1 and s.requests.get(s.pending).kind == "save"
  pure def current(id, request, state) = id == state.pending and request.revision == state.revision and request.epoch == state.epoch

  action beginSave(name) = all {
    Names.contains(name), s.pending == -1, s.next < 6,
    s.fresh, s.draft == s.accepted,
    s' = { pending: s.next, next: s.next + 1,
      requests: s.requests.set(s.next, { kind: "save", name: name, recipe: s.accepted, revision: s.revision, epoch: s.epoch }),
      phases: s.phases.set(s.next, "write"), ...s }
  }
  action finishSave(id, receiptMatches) = {
    val request = s.requests.get(id)
    val success = s.disk.get(request.name) == -1
    val report = success and receiptMatches
    all {
      id == s.pending, request.kind == "save", s.phases.get(id) == "write",
      s' = { disk: if (success) s.disk.set(request.name, request.recipe) else s.disk,
        original: if (s.original.get(request.name) == -1 and success) s.original.set(request.name, request.recipe) else s.original,
        reports: s.reports + (if (report) 1 else 0),
        badReports: s.badReports + (if (report and not(receiptMatches)) 1 else 0),
        pending: -1, phases: s.phases.set(id, "done"), ...s }
    }
  }
  // Another process uses the same create-only store independently of studio state.
  action apiSave(name, recipe) = all {
    Names.contains(name), Recipes.contains(recipe), s.disk.get(name) == -1,
    s' = { disk: s.disk.set(name, recipe), original: if (s.original.get(name) == -1) s.original.set(name, recipe) else s.original, ...s }
  }
  action beginLoad(name) = all {
    Names.contains(name), s.pending == -1, s.next < 6,
    s' = { pending: s.next, next: s.next + 1, fresh: false,
      requests: s.requests.set(s.next, { kind: "load", name: name, recipe: -1, revision: s.revision, epoch: s.epoch }),
      phases: s.phases.set(s.next, "read"), ...s }
  }
  // valid abstracts name/ID/envelope/recipe and mask compatibility checks.
  action receiveFavorite(id, valid) = {
    val request = s.requests.get(id)
    val active = current(id, request, s)
    val usable = active and valid and s.disk.get(request.name) != -1
    all {
      request.kind == "load", s.phases.get(id) == "read",
      s' = { requests: if (usable) s.requests.set(id, { recipe: s.disk.get(request.name), ...request }) else s.requests,
        phases: s.phases.set(id, if (usable) "preview" else "done"),
        pending: if (id == s.pending and not(usable)) -1 else s.pending,
        rejected: s.rejected + (if (not(usable)) 1 else 0), ...s }
    }
  }
  action beginRandom(recipe) = all {
    Recipes.contains(recipe), not(writing), s.next < 6,
    s' = { pending: s.next, next: s.next + 1, fresh: false,
      requests: s.requests.set(s.next, { kind: "random", name: -1, recipe: recipe, revision: s.revision, epoch: s.epoch }),
      phases: s.phases.set(s.next, "preview"), ...s }
  }
  action receivePreview(id, recipe) = {
    val request = s.requests.get(id)
    val active = current(id, request, s)
    val adopt = active and recipe == request.recipe
    all {
      Recipes.contains(recipe), s.phases.get(id) == "preview",
      s' = { draft: if (adopt) recipe else s.draft, accepted: if (adopt) recipe else s.accepted,
        fresh: if (active) adopt else s.fresh,
        pending: if (id == s.pending) -1 else s.pending,
        phases: s.phases.set(id, "done"),
        loaded: s.loaded + (if (adopt and request.kind == "load") 1 else 0),
        rejected: s.rejected + (if (not(adopt)) 1 else 0),
        badAdoptions: s.badAdoptions + (if (adopt and (id != s.pending or request.revision != s.revision or request.epoch != s.epoch or recipe != request.recipe)) 1 else 0), ...s }
    }
  }
  action edit(recipe) = all {
    Recipes.contains(recipe), not(writing), s.revision < 3,
    s' = { draft: recipe, revision: s.revision + 1, fresh: false, ...s }
  }
  action cancel = all {
    s.pending != -1, not(writing), s' = { pending: -1, fresh: false, ...s }
  }
  action list = all { s.listed < 2, s' = { listed: s.listed + 1, ...s } }
  action restart = all {
    not(writing), s.epoch < 2,
    s' = { epoch: s.epoch + 1, revision: 0, draft: 0, accepted: -1,
      fresh: false, pending: -1, restarted: true, ...s }
  }
  action stutter = s' = s
  action step = {
    nondet name = Names.oneOf()
    nondet recipe = Recipes.oneOf()
    nondet valid = Set(true, false).oneOf()
    nondet id = 0.to(5).oneOf()
    any { beginSave(name), if (s.requests.keys().contains(id)) finishSave(id, valid) else all { false, s' = s },
      apiSave(name, recipe), beginLoad(name),
      if (s.requests.keys().contains(id)) receiveFavorite(id, valid) else all { false, s' = s },
      beginRandom(recipe), if (s.requests.keys().contains(id)) receivePreview(id, recipe) else all { false, s' = s },
      edit(recipe), cancel, list, restart, stutter }
  }
  val immutable = s.disk == s.original
  val authority = not(s.fresh) or (s.accepted == s.draft and Recipes.contains(s.accepted))
  val safety = all { immutable, authority, s.badAdoptions == 0, s.badReports == 0 }
  val persistent = s.restarted and Names.exists(n => s.disk.get(n) != -1)
  val favoriteLoaded = s.loaded > 0
  val receiptReported = s.reports > 0
  val staleRejected = s.rejected > 0

  run saveThenRestartLoadTest = init.then(beginSave(0)).then(finishSave(0, true))
    .then(restart).expect(s.disk.get(0) == 0 and not(s.fresh))
    .then(list).then(beginLoad(0)).then(receiveFavorite(1, true))
    .expect(not(s.fresh)).then(receivePreview(1, 0)).expect(safety and favoriteLoaded and s.fresh)
  run duplicateSavePreservesOriginalTest = init.then(apiSave(0, 2)).then(beginSave(0))
    .then(finishSave(0, true)).expect(safety and s.disk.get(0) == 2 and s.reports == 0)
  run concurrentSaveOneWinnerTest = init.then(beginSave(0)).then(apiSave(0, 1))
    .then(finishSave(0, true)).expect(safety and s.disk.get(0) == 1)
  run dirtyCannotSaveTest = init.then(edit(1)).then(beginSave(0).fail())
  run listPreservesAuthorityTest = init.then(list).expect(safety and s.fresh and s.accepted == 0 and s.disk == Names.mapBy(_ => -1))
  run malformedReceiptStillPublishedTest = init.then(beginSave(0)).then(finishSave(0, false))
    .expect(safety and s.disk.get(0) == 0 and s.reports == 0)
  run loadRequiresPreviewTest = init.then(apiSave(0, 1)).then(beginLoad(0)).then(receiveFavorite(0, true))
    .expect(s.draft == 0 and not(s.fresh)).then(beginSave(1).fail())
  run wrongPreviewRejectedTest = init.then(apiSave(0, 1)).then(beginLoad(0)).then(receiveFavorite(0, true))
    .then(receivePreview(0, 2)).expect(safety and s.draft == 0 and not(s.fresh))
  run invalidFavoritePreservesEditsTest = init.then(edit(2)).then(beginLoad(0)).then(receiveFavorite(0, false))
    .expect(safety and s.draft == 2 and s.pending == -1 and not(s.fresh))
  run editedReadCanRetryTest = init.then(apiSave(0, 1)).then(beginLoad(0)).then(edit(2))
    .then(receiveFavorite(0, true)).expect(safety and s.draft == 2 and s.pending == -1 and not(s.fresh))
    .then(beginLoad(0)).then(receiveFavorite(1, true)).then(receivePreview(1, 1)).expect(safety and s.fresh and s.draft == 1)
  run cannotEditDuringWriteTest = init.then(beginSave(0)).then(edit(1).fail())
  run canceledReadIgnoredTest = init.then(apiSave(0, 1)).then(beginLoad(0)).then(cancel).then(receiveFavorite(0, true))
    .expect(safety and s.draft == 0 and s.pending == -1 and not(s.fresh))
  run canceledPreviewIgnoredTest = init.then(apiSave(0, 1)).then(beginLoad(0)).then(receiveFavorite(0, true))
    .then(cancel).then(receivePreview(0, 1)).expect(safety and s.draft == 0 and not(s.fresh))
  run editedPreviewIgnoredTest = init.then(apiSave(0, 1)).then(beginLoad(0)).then(receiveFavorite(0, true))
    .then(edit(2)).then(receivePreview(0, 1)).expect(safety and s.draft == 2 and not(s.fresh))
  run supersededPreviewIgnoredTest = init.then(apiSave(0, 1)).then(beginLoad(0)).then(receiveFavorite(0, true))
    .then(beginRandom(2)).then(receivePreview(1, 2)).then(receivePreview(0, 1))
    .expect(safety and s.fresh and s.draft == 2 and s.loaded == 0)
  run restartRejectsOldReadTest = init.then(apiSave(0, 1)).then(beginLoad(0)).then(restart).then(receiveFavorite(0, true))
    .expect(safety and s.pending == -1 and not(s.fresh))
  run writeCannotBeSupersededTest = init.then(beginSave(0)).then(beginRandom(2).fail())
  run concurrentApiCannotReplaceTest = init.then(apiSave(0, 1)).then(apiSave(0, 2).fail())

  action initReload = init.then(apiSave(0, 1)).then(restart)
  action reloadStep = any {
    if (s.next == 0) beginLoad(0) else all { false, s' = s },
    if (s.next > 0 and s.phases.get(0) == "read") receiveFavorite(0, true) else all { false, s' = s },
    if (s.next > 0 and s.phases.get(0) == "preview") receivePreview(0, 1) else all { false, s' = s },
    if (s.loaded > 0) list else all { false, s' = s }, stutter
  }
}
