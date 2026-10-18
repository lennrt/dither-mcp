// Complete canonical requests are opaque look IDs. There are four possible
// looks and three MRU/history/navigation slots instead of the real twenty.
// receive(success=true) abstracts exact canonical-response validation.
// This models restoration authority and success-only navigation, not rendering,
// request JSON, pixel data, host implementation, or source-byte snapshots.
module style_history {
  pure val Looks = 0.to(3)
  pure val Requests = 0.to(5)
  type Request = { look: int, mode: str, revision: int, epoch: int }
  type Authorization = { look: int, accepted: int, receipt: int }
  type State = {
    history: List[int], undo: List[int], redo: List[int], active: int,
    tools: bool, revision: int, epoch: int, fresh: bool,
    pending: int, next: int, inFlight: Set[int], requests: int -> Request,
    accepted: int, receipt: int, saving: bool,
    authorizations: List[Authorization], restored: int, discarded: int
  }
  var s: State
  pure def bounded(items) = items.slice(0, if (items.length() < 3) items.length() else 3)
  pure def push(items, look) = if (look < 0) items else List(look).concat(items)
  pure def member(items, id) = items.indices().exists(i => items.nth(i) == id)
  pure def retained(items, history, active) = {
    val kept = items.select(id => member(history, id)).foldl(List(), (acc, id) =>
      if (acc.length() == 0 or acc.nth(acc.length() - 1) != id) acc.append(id) else acc)
    bounded(if (first(kept) == active) kept.slice(1, kept.length()) else kept)
  }
  pure def first(items) = if (items.length() > 0) items.nth(0) else -1
  action init = s' = {
    history: List(), undo: List(), redo: List(), active: -1,
    tools: true, revision: 0, epoch: 0, fresh: false,
    pending: -1, next: 0, inFlight: Set(),
    requests: Requests.mapBy(id => { look: -1, mode: "", revision: -1, epoch: -1 }),
    accepted: -1, receipt: -1, saving: false,
    authorizations: List(), restored: 0, discarded: 0
  }
  action setTools(value) = s' = { tools: value, ...s }
  action edit = all {
    not(s.saving), s.revision < 2,
    s' = { revision: s.revision + 1, fresh: false, ...s }
  }
  action begin(look, mode) = all {
    s.tools, not(s.saving), s.next < 6, Looks.contains(look),
    Set("new", "select", "undo", "redo").contains(mode),
    mode == "new" or (s.pending == -1 and member(s.history, look)),
    mode != "undo" or first(s.undo) == look,
    mode != "redo" or first(s.redo) == look,
    s' = {
      pending: s.next, next: s.next + 1, fresh: false,
      inFlight: s.inFlight.union(Set(s.next)),
      requests: s.requests.set(s.next, { look: look, mode: mode, revision: s.revision, epoch: s.epoch }), ...s
    }
  }
  action receive(id, success) = {
    val request = s.requests.get(id)
    val current = id == s.pending and request.revision == s.revision and request.epoch == s.epoch
    val adopt = current and success
    val changed = request.look != s.active
    val history = bounded(List(request.look).concat(s.history.select(look => look != request.look)))
    val undo = if (not(changed)) s.undo
      else if (request.mode == "undo") s.undo.slice(1, s.undo.length())
      else push(s.undo, s.active)
    val redo = if (not(changed)) s.redo
      else if (request.mode == "undo") push(s.redo, s.active)
      else if (request.mode == "redo") s.redo.slice(1, s.redo.length())
      else List()
    all {
      s.inFlight.contains(id),
      s' = {
        pending: if (id == s.pending) -1 else s.pending,
        inFlight: s.inFlight.exclude(Set(id)),
        history: if (adopt) history else s.history,
        undo: if (adopt) retained(undo, history, request.look) else s.undo,
        redo: if (adopt) retained(redo, history, request.look) else s.redo,
        active: if (adopt) request.look else s.active,
        accepted: if (adopt) request.look else s.accepted,
        receipt: if (adopt) id else s.receipt,
        fresh: if (current) success else s.fresh,
        restored: s.restored + (if (adopt and request.mode != "new") 1 else 0),
        discarded: s.discarded + (if (not(current)) 1 else 0), ...s
      }
    }
  }
  action cancel = all { s.pending >= 0, s' = { pending: -1, fresh: false, ...s } }
  action reset = all {
    not(s.saving), s.epoch < 1,
    s' = { history: List(), undo: List(), redo: List(), active: -1,
      pending: -1, accepted: -1, receipt: -1, fresh: false, epoch: s.epoch + 1, ...s }
  }
  val canSave = s.tools and s.fresh and s.pending == -1 and not(s.saving) and s.active >= 0 and s.authorizations.length() < 3
  action save = all {
    canSave,
    s' = { saving: true, authorizations: s.authorizations.append({ look: s.active, accepted: s.accepted, receipt: s.receipt }), ...s }
  }
  action finishSave = all { s.saving, s' = { saving: false, ...s } }
  action stutter = s' = s
  action step = {
    nondet look = Looks.oneOf()
    nondet id = Requests.oneOf()
    nondet flag = Set(false, true).oneOf()
    nondet mode = Set("new", "select", "undo", "redo").oneOf()
    any { begin(look, mode), receive(id, flag), edit, cancel, reset, setTools(flag), save, finishSave, stutter }
  }
  val safety = all {
    s.history.length() <= 3, s.undo.length() <= 3, s.redo.length() <= 3,
    s.history.indices().forall(i => s.history.indices().forall(j => i == j or s.history.nth(i) != s.history.nth(j))),
    s.history.indices().forall(i => Looks.contains(s.history.nth(i))),
    s.undo.indices().forall(i => member(s.history, s.undo.nth(i))),
    s.redo.indices().forall(i => member(s.history, s.redo.nth(i))),
    s.undo.length() > 0 implies first(s.undo) != s.active,
    s.redo.length() > 0 implies first(s.redo) != s.active,
    s.undo.indices().forall(i => i == 0 or s.undo.nth(i) != s.undo.nth(i - 1)),
    s.redo.indices().forall(i => i == 0 or s.redo.nth(i) != s.redo.nth(i - 1)),
    s.active >= 0 implies first(s.history) == s.active,
    s.fresh implies (s.active == s.accepted and s.receipt >= 0),
    s.pending >= 0 implies (not(s.fresh) and s.inFlight.contains(s.pending)),
    s.authorizations.indices().forall(i => {
      val a = s.authorizations.nth(i)
      a.look == a.accepted and a.receipt >= 0
    })
  }
  val hasHistory = s.history.length() >= 2
  val historyRestored = s.restored > 0
  val staleIgnored = s.discarded > 0
  val historySaved = s.authorizations.length() > 0 and s.restored > 0

  action acceptLook(look) = begin(look, "new").then(receive(s.pending, true))
  action initNavigation = init.then(acceptLook(0)).then(acceptLook(1))
  action historyStep = any {
    all { s.next == 2, begin(0, "undo") },
    all { s.pending == 2, receive(2, true) },
    all { s.restored > 0, save }, finishSave, stutter
  }

  run startingLookAndRandomAreRecoverableTest = initNavigation
    .expect(safety and s.history == List(1, 0) and s.undo == List(0))
  run undoRedoUseFreshPreviewTest = initNavigation.then(begin(0, "undo"))
    .expect(safety and s.active == 1 and not(canSave) and s.undo == List(0))
    .then(receive(2, true)).expect(safety and s.active == 0 and s.redo == List(1))
    .then(begin(1, "redo")).then(receive(3, true))
    .expect(safety and s.active == 1 and s.undo == List(0) and s.redo == List())
  run branchKeepsDiscoveriesTest = initNavigation.then(acceptLook(2))
    .then(begin(0, "select")).then(receive(3, true)).then(acceptLook(3))
    .expect(safety and s.history == List(3, 0, 2) and first(s.undo) == 0 and s.redo == List())
    .then(begin(0, "undo")).then(receive(5, true)).expect(safety and s.active == 0)
  run sameLookPreservesRedoTest = initNavigation.then(begin(0, "undo")).then(receive(2, true))
    .then(begin(0, "select")).then(receive(3, true))
    .expect(safety and s.history == List(0, 1) and s.undo == List() and s.redo == List(1))
  run capacityPrunesNavigationTest = initNavigation.then(acceptLook(2)).then(acceptLook(3))
    .expect(safety and s.history == List(3, 2, 1) and s.undo == List(2, 1))
  run failedRestoreKeepsNavigationTest = initNavigation.then(begin(0, "undo")).then(receive(2, false))
    .expect(safety and s.active == 1 and s.history == List(1, 0) and s.undo == List(0) and not(canSave))
  run canceledRestoreCannotAdoptTest = initNavigation.then(begin(0, "undo")).then(cancel).then(receive(2, true))
    .expect(safety and s.active == 1 and s.undo == List(0) and not(canSave) and staleIgnored)
  run editedRestoreCannotAdoptTest = initNavigation.then(begin(0, "undo")).then(edit).then(receive(2, true))
    .expect(safety and s.active == 1 and s.undo == List(0) and not(canSave))
  run randomSupersedesRestoreTest = initNavigation.then(begin(0, "undo")).then(acceptLook(2)).then(receive(2, true))
    .expect(safety and s.active == 2 and s.history == List(2, 1, 0) and s.undo == List(1, 0))
  run historyWaitsForPendingReadTest = initNavigation.then(begin(0, "undo")).then(begin(1, "select").fail())
  run historyCannotInterruptWriteTest = initNavigation.then(save).then(begin(0, "undo").fail())
  run historyRequiresToolsTest = initNavigation.then(setTools(false)).then(begin(0, "undo").fail())
  run newInputClearsAndRejectsOldRestoreTest = initNavigation.then(begin(0, "undo")).then(reset).then(receive(2, true))
    .expect(safety and s.history == List() and s.undo == List() and s.redo == List() and s.active == -1 and not(canSave))
  run restoredLookAuthorizesItsOwnSaveTest = initNavigation.then(begin(0, "undo")).then(receive(2, true)).then(save)
    .expect(safety and historySaved and s.authorizations.nth(0).look == 0 and s.authorizations.nth(0).receipt == 2)
}
