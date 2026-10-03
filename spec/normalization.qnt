// Six distinct pixels form an asymmetric 3-by-2 source image.
// Abstract tones represent color conversion. Alpha values include 16-bit precision.
// Admission predicates represent bounded metadata parsing, not binary parsers.
module normalization {
  pure val Pixels = 0.to(5)
  pure val SourceTone = Map(0 -> 0, 1 -> 1, 2 -> 2, 3 -> 3, 4 -> 4, 5 -> 5)
  pure val SourceAlpha = Map(0 -> 0, 1 -> 1, 2 -> 257, 3 -> 32768, 4 -> 65534, 5 -> 65535)
  pure def width(o) = if (o >= 5) 2 else 3
  pure def height(o) = if (o >= 5) 3 else 2
  pure def sourceAt(o, p) = {
    val x = p % width(o)
    val y = p / width(o)
    if (o == 1) y * 3 + x
    else if (o == 2) y * 3 + 2 - x
    else if (o == 3) (1 - y) * 3 + 2 - x
    else if (o == 4) (1 - y) * 3 + x
    else if (o == 5) x * 3 + y
    else if (o == 6) (1 - x) * 3 + y
    else if (o == 7) (1 - x) * 3 + 2 - y
    else x * 3 + 2 - y
  }
  pure val ExpectedOrder = Map(
    1 -> List(0, 1, 2, 3, 4, 5), 2 -> List(2, 1, 0, 5, 4, 3),
    3 -> List(5, 4, 3, 2, 1, 0), 4 -> List(3, 4, 5, 0, 1, 2),
    5 -> List(0, 3, 1, 4, 2, 5), 6 -> List(3, 0, 4, 1, 5, 2),
    7 -> List(5, 2, 4, 1, 3, 0), 8 -> List(2, 5, 1, 4, 0, 3))
  pure def converted(tone, profile) = if (profile == "matrix") (tone + 1) % 8 else tone
  pure def acceptable(o, profile, bounded, metadataOK) = all {
    o >= 1, o <= 8, Set("none", "matrix").contains(profile), bounded, metadataOK
  }

