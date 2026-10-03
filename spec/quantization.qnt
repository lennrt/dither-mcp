// This finite model abstracts seeded quantization, selection, and alpha handling.
// Eight scalar tones represent RGB. Four pixels represent a bounded image.
// The model excludes floating point, diffusion kernels, geometry, and encoders.
module quantization {
  pure val Pixels = Set(0, 1, 2, 3)
  pure val Palette = Set(0, 3, 7)
  pure val Tones = 0.to(7)
  pure def clamp(v) = if (v < 0) 0 else if (v > 7) 7 else v
  pure def noise(pixel, seed) = ((pixel * 5 + seed * 2) % 3) - 1
  pure def nearest(v) = if (v <= 1) 0 else if (v <= 5) 3 else 7
  pure def quantize(v, pixel, seed) = nearest(clamp(v + noise(pixel, seed)))
  type State = {
    source: int -> int, alpha: int -> int, selected: Set[int], seed: int,
    left: int -> int, right: int -> int, leftDone: Set[int], rightDone: Set[int],
    cancelled: bool, cancelledLeft: Set[int], cancelledRight: Set[int]
  }
  var s: State
  action initCase(selected, seed) = s' = {
    source: Map(0 -> 1, 1 -> 2, 2 -> 4, 3 -> 6),
    alpha: Map(0 -> 0, 1 -> 1, 2 -> 3, 3 -> 3), selected: selected, seed: seed,
    left: Pixels.mapBy(_ => -1), right: Pixels.mapBy(_ => -1),
    leftDone: Set(), rightDone: Set(), cancelled: false, cancelledLeft: Set(), cancelledRight: Set()
  }
  action init = {
    nondet selected = Pixels.powerset().oneOf()
    nondet seed = 0.to(3).oneOf()
    initCase(selected, seed)
  }
  def expected(p) = if (s.alpha.get(p) == 0) 0 else if (s.selected.contains(p))
    quantize(s.source.get(p), p, s.seed) else s.source.get(p)
  action processLeft(p) = all {
    not(s.cancelled), not(s.leftDone.contains(p)),
    s' = { left: s.left.set(p, expected(p)), leftDone: s.leftDone.union(Set(p)), ...s }
  }
  action processRight(p) = all {
    not(s.cancelled), not(s.rightDone.contains(p)),
    s' = { right: s.right.set(p, expected(p)), rightDone: s.rightDone.union(Set(p)), ...s }
  }
  action cancel = all {
    not(s.cancelled),
    s' = { cancelled: true, cancelledLeft: s.leftDone, cancelledRight: s.rightDone, ...s }
  }
  action stutter = s' = s
  action step = {
    nondet p = Pixels.oneOf()
    any { processLeft(p), processRight(p), cancel, stutter }
  }
  action productiveStep = {
    nondet p = Pixels.oneOf()
    any { processLeft(p), processRight(p), stutter }
  }
  val paletteMembership = s.leftDone.forall(p =>
    (s.selected.contains(p) and s.alpha.get(p) > 0) implies Palette.contains(s.left.get(p)))
  val maskedSourcePreserved = s.leftDone.forall(p =>
    (not(s.selected.contains(p)) and s.alpha.get(p) > 0) implies s.left.get(p) == s.source.get(p))
  val transparentIsZero = s.leftDone.forall(p => s.alpha.get(p) == 0 implies s.left.get(p) == 0)
  val sameSeedSamePixels = s.leftDone.intersect(s.rightDone).forall(p => s.left.get(p) == s.right.get(p))
  val cancellationStopsWork = s.cancelled implies (s.leftDone == s.cancelledLeft and s.rightDone == s.cancelledRight)
  val safety = all { paletteMembership, maskedSourcePreserved, transparentIsZero, sameSeedSamePixels, cancellationStopsWork }
  val bothComplete = s.leftDone == Pixels and s.rightDone == Pixels
  val hasMaskedRegion = s.selected.size() > 0 and s.selected.size() < 4

  run deterministicDifferentTraversalTest = initCase(Pixels, 2)
    .then(processLeft(0)).then(processLeft(1)).then(processLeft(2)).then(processLeft(3))
    .then(processRight(3)).then(processRight(2)).then(processRight(1)).then(processRight(0))
    .expect(safety and bothComplete and s.left == s.right)
  run maskedSourceTest = initCase(Set(1), 0).then(processLeft(2))
    .expect(safety and s.left.get(2) == 4)
  run transparentBlackTest = initCase(Pixels, 0).then(processLeft(0))
    .expect(safety and s.left.get(0) == 0)
  run seedChangesNoiseTest = initCase(Pixels, 0)
    .expect(noise(0, 0) != noise(0, 1))
  run cannotProcessAfterCancelTest = initCase(Pixels, 0).then(processLeft(0)).then(cancel).then(processLeft(1).fail())
}
