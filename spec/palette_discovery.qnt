// The model has six sorted abstract palette IDs, three categories, and four searchable tokens.
// Go tests check real strings, hex colors, provenance, and the 256-entry data.
// Page limit 3/default 2 models the same bounded protocol as real 256/default 32.
module palette_discovery {
  pure val Catalog = List(0, 1, 2, 3, 4, 5)
  pure val Categories = Set(0, 1, 2)
  pure val Category = Map(0 -> 0, 1 -> 1, 2 -> 0, 3 -> 2, 4 -> 1, 5 -> 2)
  pure val ColorCount = Map(0 -> 2, 1 -> 4, 2 -> 3, 3 -> 5, 4 -> 2, 5 -> 4)
  pure val MetadataTokens = Map(
    0 -> Set(0, 1), 1 -> Set(1), 2 -> Set(0, 2),
    3 -> Set(2), 4 -> Set(0, 1), 5 -> Set(1, 2))
  pure def matches(id, category, lo, hi, terms) = all {
    category == -1 or Category.get(id) == category,
    ColorCount.get(id) >= lo, ColorCount.get(id) <= hi,
    terms.subseteq(MetadataTokens.get(id))
  }
  pure def filtered(category, lo, hi, terms) =
    Catalog.select(id => matches(id, category, lo, hi, terms))
  pure def pageAt(found, offset, limit) = {
    val start = if (offset > found.length()) found.length() else offset
    val end = if (start + limit > found.length()) found.length() else start + limit
    found.slice(start, end)
  }
  pure def nextOffset(total, offset, count) =
    if (offset + count >= total) -1 else offset + count

