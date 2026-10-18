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

  // A two-entry limit and two coarse tone bins abstract bounded RGB storage.
  // The final histogram includes every nontransparent position. This abstracts
  // the one-way exact-to-binned transition, not allocation or RGB arithmetic.
  pure def visiblePositions(tones, weights) = tones.indices().filter(i => weights.nth(i) > 0)
  pure def histogram(tones, weights) = {
    val positions = visiblePositions(tones, weights)
    val exact = positions.map(i => tones.nth(i))
    val folded = exact.size() > 2
    pure def keyFor(i) = if (folded) tones.nth(i) / 4 else tones.nth(i)
    val bins = positions.map(keyFor)
    bins.mapBy(key => {
      val members = positions.filter(i => keyFor(i) == key)
      { weight: members.fold(0, (sum, i) => sum + weights.nth(i)),
        sum: members.fold(0, (sum, i) => sum + tones.nth(i) * weights.nth(i)) }
    })
  }

  // Three candidates abstract the implementation's twelve-cell bound.
  pure val PaletteIDs = Set(0, 1, 2)
  pure val AlgorithmIDs = Set(0, 1)
  pure def candidatesValid(ids, allowed) = ids.length() >= 1 and ids.length() <= 3 and
    ids.indices().forall(i => allowed.contains(ids.nth(i)))
  pure def comparisonValid(algorithms, palettes, sharedPaletteOrRecipe) =
    if (palettes.length() > 0)
      algorithms.length() == 0 and not(sharedPaletteOrRecipe) and candidatesValid(palettes, PaletteIDs)
    else candidatesValid(algorithms, AlgorithmIDs)
  pure def paletteCells(ids, algorithm) = ids.indices().mapBy(i =>
    { palette: ids.nth(i), algorithm: algorithm, x: i % 2, y: i / 2 })
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

  run rareAccentIncludedTest = initCase(Pixels, 0)
    .expect(histogram(List(0, 7, 0, 0), List(3, 3, 3, 3)) ==
      Map(0 -> { weight: 9, sum: 0 }, 7 -> { weight: 3, sum: 21 }))
  run sparseExtractionFindsVisibleTest = initCase(Pixels, 0)
    .expect(histogram(List(0, 2, 0, 7), List(0, 1, 0, 3)) ==
      Map(2 -> { weight: 1, sum: 2 }, 7 -> { weight: 3, sum: 21 }))
  run transparentExtractionTest = initCase(Pixels, 0)
    .expect(histogram(List(0, 1, 2, 3), List(0, 0, 0, 0)).keys() == Set())
  run foldedHistogramPreservesOriginalSumsTest = initCase(Pixels, 0)
    .expect(histogram(List(0, 1, 6, 7), List(3, 1, 1, 3)) ==
      Map(0 -> { weight: 4, sum: 1 }, 1 -> { weight: 4, sum: 27 }))
  run histogramBoundAndCoverageTest = initCase(Pixels, 0)
    .expect(Pixels.powerset().forall(visible => {
      val weights = List(0, 1, 2, 3).foldl(List(), (list, i) => list.append(if (visible.contains(i)) 3 else 0))
      val h = histogram(List(0, 1, 6, 7), weights)
      all { h.keys().size() <= 2,
        h.keys().fold(0, (sum, key) => sum + h.get(key).weight) == visible.size() * 3,
        (h.keys().size() == 0) == (visible.size() == 0) }
    }))
  run histogramOrderIndependentTest = initCase(Pixels, 0)
    .expect(histogram(List(0, 1, 6, 7), List(3, 1, 1, 3)) ==
      histogram(List(7, 6, 1, 0), List(3, 1, 1, 3)))
  run exactHistogramRetainsNearbyColorsTest = initCase(Pixels, 0)
    .expect(histogram(List(0, 1, 0, 1), List(3, 2, 1, 3)) ==
      Map(0 -> { weight: 4, sum: 0 }, 1 -> { weight: 5, sum: 5 }))
  run paletteComparisonOrderTest = initCase(Pixels, 0)
    .expect(comparisonValid(List(), List(2, 0, 1), false) and
      paletteCells(List(2, 0, 1), 1) == Map(
        0 -> { palette: 2, algorithm: 1, x: 0, y: 0 },
        1 -> { palette: 0, algorithm: 1, x: 1, y: 0 },
        2 -> { palette: 1, algorithm: 1, x: 0, y: 1 }))
  run algorithmComparisonCompatibilityTest = initCase(Pixels, 0)
    .expect(comparisonValid(List(1, 0), List(), true))
  run ambiguousComparisonRejectedTest = initCase(Pixels, 0)
    .expect(not(comparisonValid(List(0), List(1), false)) and
      not(comparisonValid(List(), List(1), true)))
  run invalidComparisonRejectedTest = initCase(Pixels, 0)
    .expect(not(comparisonValid(List(), List(), false)) and
      not(comparisonValid(List(), List(3), false)) and
      not(comparisonValid(List(), List(0, 1, 2, 0), false)))
}