  type State = {
    phase: str, orientation: int, profile: str, admitted: bool,
    ids: int -> int, tones: int -> int, alpha: int -> int,
    orientationRuns: int, colorRuns: int, workingWidth: int, workingHeight: int,
    processed: bool, cropID: int, cropTone: int, cropAlpha: int,
    frozenIDs: int -> int, frozenTones: int -> int, frozenAlpha: int -> int
  }
  var s: State
  action init = s' = {
    phase: "new", orientation: 1, profile: "none", admitted: false,
    ids: Pixels.mapBy(p => p), tones: SourceTone, alpha: SourceAlpha,
    orientationRuns: 0, colorRuns: 0, workingWidth: 3, workingHeight: 2,
    processed: false, cropID: -1, cropTone: -1, cropAlpha: -1,
    frozenIDs: Pixels.mapBy(p => p), frozenTones: SourceTone, frozenAlpha: SourceAlpha
  }
  action admit(o, profile, bounded, metadataOK) = {
    val valid = acceptable(o, profile, bounded, metadataOK)
    all {
      s.phase == "new",
      s' = { phase: if (valid) "active" else "rejected", orientation: o,
        profile: profile, admitted: valid, ...s }
    }
  }
  action orient = all {
    s.phase == "active", s.orientationRuns == 0,
    s' = { ids: Pixels.mapBy(p => s.ids.get(sourceAt(s.orientation, p))),
      tones: Pixels.mapBy(p => s.tones.get(sourceAt(s.orientation, p))),
      alpha: Pixels.mapBy(p => s.alpha.get(sourceAt(s.orientation, p))),
      orientationRuns: 1, workingWidth: width(s.orientation), workingHeight: height(s.orientation), ...s }
  }
  action convert = all {
    s.phase == "active", s.colorRuns == 0,
    s' = { tones: Pixels.mapBy(p => converted(s.tones.get(p), s.profile)), colorRuns: 1, ...s }
  }
  // The one-pixel top-left crop distinguishes raw and normalized coordinates.
  action process = all {
    s.phase == "active", s.orientationRuns == 1, s.colorRuns == 1,
    s' = { phase: "complete", processed: true,
      cropID: s.ids.get(0), cropTone: s.tones.get(0), cropAlpha: s.alpha.get(0), ...s }
  }
  action cancel = all {
    Set("new", "active").contains(s.phase),
    s' = { phase: "canceled", frozenIDs: s.ids, frozenTones: s.tones, frozenAlpha: s.alpha, ...s }
  }
  action stutter = s' = s
  action step = {
    nondet o = 0.to(9).oneOf()
    nondet profile = Set("none", "matrix", "lut", "cmyk", "malformed").oneOf()
    nondet bounded = Set(true, false).oneOf()
    nondet metadataOK = Set(true, false).oneOf()
    any { admit(o, profile, bounded, metadataOK), orient, convert, process, cancel, stutter }
  }
  action initNormalize = {
    nondet o = 1.to(8).oneOf()
    init.then(admit(o, "matrix", true, true))
  }
  action normalizeStep = any { orient, convert, process, stutter }

  val permutation = s.ids.keys() == Pixels and Pixels.map(p => s.ids.get(p)) == Pixels
  val alphaPreserved = Pixels.forall(p => s.alpha.get(p) == SourceAlpha.get(s.ids.get(p)))
  val colorCorrect = Pixels.forall(p => s.tones.get(p) ==
    (if (s.colorRuns == 1) converted(SourceTone.get(s.ids.get(p)), s.profile) else SourceTone.get(s.ids.get(p))))
  val orientationCorrect = s.orientationRuns == 1 implies all {
    Pixels.forall(p => s.ids.get(p) == ExpectedOrder.get(s.orientation).nth(p)),
    s.workingWidth == width(s.orientation), s.workingHeight == height(s.orientation)
  }
  val readyBeforeProcessing = s.processed implies all {
    s.admitted, s.orientationRuns == 1, s.colorRuns == 1,
    s.cropID == ExpectedOrder.get(s.orientation).nth(0),
    s.cropAlpha == SourceAlpha.get(s.cropID),
    s.cropTone == converted(SourceTone.get(s.cropID), s.profile)
  }
  val rejectedDoesNotProcess = s.phase == "rejected" implies all {
    not(s.admitted), not(s.processed), s.orientationRuns == 0, s.colorRuns == 0
  }
  val canceledDoesNotChange = s.phase == "canceled" implies all {
    not(s.processed), s.ids == s.frozenIDs, s.tones == s.frozenTones, s.alpha == s.frozenAlpha
  }
  val safety = all {
    permutation, alphaPreserved, colorCorrect, orientationCorrect, readyBeforeProcessing,
    rejectedDoesNotProcess, canceledDoesNotChange,
    s.orientationRuns >= 0, s.orientationRuns <= 1, s.colorRuns >= 0, s.colorRuns <= 1,
    s.workingWidth * s.workingHeight == 6
  }
  val normalizedComplete = s.phase == "complete"
  val swappedAxes = s.orientationRuns == 1 and s.orientation >= 5
  val rejectedMetadata = s.phase == "rejected"
  val canceledDuringNormalization = s.phase == "canceled" and (s.orientationRuns == 1 or s.colorRuns == 1)

  run allOrientationsMatchFixtureTest = init.expect(1.to(8).forall(o =>
    Pixels.forall(p => sourceAt(o, p) == ExpectedOrder.get(o).nth(p))))
  run orientationBeforeCropTest = init.then(admit(6, "none", true, true)).then(orient).then(convert).then(process)
    .expect(safety and s.cropID == 3 and s.cropAlpha == 32768 and s.workingWidth == 2 and s.workingHeight == 3)
  run conversionBeforeOrientationTest = init.then(admit(7, "matrix", true, true)).then(convert).then(orient).then(process)
    .expect(safety and s.cropID == 5 and s.cropTone == 6 and s.cropAlpha == 65535)
  run orientationBeforeConversionTest = init.then(admit(7, "matrix", true, true)).then(orient).then(convert).then(process)
    .expect(safety and s.cropID == 5 and s.cropTone == 6 and s.cropAlpha == 65535)
  run untaggedIdentityTest = init.then(admit(1, "none", true, true)).then(orient).then(convert).then(process)
    .expect(safety and s.tones == SourceTone and s.alpha == SourceAlpha)
  run cannotProcessBeforeOrientationTest = init.then(admit(6, "none", true, true)).then(convert).then(process.fail())
  run cannotProcessBeforeColorTest = init.then(admit(1, "matrix", true, true)).then(orient).then(process.fail())
  run cannotOrientTwiceTest = init.then(admit(6, "none", true, true)).then(orient).then(orient.fail())
  run cannotConvertTwiceTest = init.then(admit(1, "matrix", true, true)).then(convert).then(convert.fail())
  run unsupportedLUTRejectedTest = init.then(admit(1, "lut", true, true)).then(orient.fail())
  run unsupportedCMYKRejectedTest = init.then(admit(1, "cmyk", true, true)).then(convert.fail())
  run malformedProfileRejectedTest = init.then(admit(1, "malformed", true, true)).expect(safety and rejectedMetadata)
  run oversizedMetadataRejectedTest = init.then(admit(1, "matrix", false, true)).expect(safety and rejectedMetadata)
  run invalidOrientationRejectedTest = init.then(admit(9, "none", true, true)).expect(safety and rejectedMetadata)
  run conflictingMetadataRejectedTest = init.then(admit(1, "matrix", true, false)).expect(safety and rejectedMetadata)
  run canceledNormalizationCannotProcessTest = init.then(admit(6, "matrix", true, true)).then(orient).then(cancel)
    .expect(safety and canceledDuringNormalization).then(convert.fail())
}