  type State = {
    phase: str, category: int, lo: int, hi: int, terms: Set[int],
    found: List[int], page: List[int], collected: List[int],
    initial: int, offset: int, limit: int, next: int
  }
  var s: State
  action init = s' = {
    phase: "idle", category: -1, lo: 2, hi: 5, terms: Set(),
    found: List(), page: List(), collected: List(),
    initial: 0, offset: 0, limit: 2, next: -1
  }

  action request(category, rawLo, rawHi, terms, rawLimit, offset, queryBounded) = {
    val lo = if (rawLo == 0) 2 else rawLo
    val hi = if (rawHi == 0) 5 else rawHi
    val limit = if (rawLimit == 0) 2 else rawLimit
    val valid = all {
      category == -1 or Categories.contains(category),
      lo >= 2, lo <= 5, hi >= 2, hi <= 5, lo <= hi,
      limit >= 1, limit <= 3, offset >= 0, queryBounded
    }
    val found = if (valid) filtered(category, lo, hi, terms) else List()
    val page = if (valid) pageAt(found, offset, limit) else List()
    s' = {
      phase: if (valid) "active" else "rejected",
      category: category, lo: lo, hi: hi, terms: terms,
      found: found, page: page, collected: page,
      initial: offset, offset: offset, limit: limit,
      next: if (valid) nextOffset(found.length(), offset, page.length()) else -1
    }
  }

  action advance = {
    val page = pageAt(s.found, s.next, s.limit)
    all {
      s.phase == "active", s.next >= 0,
      s' = {
        offset: s.next, page: page, collected: s.collected.concat(page),
        next: nextOffset(s.found.length(), s.next, page.length()), ...s
      }
    }
  }
  action repeatPage = all {
    s.phase == "active", s.page == pageAt(s.found, s.offset, s.limit), s' = s
  }
  action stutter = s' = s
  action step = {
    nondet category = Set(-1, 0, 1, 2, 3).oneOf()
    nondet lo = Set(0, 1, 2, 3, 4, 5, 6).oneOf()
    nondet hi = Set(0, 1, 2, 3, 4, 5, 6).oneOf()
    nondet terms = Set(0, 1, 2, 3).powerset().oneOf()
    nondet limit = Set(-1, 0, 1, 2, 3, 4).oneOf()
    nondet offset = Set(-1, 0, 1, 2, 3, 4, 5, 6, 7).oneOf()
    nondet bounded = Set(true, false).oneOf()
    any { request(category, lo, hi, terms, limit, offset, bounded), advance, repeatPage, stutter }
  }

  action initBrowse = request(-1, 0, 0, Set(), 2, 0, true)
  action browseStep = any { advance, repeatPage, stutter }

  val pageSound = s.page.indices().forall(i => matches(s.page.nth(i), s.category, s.lo, s.hi, s.terms))
  val pageBound = s.page.length() <= s.limit and s.page.length() <= 3
  val matchedCount = s.found == filtered(s.category, s.lo, s.hi, s.terms)
  val stableSlice = s.page == pageAt(s.found, s.offset, s.limit)
  val strictlyOrdered = s.collected.indices().forall(i =>
    i == 0 or s.collected.nth(i - 1) < s.collected.nth(i))
  val noDuplicates = s.collected.indices().map(i => s.collected.nth(i)).size() == s.collected.length()
  val concatenates = {
    val start = if (s.initial > s.found.length()) s.found.length() else s.initial
    val end = if (s.offset + s.page.length() > s.found.length()) s.found.length() else s.offset + s.page.length()
    s.collected == s.found.slice(start, end)
  }
  val correctNext = if (s.offset + s.page.length() >= s.found.length()) s.next == -1
    else s.next == s.offset + s.page.length() and s.next > s.offset
  val safety = all {
    Set("idle", "active", "rejected").contains(s.phase),
    s.phase == "active" implies all { pageSound, pageBound, matchedCount, stableSlice, strictlyOrdered, noDuplicates, concatenates, correctNext },
    s.phase == "rejected" implies (s.page == List() and s.collected == List() and s.next == -1)
  }
  val filteredNonempty = s.phase == "active" and s.page.length() > 0 and (s.category >= 0 or s.terms.size() > 0)
  val emptyMatch = s.phase == "active" and s.found.length() == 0
  val completeTraversal = s.phase == "active" and s.initial == 0 and s.found.length() > s.limit and s.next == -1
  val beyondEnd = s.phase == "active" and s.offset > s.found.length() and s.page == List()

  run defaultFirstPageTest = init.then(request(-1, 0, 0, Set(), 0, 0, true))
    .expect(safety and s.limit == 2 and s.page == List(0, 1) and s.next == 2 and s.found.length() == 6)
  run completeEnumerationTest = init.then(request(-1, 0, 0, Set(), 2, 0, true)).then(advance).then(advance)
    .expect(safety and completeTraversal and s.collected == Catalog and s.next == -1)
  run filterBeforePaginationTest = init.then(request(0, 0, 0, Set(), 1, 1, true))
    .expect(safety and s.page == List(2) and s.found == List(0, 2) and s.next == -1)
  run filtersComposeWithANDTest = init.then(request(1, 2, 3, Set(0, 1), 3, 0, true))
    .expect(safety and s.page == List(4))
  run emptyQueryResultTest = init.then(request(-1, 0, 0, Set(3), 3, 0, true))
    .expect(safety and emptyMatch and s.next == -1)
  run beyondEndIsEmptyTest = init.then(request(0, 0, 0, Set(), 2, 7, true))
    .expect(safety and beyondEnd and s.offset == 7 and s.next == -1)
  run repeatedPageIsStableTest = init.then(request(-1, 3, 5, Set(2), 2, 0, true)).then(repeatPage)
    .expect(safety and s.page == List(2, 3) and s.next == 2)
  run invalidRangeRejectedTest = init.then(request(-1, 5, 2, Set(), 2, 0, true))
    .expect(safety and s.phase == "rejected")
  run excessivePageRejectedTest = init.then(request(-1, 0, 0, Set(), 4, 0, true))
    .expect(safety and s.phase == "rejected")
  run negativeOffsetRejectedTest = init.then(request(-1, 0, 0, Set(), 2, -1, true))
    .expect(safety and s.phase == "rejected")
  run unknownCategoryRejectedTest = init.then(request(3, 0, 0, Set(), 2, 0, true))
    .expect(safety and s.phase == "rejected")
  run oversizedQueryRejectedTest = init.then(request(-1, 0, 0, Set(), 2, 0, false))
    .expect(safety and s.phase == "rejected")
  run cannotAdvanceFinalPageTest = init.then(request(-1, 0, 0, Set(), 3, 4, true)).then(advance.fail())
}
