// This finite contract covers independent output transactions and collisions.
// Paths and byte streams are abstract values. The model does not prove OS or codec code.
module publication {
  pure val Jobs = Set(0, 1)
  pure val Paths = Set(0, 1)
  pure val Stages = Set("new", "admitted", "rendered", "staged", "complete", "cancelled", "failed")
  type State = {
    stage: int -> str, target: int -> int, safe: int -> bool,
    bounded: int -> bool,
    files: int -> int, completeBytes: Set[int], published: Set[int],
    cancelled: Set[int], unauthorizedWrites: int, unsafeWrites: int,
    partialWrites: int, networkRequests: int
  }
  var s: State

  // Path 0 initially contains a file owned by an external caller (token 9).
  action init = s' = {
    stage: Jobs.mapBy(_ => "new"), target: Jobs.mapBy(_ => 0),
    safe: Jobs.mapBy(_ => false), bounded: Jobs.mapBy(_ => false),
    files: Map(0 -> 9, 1 -> -1),
    completeBytes: Set(9), published: Set(), cancelled: Set(),
    unauthorizedWrites: 0, unsafeWrites: 0, partialWrites: 0, networkRequests: 0
  }

  action admit(j, p, safe, bounded) = all {
    s.stage.get(j) == "new", Jobs.contains(j), Paths.contains(p),
    s' = {
      stage: s.stage.set(j, if (safe and bounded) "admitted" else "failed"),
      target: s.target.set(j, p), safe: s.safe.set(j, safe),
      bounded: s.bounded.set(j, bounded), ...s
    }
  }

  action render(j) = all {
    s.stage.get(j) == "admitted",
    s' = { stage: s.stage.set(j, "rendered"), ...s }
  }

  action encode(j) = all {
    s.stage.get(j) == "rendered",
    s' = { stage: s.stage.set(j, "staged"), completeBytes: s.completeBytes.union(Set(j)), ...s }
  }

  action failJob(j) = all {
    Set("admitted", "rendered", "staged").contains(s.stage.get(j)),
    s' = { stage: s.stage.set(j, "failed"), ...s }
  }

  action cancel(j) = all {
    Set("new", "admitted", "rendered", "staged").contains(s.stage.get(j)),
    s' = { stage: s.stage.set(j, "cancelled"), cancelled: s.cancelled.union(Set(j)), ...s }
  }

  // The existence check and file publication are one abstract atomic action.
  // Refinement requires no-clobber Link, not Stat followed by Rename.
  action publish(j) = {
    val p = s.target.get(j)
    all {
      s.stage.get(j) == "staged",
      s.files.get(p) == -1,
      s' = {
        stage: s.stage.set(j, "complete"), files: s.files.set(p, j),
        published: s.published.union(Set(j)),
        unauthorizedWrites: s.unauthorizedWrites +
          (if (s.files.get(p) != -1) 1 else 0),
        unsafeWrites: s.unsafeWrites + (if (not(s.safe.get(j)) or not(s.bounded.get(j))) 1 else 0),
        partialWrites: s.partialWrites + (if (not(s.completeBytes.contains(j))) 1 else 0), ...s
      }
    }
  }

  action collide(j) = all {
    s.stage.get(j) == "staged", s.files.get(s.target.get(j)) != -1,
    s' = { stage: s.stage.set(j, "failed"), ...s }
  }
  action stutter = s' = s
  action step = {
    nondet j = Jobs.oneOf()
    nondet p = Paths.oneOf()
    nondet safe = Set(true, false).oneOf()
    nondet bounded = Set(true, false).oneOf()
    any { admit(j, p, safe, bounded), render(j), encode(j), failJob(j), cancel(j), publish(j), collide(j), stutter }
  }

  val typeOK = all {
    s.stage.keys() == Jobs, s.files.keys() == Paths,
    Jobs.forall(j => Stages.contains(s.stage.get(j))),
    s.published.subseteq(Jobs), s.cancelled.subseteq(Jobs)
  }
  val onlyCompleteVisible = Paths.forall(p => s.files.get(p) == -1 or s.completeBytes.contains(s.files.get(p)))
  val noUnauthorizedOverwrite = s.unauthorizedWrites == 0
  val onlyValidatedWrites = s.unsafeWrites == 0
  val noPartialPublication = s.partialWrites == 0
  val cancellationStopsPublication = s.cancelled.intersect(s.published) == Set()
  val noNetwork = s.networkRequests == 0
  val safety = all { typeOK, onlyCompleteVisible, noUnauthorizedOverwrite, onlyValidatedWrites, noPartialPublication, cancellationStopsPublication, noNetwork }
  val completed = s.published.size() > 0
  val collision = Jobs.exists(j => s.stage.get(j) == "failed" and s.completeBytes.contains(j))
  val cancelledDuringWork = s.cancelled.size() > 0 and s.completeBytes.size() > 1

  run publishCompleteTest = init.then(admit(0, 1, true, true)).then(render(0)).then(encode(0)).then(publish(0))
    .expect(safety and s.files.get(1) == 0)
  run unsafePathRejectedTest = init.then(admit(0, 1, false, true)).then(render(0).fail())
  run oversizedRejectedTest = init.then(admit(0, 1, true, false)).then(render(0).fail())
  run partialCannotPublishTest = init.then(admit(0, 1, true, true)).then(render(0)).then(publish(0).fail())
  run existingFilePreservedTest = init.then(admit(0, 0, true, true)).then(render(0)).then(encode(0)).then(collide(0))
    .expect(safety and s.files.get(0) == 9)
  run cannotReplaceExistingFileTest = init.then(admit(0, 0, true, true)).then(render(0)).then(encode(0)).then(publish(0).fail())
  run cancelBeforePublishTest = init.then(admit(0, 1, true, true)).then(render(0)).then(encode(0)).then(cancel(0)).then(publish(0).fail())
  run collisionOneWinnerTest = init
    .then(admit(0, 1, true, true)).then(admit(1, 1, true, true))
    .then(render(0)).then(render(1)).then(encode(0)).then(encode(1))
    .then(publish(0)).then(collide(1)).expect(safety and s.published == Set(0) and s.files.get(1) == 0)
  run failedBatchItemKeepsEarlierOutputTest = init
    .then(admit(0, 1, true, true)).then(render(0)).then(encode(0)).then(publish(0))
    .then(admit(1, 0, false, true)).expect(safety and s.published == Set(0) and s.stage.get(1) == "failed")
}
