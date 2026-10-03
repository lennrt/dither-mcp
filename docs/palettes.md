# The palette library

The library contains **256 palettes across 16 categories**. Use them with the same image pipeline, recipes, comparisons, animations, and print workflows. Each palette includes exact sRGB hex colors, a description, searchable tags, and provenance. The original 22 palette IDs and colors remain stable.

## Find a palette

The CLI and MCP use the same search. The default page contains 32 palettes
sorted by ID. Request up to 256 palettes per page. For more results, use `next_offset`.

```sh
./bin/dither-mcp palettes --query 'game boy' --max-colors 4
./bin/dither-mcp palettes --category ocean --min-colors 4 --max-colors 8 --limit 8
./bin/dither-mcp palettes --category print --limit 8
./bin/dither-mcp palettes --limit 256
```

```json
{
  "name": "dither_palettes",
  "arguments": {
    "query": "ocean",
    "min_colors": 4,
    "max_colors": 8,
    "limit": 8,
    "offset": 0
  }
}
```

Separate query terms with whitespace. Search ignores case and matches each term
as a substring. Every term must match at least one of these fields:

- ID or name.
- Description or category.
- Tags or origin.
- Hex colors.

Category and color-count filters further restrict the results. The response
contains these fields:

| Field | Meaning |
|---|---|
| `total` | Number of palettes in the complete library. |
| `matched` | Number of palettes that match all filters. |
| `count` | Number of palettes on this page. |
| `offset` | Requested start index in the filtered results. |
| `limit` | Maximum number of palettes on this page. |
| `next_offset` | Start index for the next page. The response omits this field at the end. |
| `categories` | Categories and their counts for the complete library. |

An offset beyond the results returns an empty array. Query text accepts up to
256 UTF-8 bytes. Color-count bounds range from 2 to 256. If the category is
unknown, the tool returns an error.

Use a returned `id` as the `palette` value in a render request.
You can also supply its `colors` array or store the palette in a version-1 recipe.
For print separation, choose a palette with up to 16 colors. Larger palettes work
with still images, comparisons, and animation.

## Browse categories

| Category | Palettes | Reference |
|---|---:|---|
| architecture | 16 | [Browse architecture](#architecture) |
| botanical | 16 | [Browse botanical](#botanical) |
| cosmic | 16 | [Browse cosmic](#cosmic) |
| duotone | 16 | [Browse duotone](#duotone) |
| food | 16 | [Browse food](#food) |
| interface | 16 | [Browse interface](#interface) |
| landscape | 16 | [Browse landscape](#landscape) |
| mineral | 16 | [Browse mineral](#mineral) |
| neon | 16 | [Browse neon](#neon) |
| neutral | 16 | [Browse neutral](#neutral) |
| ocean | 16 | [Browse ocean](#ocean) |
| pastel | 16 | [Browse pastel](#pastel) |
| print | 16 | [Browse print](#print) |
| retro | 16 | [Browse retro](#retro) |
| seasonal | 16 | [Browse seasonal](#seasonal) |
| terminal | 16 | [Browse terminal](#terminal) |

## Provenance and color interpretation

`original` identifies palettes composed for this project. `reference` identifies
published palette values with a source link. `approximation` identifies an sRGB
interpretation of historical hardware, pigments, or displays. Descriptions explain the intended
appearance. Display calibration and print profiles are separate.

Color counts describe unique colors in each palette. Every preset has a distinct
unordered color set. Reordering the same colors does not create another preset.

[The visual atlas](assets/palette-atlas.png) presents the complete collection.
The standalone showcase includes interactive swatches, filters, and copy controls.
The Go binary embeds the catalog. Palette discovery uses only local data.

## Architecture

### Art Deco Lobby

`art-deco-lobby` · 6 colors · original

Emerald, polished gold and dark walnut create a compact ornamental palette.

Colors: `#182c28`, `#345a49`, `#79906b`, `#c5b56d`, `#e6d6a2`, `#604235`

Tags: emerald, gold, ornament.

### Basalt Bungalow

`basalt-bungalow` · 7 colors · original

Volcanic stone, weathered timber and pale linen give domestic scenes a grounded palette.

Colors: `#232d30`, `#536164`, `#889693`, `#bac2b4`, `#e4e4d0`, `#937557`, `#c3a578`

Tags: stone, timber, linen.

### Bauhaus Primary

`bauhaus-primary` · 7 colors · original

Red, blue and yellow share balanced dark, gray and cream supports for geometric studies.

Colors: `#1c252c`, `#596369`, `#a6ada7`, `#f2eada`, `#c73a32`, `#d9b637`, `#245893`

Tags: primary, geometric, poster.

### Brick and Mortar

`brick-and-mortar` · 7 colors · original

Warm brick reds and cool mortar grays describe a familiar urban texture.

Colors: `#302b2e`, `#663c37`, `#a15b4b`, `#cd8d6c`, `#92958a`, `#c9cabe`, `#ece7d5`

Tags: brick, red, gray.

### Brutalist Sun

`brutalist-sun` · 6 colors · original

Concrete grays and a clear ochre accent give architectural planes a sunlit graphic finish.

Colors: `#1e2930`, `#48575c`, `#81908d`, `#bcc6ba`, `#e8e7d1`, `#d89839`

Tags: concrete, ochre, graphic.

### Cobalt Tile

`cobalt-tile` · 7 colors · original

Glazed cobalt, turquoise and warm grout give repeating patterns crisp depth and sparkle.

Colors: `#111e42`, `#23417b`, `#3d71b2`, `#6ca3c8`, `#b6d4d4`, `#f2ecdc`, `#368e8d`

Tags: tile, cobalt, ceramic.

### Copper Roof

`copper-roof` · 7 colors · original

Oxidized copper greens and roof-red accents capture the color of aged urban surfaces.

Colors: `#203933`, `#41675b`, `#719889`, `#acc3a6`, `#e2e3c5`, `#8e4d35`, `#c47c50`

Tags: copper, patina, roof.

### Glass Tower

`glass-tower` · 8 colors · original

Blue glass and silver reflections create a cool eight-tone palette for city studies.

Colors: `#152734`, `#284b61`, `#487990`, `#74a9b6`, `#a9ced0`, `#d5e4df`, `#f0f4e8`, `#777f89`

Tags: glass, blue, silver.

### Limestone Light

`limestone-light` · 6 colors · original

Warm stone grays progress into chalk white for soft architectural modeling.

Colors: `#35332e`, `#666359`, `#979181`, `#c6bcaa`, `#e7decd`, `#faf3e7`

Tags: stone, warm, tonal.

### Nocturne Window

`nocturne-window` · 7 colors · original

Deep blue city silhouettes carry amber and cream window lights.

Colors: `#111d31`, `#2a3e5a`, `#506d89`, `#8dabbc`, `#9b663a`, `#e5aa58`, `#f9dda0`

Tags: night, window, blue.

### Poolside Modern

`poolside-modern` · 7 colors · original

Aqua water, coral upholstery and pale concrete capture a bright modern courtyard.

Colors: `#22424a`, `#3a7e87`, `#79bec0`, `#b8ded4`, `#f1e9d2`, `#e9997d`, `#bb5960`

Tags: aqua, coral, summer.

### Porcelain Station

`porcelain-station` · 7 colors · original

Subway cream, deep green and blue-gray suggest glazed tile and cast-iron signs.

Colors: `#172e30`, `#325b50`, `#6d8b77`, `#aebbac`, `#e8e2cb`, `#f8f1db`, `#648897`

Tags: tile, transport, green.

### Rose Stucco

`rose-stucco` · 2 colors · original

Dusty rose and chalk form a warm two-color surface study.

Colors: `#92545e`, `#f8ead7`

Tags: rose, plaster, duotone.

### Sandstone Arch

`sandstone-arch` · 6 colors · original

Sandstone reds and pale apricot highlights echo a warm desert archway.

Colors: `#3b2630`, `#754140`, `#b66c53`, `#dba074`, `#f0c99b`, `#fae7c3`

Tags: sandstone, desert, orange.

### Terracotta Courtyard

`terracotta-courtyard` · 7 colors · original

Clay reds, dusty pink and pale plaster pair with a quiet garden green.

Colors: `#382a29`, `#7a4338`, `#b56e51`, `#da9d78`, `#edc7a5`, `#f4e5ce`, `#6c826b`

Tags: clay, plaster, courtyard.

### Venetian Facade

`venetian-facade` · 8 colors · original

Faded rose, lagoon green and golden plaster make a colorful waterside streetscape.

Colors: `#393445`, `#755263`, `#ab7775`, `#d5a28d`, `#f1d0ad`, `#efe5cc`, `#407b80`, `#89b4a5`

Tags: plaster, rose, lagoon.

## Botanical

### Bluebell Wood

`bluebell-wood` · 8 colors · original

Bluebell violet and pale blue contrast with shaded forest greens and soft spring sunlight.

Colors: `#1e3040`, `#43566b`, `#766c9f`, `#a598ce`, `#d2c9e7`, `#3f6251`, `#86a77c`, `#ece6c6`

Tags: bluebell, woodland, blue.

### Carnivorous Garden

`carnivorous-garden` · 8 colors · original

Deep emerald leaves, vivid chartreuse, and crimson pitchers create a contrasting palette for exotic plant studies.

Colors: `#132c2a`, `#245345`, `#4f8b59`, `#93bb63`, `#cde696`, `#7c2942`, `#c04c64`, `#f0a58b`

Tags: tropical, green, crimson.

### Desert Bloom

`desert-bloom` · 8 colors · original

Rose-red flowers interrupt sage cactus, ochre soil and pale desert light for a dry botanical study.

Colors: `#33362d`, `#63715a`, `#9baa7c`, `#d2d7a4`, `#905040`, `#c9796d`, `#eab190`, `#f4e1b8`

Tags: cactus, rose, desert.

### Eucalyptus

`eucalyptus` · 6 colors · original

Blue-green foliage, silver leaf and pale sage contrast gently with the warm brown of bark.

Colors: `#213c3c`, `#496466`, `#7b9692`, `#b1c3b1`, `#dce3c9`, `#907762`

Tags: sage, blue-green, muted.

### Fern Moon

`fern-moon` · 7 colors · original

Moonlit teal fronds and silver-green highlights give botanical silhouettes a cool evening atmosphere.

Colors: `#102b32`, `#28535b`, `#568183`, `#8fb1a3`, `#cad7bd`, `#eef1db`, `#52677c`

Tags: fern, moon, teal.

### Fern Study

`fern-study` · 6 colors · original

Fern fronds move from deep green shadows through yellow-green leaf light to warm botanical paper.

Colors: `#142c25`, `#345441`, `#5e8056`, `#95ac73`, `#c5d19b`, `#efe8c3`

Tags: fern, green, paper.

### Lavender Stem

`lavender-stem` · 8 colors · original

Lavender flowers cross cool purple shadows, gray-green stems and a sun-washed cream ground.

Colors: `#302b49`, `#655479`, `#9988b7`, `#c6b5d8`, `#e9dbec`, `#4c7165`, `#91a184`, `#f4edd7`

Tags: lavender, petal, muted.

### Lotus Pond

`lotus-pond` · 8 colors · original

Water green, pink lotus petals and yellow pollen reflect together in a calm eight-color flower palette.

Colors: `#183d3b`, `#3f7270`, `#7ea49a`, `#bad0b7`, `#ece4ca`, `#984c6b`, `#d48aa3`, `#e2c071`

Tags: lotus, water, pink.

### Magnolia

`magnolia` · 7 colors · original

Creamy magnolia petals and rose-tinted shadows sit against dark glossy foliage and brown branches.

Colors: `#22332d`, `#4c6250`, `#8e9b75`, `#b5aea1`, `#d8cab9`, `#f6ebd8`, `#b77d82`

Tags: flower, cream, green.

### Midnight Orchid

`midnight-orchid` · 8 colors · original

Black plum, orchid violet, and pale petal pink frame green leaves in a nighttime flower study.

Colors: `#171c2b`, `#3b304d`, `#6a4f79`, `#a678a8`, `#d2accc`, `#f0dedb`, `#3d665d`, `#7d9f7d`

Tags: orchid, night, petal.

### Moss & Lichen

`moss-and-lichen` · 7 colors · original

Velvety forest brown, moss green and pale lichen offer a richly textured woodland palette.

Colors: `#232b23`, `#4b5635`, `#7b8248`, `#a8aa65`, `#d1cb9c`, `#eae5bd`, `#847052`

Tags: moss, forest, earth.

### Nettle

`nettle` · 6 colors · original

A compact bright-green leaf palette balances shaded nettle stems with creamy warm sunlight.

Colors: `#172d21`, `#35553a`, `#618549`, `#9eb963`, `#dbe299`, `#f6edc6`

Tags: leaf, green, compact.

### Olive Grove

`olive-grove` · 7 colors · original

Silvery olive leaves and ochre bark rest in deep blue-green shade beneath warm Mediterranean light.

Colors: `#28352c`, `#53644b`, `#8b9871`, `#bac49b`, `#e4dfb6`, `#86634c`, `#b59067`

Tags: olive, silver, earth.

### Rose Garden

`rose-garden` · 8 colors · original

Wine petals, bright rose and peach light bloom against two clear leaf greens and soft cream.

Colors: `#392333`, `#78394d`, `#bd5d79`, `#e98fa0`, `#f4c2b8`, `#fae7d3`, `#31594b`, `#74956a`

Tags: rose, petal, garden.

### Saffron Crocus

`saffron-crocus` · 8 colors · original

Violet petals and saffron-orange stamens pair with emerald foliage and parchment highlights.

Colors: `#25263b`, `#584977`, `#9875b4`, `#d0add8`, `#f2dfeb`, `#be6c32`, `#ebb951`, `#45684d`

Tags: crocus, violet, saffron.

### Wildflower Meadow

`wildflower-meadow` · 9 colors · original

Leaf greens and blue sky meet cornflower, poppy and yellow blossoms for a broad summer bouquet.

Colors: `#263e39`, `#50795a`, `#98b375`, `#d9db9b`, `#f5e9c3`, `#697eb3`, `#a79ac9`, `#d47b7a`, `#e6b457`

Tags: flowers, multicolor, meadow.

## Cosmic

### Asteroid Belt

`asteroid-belt` · 8 colors · original

Dark basalt, steel-gray fragments and warm dust provide a quiet broad space-rock palette.

Colors: `#232c39`, `#4d5b6a`, `#7b8b92`, `#aebab8`, `#dadac7`, `#f0e9d6`, `#927e72`, `#c0a387`

Tags: asteroid, stone, neutral.

### Aurora Borealis

`aurora-borealis` · 8 colors · original

Inky winter blue frames emerald curtains, pale lime and violet sky light beneath bright snow cream.

Colors: `#0b2238`, `#204e60`, `#398575`, `#75c391`, `#b4e1a7`, `#e8eed0`, `#6f5895`, `#b698c2`

Tags: aurora, green, night.

### Comet Tail

`comet-tail` · 7 colors · original

Dark violet night trails through frosted blue and pale mint into warm star-cream light.

Colors: `#1e2443`, `#414d78`, `#7388af`, `#adc4da`, `#d9e7df`, `#f6efd8`, `#a190b2`

Tags: comet, ice, blue.

### Cosmic Ray

`cosmic-ray` · 8 colors · original

Deep space blue flashes through cool cyan, fluorescent green and violet with a star-white highlight.

Colors: `#10182d`, `#293e69`, `#546ba1`, `#85b8cb`, `#c8e4d7`, `#f3f1dc`, `#7ec580`, `#b381d1`

Tags: space, signal, bright.

### Deep Orbit

`deep-orbit` · 8 colors · original

Navy and graphite space shadows hold desaturated cyan, warm metal and a tiny amber thruster accent.

Colors: `#111f32`, `#2b4059`, `#526d83`, `#89a4af`, `#c8d4cc`, `#efead1`, `#9c806d`, `#d4ad67`

Tags: orbit, navy, metal.

### Eclipse Gold

`eclipse-gold` · 7 colors · original

Near-black navy and smoky brown silhouettes face copper corona, luminous gold and pale cream.

Colors: `#171c2c`, `#45424b`, `#80634d`, `#b88a4e`, `#e4bc68`, `#f4e3aa`, `#738b93`

Tags: eclipse, gold, contrast.

### Europa Ice

`europa-ice` · 7 colors · original

Dark ocean-blue fissures cross frozen aqua, chalky ice and pale tan surface streaks.

Colors: `#183849`, `#3b6475`, `#759aab`, `#b2cbcf`, `#e4e9dd`, `#f6f0d9`, `#a18875`

Tags: europa, ice, moon.

### Ion Drive

`ion-drive` · 7 colors · original

Deep violet hull shadow, cobalt ion light and electric cyan meet soft silver and pale mint exhaust.

Colors: `#161d39`, `#37436e`, `#5f71b0`, `#85afd9`, `#b1e4e5`, `#e7f2e3`, `#8b769e`

Tags: ion, cobalt, technology.

### Lunar Dust

`lunar-dust` · 7 colors · original

Cold slate, warm gray and pale regolith offer subtle lunar terrain with a restrained blue shadow.

Colors: `#29323c`, `#505d68`, `#81908f`, `#b6bdb0`, `#dedccb`, `#f4eddc`, `#736778`

Tags: moon, gray, terrain.

### Mars Survey

`mars-survey` · 7 colors · original

Wine-black craters, oxidized red soil and dusty orange span pale beige light and blue instrument shadow.

Colors: `#342530`, `#733d3c`, `#aa6250`, `#d49772`, `#e4bc91`, `#f1dfbd`, `#586a7c`

Tags: mars, rust, terrain.

### Midnight Zodiac

`midnight-zodiac` · 8 colors · original

Celestial indigo and violet surround muted teal, antique gold and pale constellation light.

Colors: `#17233e`, `#3c3e69`, `#706080`, `#a496a8`, `#d5c5bf`, `#efe4cd`, `#648e86`, `#c7ad68`

Tags: zodiac, indigo, gold.

### Nebula Rose

`nebula-rose` · 8 colors · original

Deep indigo space and violet dust contrast with rose nebulae, peach starlight, and small cool teal wisps.

Colors: `#141c3a`, `#353a69`, `#70577f`, `#ad779b`, `#db9cb2`, `#f1c9bc`, `#f5e6cc`, `#5d959a`

Tags: nebula, rose, space.

### Red Giant

`red-giant` · 8 colors · original

Dark wine and ember red grow through peach-orange to buttery star light with a faint smoky blue edge.

Colors: `#341d2c`, `#792b40`, `#b74850`, `#dd7b64`, `#f1b184`, `#f6dbb0`, `#f7edd5`, `#546079`

Tags: star, red, warm.

### Saturn Rings

`saturn-rings` · 7 colors · original

Soft gold, smoky mauve and creamy ring light orbit deep navy and gray-teal space shadows.

Colors: `#1d2941`, `#4e566b`, `#8a8491`, `#b9b09d`, `#e0c995`, `#f2e5c8`, `#668c8c`

Tags: saturn, rings, gold.

### Solar Wind

`solar-wind` · 8 colors · original

Burgundy shade and fiery coral stretch into orange, lemon and white light with a dark blue counterpoint.

Colors: `#34202d`, `#792f40`, `#ba554e`, `#e99159`, `#f0c36b`, `#f7e6a8`, `#f9f2d8`, `#375771`

Tags: solar, orange, fire.

### Stellar Nursery

`stellar-nursery` · 9 colors · original

Teal clouds, dusty violet and rose star-forming haze mingle under cream points of light.

Colors: `#182c40`, `#3a566e`, `#6c8098`, `#a0adb0`, `#d5d9ca`, `#f1e6cc`, `#8a5d82`, `#c894a7`, `#55938a`

Tags: stars, multicolor, dust.

## Duotone

### Aubergine & Cream

`aubergine-cream` · 2 colors · original

Rich aubergine against vanilla cream makes expressive dot patterns and soft illustrated shadows.

Colors: `#482a46`, `#f6e9c7`

Tags: two-color, plum, warm.

### Bottle Green & Lime

`bottle-green-lime` · 2 colors · original

Dark bottle green and pale lime produce crisp plant silhouettes and lively patterned fills.

Colors: `#164833`, `#e0ed94`

Tags: two-color, green, lime.

### Burgundy & Blush

`burgundy-blush` · 2 colors · original

Dark burgundy and a light blush create a tender print-like two-color portrait palette.

Colors: `#5b2235`, `#f6d7cb`

Tags: two-color, red, portrait.

### Chocolate & Rose

`chocolate-rose` · 2 colors · original

Chocolate ink and pale rose paper soften high-contrast portraits with a confectionery warmth.

Colors: `#47302f`, `#f5d8df`

Tags: two-color, brown, rose.

### Coal & Apricot

`coal-apricot` · 2 colors · original

Cool coal and warm apricot give silhouettes a bright, softly colored paper ground.

Colors: `#25333c`, `#f4c997`

Tags: two-color, orange, contrast.

### Cobalt & Ice

`cobalt-ice` · 2 colors · original

Cobalt blue against pale ice keeps geometry and one-bit-style texture cool and luminous.

Colors: `#244f91`, `#e3f4ef`

Tags: two-color, cobalt, ice.

### Espresso & Parchment

`espresso-parchment` · 2 colors · original

Espresso-brown ink on warm parchment creates a compact classic photographic print treatment.

Colors: `#3d281f`, `#efdcb5`

Tags: two-color, brown, paper.

### Indigo & Paper

`indigo-paper` · 2 colors · original

Deep indigo and warm paper form a crisp two-color treatment for drawings and type.

Colors: `#202d5c`, `#f0e5c9`

Tags: two-color, indigo, paper.

### Midnight & Peach

`midnight-peach` · 2 colors · original

Midnight navy and pale peach pair strong nighttime shadows with warm human highlights.

Colors: `#152d45`, `#fbd4b8`

Tags: two-color, navy, peach.

### Oxblood & Sand

`oxblood-sand` · 2 colors · original

A deep wine-red ink and light sandstone ground give poster work a grounded, antique warmth.

Colors: `#57202a`, `#e8cf9e`

Tags: two-color, wine, sand.

### Petrol & Celadon

`petrol-celadon` · 2 colors · original

Deep petrol and pale celadon give technical line art a cool, ceramic-like two-color finish.

Colors: `#153f4b`, `#d6e7cd`

Tags: two-color, teal, celadon.

### Plum & Mint

`plum-mint` · 2 colors · original

Deep plum shadows meet pale mint for a cool complementary two-color texture.

Colors: `#432d55`, `#d8efc9`

Tags: two-color, purple, mint.

### Rust & Ivory

`rust-ivory` · 2 colors · original

Dark red rust and clean ivory turn photos into warm, graphic dot studies.

Colors: `#853d30`, `#f5ebd2`

Tags: two-color, rust, ivory.

### Spruce & Milk

`spruce-milk` · 2 colors · original

Dark spruce and pale green milk keep foliage and line art calm and clear.

Colors: `#183e37`, `#e9efd6`

Tags: two-color, green, botanical.

### Ultramarine & Lemon

`ultramarine-lemon` · 2 colors · original

Strong ultramarine with lemon-yellow light turns simple images into energetic poster art.

Colors: `#283780`, `#f4ed89`

Tags: two-color, blue, yellow.

### Violet & Lilac

`violet-lilac` · 2 colors · original

Ink violet and frosted lilac create a gentle monochrome-like screen with purple depth.

Colors: `#392763`, `#e1d8f3`

Tags: two-color, violet, lilac.

## Food

### Black Sesame

`black-sesame` · 6 colors · original

Warm ivory and toasted gray bring a quiet ceramic character to food photographs and still life.

Colors: `#171819`, `#44403b`, `#71685c`, `#a89881`, `#d0c1a6`, `#f4ead4`

Tags: sesame, ivory, tonal.

### Blood Orange

`blood-orange` · 6 colors · original

Deep burgundy, vermilion and citrus cream celebrate translucent fruit and bright highlights.

Colors: `#321328`, `#701d34`, `#b83c3a`, `#eb7147`, `#f6a963`, `#fce0ac`

Tags: citrus, red, juicy.

### Blueberry Tart

`blueberry-tart` · 7 colors · original

Ink-blue berries meet toasted pastry and lavender cream in a balanced dessert palette.

Colors: `#171b38`, `#333a6a`, `#66679b`, `#a4a1c8`, `#e0d8e3`, `#f6ead2`, `#bc884e`

Tags: berry, blue, pastry.

### Cocoa and Cherry

`cocoa-and-cherry` · 8 colors · original

Chocolate browns and cherry reds create a deep, warm dessert palette with cream highlights.

Colors: `#241918`, `#52332b`, `#885340`, `#c08867`, `#e3ba91`, `#f7dfbd`, `#722239`, `#bc3d54`

Tags: chocolate, cherry, warm.

### Espresso Crema

`espresso-crema` · 2 colors · original

Espresso black and warm crema form a two-ink palette for strong silhouettes.

Colors: `#251b17`, `#e9c79a`

Tags: coffee, duotone, warm.

### Fig and Honey

`fig-and-honey` · 8 colors · original

Fig-purple shadows open into honey gold, pale flesh and a leaf-green accent.

Colors: `#2e1c35`, `#62425d`, `#9d6680`, `#c798a1`, `#e6c7b0`, `#f4deb3`, `#c9943f`, `#637f55`

Tags: fig, purple, honey.

### Matcha Ceremony

`matcha-ceremony` · 6 colors · original

Earthy tea greens and pale rice paper form a gentle six-tone study.

Colors: `#192f29`, `#3d5040`, `#697853`, `#9aa56a`, `#cdd4a0`, `#f1f0d7`

Tags: tea, green, paper.

### Oolong Steam

`oolong-steam` · 6 colors · original

Smoky blue-gray, tea amber and porcelain white bring a restrained warmth to interiors.

Colors: `#263439`, `#5b6c6d`, `#95a4a0`, `#d8ded0`, `#f4f0df`, `#9c724c`

Tags: tea, porcelain, smoky.

### Persimmon Tea

`persimmon-tea` · 6 colors · original

Roasted tea brown and persimmon orange glow against a warm ceramic cream.

Colors: `#302723`, `#6b4635`, `#b2603d`, `#e79553`, `#f4c880`, `#f9e6bc`

Tags: tea, orange, ceramic.

### Pistachio Gelato

`pistachio-gelato` · 6 colors · original

Pistachio, blush and almond cream lend soft illustrations a fresh confectionery color.

Colors: `#344c46`, `#68866a`, `#a5ba89`, `#d9dfb4`, `#f6ebd3`, `#d9a8a0`

Tags: gelato, green, pastel.

### Plum Wine

`plum-wine` · 4 colors · original

A compact four-color plum ramp gives portraits and folded fabric a rich, velvety tone.

Colors: `#261b32`, `#65415e`, `#af788a`, `#e7b8b5`

Tags: plum, wine, tonal.

### Raspberry Sorbet

`raspberry-sorbet` · 7 colors · original

Berry pinks and cool mint give playful prints a refreshing, bright finish.

Colors: `#461d47`, `#913568`, `#d85e8c`, `#f2a3b5`, `#f9d8d4`, `#709f91`, `#bce0bc`

Tags: berry, pink, mint.

### Saffron Market

`saffron-market` · 7 colors · original

Saffron yellow, paprika red and deep spice shadows make a rich market scene.

Colors: `#291b27`, `#6a2934`, `#a7482c`, `#d88924`, `#f2bd42`, `#f7df91`, `#56806b`

Tags: spice, gold, warm.

### Salted Caramel

`salted-caramel` · 5 colors · original

Five caramel tones carry luminous highlights through amber midtones into dark sugar.

Colors: `#34221d`, `#785036`, `#b58250`, `#d8b27b`, `#f4dfba`

Tags: caramel, amber, tonal.

### Sardine Tin

`sardine-tin` · 6 colors · original

Marine blue, silvery gray and tomato red evoke illustrated pantry labels.

Colors: `#102e49`, `#276488`, `#6ca8bc`, `#c2d8d5`, `#f1ead0`, `#c73f35`

Tags: pantry, blue, red.

### Wasabi Nori

`wasabi-nori` · 5 colors · original

Dark seaweed and bright yellow-green create contrasting colors for graphic food studies.

Colors: `#101f21`, `#3b5140`, `#849347`, `#d2d871`, `#f4efd0`

Tags: seaweed, green, high-contrast.

## Interface

### Dracula

`dracula` · 11 colors · reference

The Dracula theme colors pair charcoal and cream with bright pastel syntax accents.

Colors: `#282a36`, `#44475a`, `#f8f8f2`, `#6272a4`, `#8be9fd`, `#50fa7b`, `#ffb86c`, `#ff79c6`, `#bd93f9`, `#ff5555`, `#f1fa8c`

Tags: theme, dark, pastel.

[Dracula source](https://draculatheme.com/contribute).

### Nord

`nord` · 16 colors · reference

Nord's sixteen arctic colors pair polar neutrals with frost and aurora accents.

Colors: `#2e3440`, `#3b4252`, `#434c5e`, `#4c566a`, `#d8dee9`, `#e5e9f0`, `#eceff4`, `#8fbcbb`, `#88c0d0`, `#81a1c1`, `#5e81ac`, `#bf616a`, `#d08770`, `#ebcb8b`, `#a3be8c`, `#b48ead`

Tags: theme, arctic, 16-color.

[Nord source](https://www.nordtheme.com/docs/colors-and-palettes/).

### Solarized

`solarized` · 16 colors · reference

Ethan Schoonover's sixteen-color Solarized palette combines balanced base tones with eight accents.

Colors: `#002b36`, `#073642`, `#586e75`, `#657b83`, `#839496`, `#93a1a1`, `#eee8d5`, `#fdf6e3`, `#b58900`, `#cb4b16`, `#dc322f`, `#d33682`, `#6c71c4`, `#268bd2`, `#2aa198`, `#859900`

Tags: theme, balanced, 16-color.

[Solarized source](https://ethanschoonover.com/solarized/).

### Syntax Carbon

`syntax-carbon` · 12 colors · original

Six graphite neutrals and six sharp accent colors create a versatile modern indexed palette.

Colors: `#181a22`, `#333742`, `#555c69`, `#86929f`, `#bac4cc`, `#f0f1e9`, `#dc777a`, `#e5ae68`, `#a2ba76`, `#72b9b4`, `#819ed1`, `#b68bc3`

Tags: theme, graphite, contrast.

### Syntax Clay

`syntax-clay` · 12 colors · original

A warm editor-style palette combines terracotta accents with stone, sage and dusty blue.

Colors: `#28232a`, `#49404a`, `#736572`, `#a294a0`, `#cbbfc5`, `#f2e8dd`, `#b76a57`, `#dcab78`, `#91a180`, `#6c99a2`, `#a486b9`, `#c7819e`

Tags: theme, earth, muted.

### Syntax Cobalt

`syntax-cobalt` · 12 colors · original

Navy neutrals and cobalt blue set a strong stage for lime, apricot and lavender accents.

Colors: `#17213c`, `#2b3e63`, `#4a648e`, `#8199bb`, `#b9cbe0`, `#f0f2e8`, `#568fda`, `#7cb8cb`, `#aabc76`, `#dca875`, `#c982a4`, `#a68ecf`

Tags: theme, cobalt, bright.

### Syntax Dawn

`syntax-dawn` · 12 colors · original

Soft dawn neutrals and peach, violet and blue accents keep broad indexed artwork light and gentle.

Colors: `#302d42`, `#514d68`, `#817994`, `#aaa4b9`, `#d3cedd`, `#f5f0e8`, `#e8a98b`, `#d6b46c`, `#9db68e`, `#85b7cd`, `#b39bce`, `#dca4ba`

Tags: theme, dawn, pastel.

### Syntax Ember

`syntax-ember` · 12 colors · original

Dark warm neutrals and ember-orange accents balance muted aqua, sage and rose for firelit scenes.

Colors: `#241d20`, `#48363a`, `#735357`, `#a38280`, `#cbb4a9`, `#f2e4ca`, `#d17f57`, `#e8b765`, `#8caa81`, `#6d9fa1`, `#a98bbb`, `#cc8c9e`

Tags: theme, ember, warm.

### Syntax Frost

`syntax-frost` · 12 colors · original

Pale frosted neutrals and cool blue accents retain gentle gold, berry and green notes.

Colors: `#243341`, `#455b6a`, `#718997`, `#a0b6bf`, `#c8d8dd`, `#eff3ed`, `#709bca`, `#76b3b6`, `#a6bd8f`, `#d2b579`, `#c38b9a`, `#a29cc4`

Tags: theme, frost, cool.

### Syntax Garden

`syntax-garden` · 12 colors · original

Twelve calm interface colors pair slate neutrals with leaf, rose, honey and sky accents.

Colors: `#1d2928`, `#364541`, `#56675c`, `#829582`, `#b8c5af`, `#e8ead6`, `#7fb89e`, `#d8ad70`, `#c7838b`, `#a697c5`, `#82aec5`, `#d8d0a2`

Tags: theme, garden, balanced.

### Syntax Mint

`syntax-mint` · 12 colors · original

Cool dark ink and creamy mint neutrals meet restrained red, blue and lavender syntax accents.

Colors: `#172b2a`, `#344b48`, `#5b746d`, `#8eaaa0`, `#c0d4c2`, `#edf3df`, `#69b6a2`, `#a7c26a`, `#c68172`, `#739dc6`, `#b799c7`, `#e1c184`

Tags: theme, mint, cool.

### Syntax Ocean

`syntax-ocean` · 12 colors · original

Twelve sea-toned interface colors use coral and gold accents to animate navy and pale surf neutrals.

Colors: `#102431`, `#294555`, `#526b79`, `#7b96a2`, `#b0c7cc`, `#e5ece2`, `#4e9b9d`, `#84c8bc`, `#e6a080`, `#d7bd7a`, `#a192c4`, `#c88599`

Tags: theme, ocean, balanced.

### Syntax Orchid

`syntax-orchid` · 12 colors · original

Plum neutrals support lilac, pink, fern and gold accents in an expressive dark-theme study.

Colors: `#271e33`, `#463751`, `#6e5b7b`, `#9b84a2`, `#c7b7cc`, `#f0e5e7`, `#ac8fce`, `#d196b4`, `#8aaa83`, `#d3b879`, `#80a4b9`, `#e2aa82`

Tags: theme, orchid, violet.

### Syntax Parchment

`syntax-parchment` · 12 colors · original

Warm reading-paper neutrals carry restrained ink-blue, red-clay, olive and violet accents.

Colors: `#2c2926`, `#554d44`, `#85776a`, `#b2a18d`, `#d8cbb4`, `#f6efdb`, `#4f7183`, `#8b5b4e`, `#b28b51`, `#6d8460`, `#8f729c`, `#ba8c96`

Tags: theme, paper, warm.

### Syntax Spectrum 32

`syntax-spectrum` · 32 colors · original

Thirty-two deliberately spaced neutrals and colored accents support broad diagrams and detailed pixel illustrations.

Colors, in palette order:

```text
#101722 #26313f #425465 #647c8b #8ca4ae #b5c8cc #dbe5e1 #f8f5e6
#662e48 #9f4660 #d16b79 #f5a19b #6d4327 #a4753d #d0a35c #f1cc86
#344e36 #557b48 #82a45e #b6cc88 #164a4e #2c7d7b #55aba3 #96d8c6
#243d6b #3e64a0 #6d91c5 #a6c4e7 #4b3268 #775498 #a77bbe #d4afe0
```

Tags: theme, 32-color, multicolor.

### Web-safe 216

`web-safe-216` · 216 colors · reference

The exact Go standard-library browser cube provides all 216 combinations of red, green and blue at steps of 51.

Colors, in palette order:

```text
#000000 #000033 #000066 #000099 #0000cc #0000ff #003300 #003333
#003366 #003399 #0033cc #0033ff #006600 #006633 #006666 #006699
#0066cc #0066ff #009900 #009933 #009966 #009999 #0099cc #0099ff
#00cc00 #00cc33 #00cc66 #00cc99 #00cccc #00ccff #00ff00 #00ff33
#00ff66 #00ff99 #00ffcc #00ffff #330000 #330033 #330066 #330099
#3300cc #3300ff #333300 #333333 #333366 #333399 #3333cc #3333ff
#336600 #336633 #336666 #336699 #3366cc #3366ff #339900 #339933
#339966 #339999 #3399cc #3399ff #33cc00 #33cc33 #33cc66 #33cc99
#33cccc #33ccff #33ff00 #33ff33 #33ff66 #33ff99 #33ffcc #33ffff
#660000 #660033 #660066 #660099 #6600cc #6600ff #663300 #663333
#663366 #663399 #6633cc #6633ff #666600 #666633 #666666 #666699
#6666cc #6666ff #669900 #669933 #669966 #669999 #6699cc #6699ff
#66cc00 #66cc33 #66cc66 #66cc99 #66cccc #66ccff #66ff00 #66ff33
#66ff66 #66ff99 #66ffcc #66ffff #990000 #990033 #990066 #990099
#9900cc #9900ff #993300 #993333 #993366 #993399 #9933cc #9933ff
#996600 #996633 #996666 #996699 #9966cc #9966ff #999900 #999933
#999966 #999999 #9999cc #9999ff #99cc00 #99cc33 #99cc66 #99cc99
#99cccc #99ccff #99ff00 #99ff33 #99ff66 #99ff99 #99ffcc #99ffff
#cc0000 #cc0033 #cc0066 #cc0099 #cc00cc #cc00ff #cc3300 #cc3333
#cc3366 #cc3399 #cc33cc #cc33ff #cc6600 #cc6633 #cc6666 #cc6699
#cc66cc #cc66ff #cc9900 #cc9933 #cc9966 #cc9999 #cc99cc #cc99ff
#cccc00 #cccc33 #cccc66 #cccc99 #cccccc #ccccff #ccff00 #ccff33
#ccff66 #ccff99 #ccffcc #ccffff #ff0000 #ff0033 #ff0066 #ff0099
#ff00cc #ff00ff #ff3300 #ff3333 #ff3366 #ff3399 #ff33cc #ff33ff
#ff6600 #ff6633 #ff6666 #ff6699 #ff66cc #ff66ff #ff9900 #ff9933
#ff9966 #ff9999 #ff99cc #ff99ff #ffcc00 #ffcc33 #ffcc66 #ffcc99
#ffcccc #ffccff #ffff00 #ffff33 #ffff66 #ffff99 #ffffcc #ffffff
```

Tags: historical, 216-color, color-cube.

[Web-safe 216 source](https://pkg.go.dev/image/color/palette#WebSafe).

## Landscape

### Alpine Morning

`alpine-morning` · 8 colors · original

Deep blue peaks, glacier blue and soft golden light share mossy meadow green and pale snow.

Colors: `#172b42`, `#385470`, `#698fa8`, `#a7c9d0`, `#e4eee2`, `#f1d9a0`, `#526c52`, `#8da174`

Tags: alpine, sky, morning.

### Badlands

`badlands` · 7 colors · original

Purple shale, brick-red strata and ochre dust create richly stepped warm rock formations.

Colors: `#342838`, `#62404a`, `#965a51`, `#c38869`, `#dab396`, `#eed9b8`, `#857e8c`

Tags: rock, red, earth.

### Basalt Coast

`basalt-coast` · 7 colors · original

Basalt black and storm blue contrast with sage surf, cool foam, and a small copper shore accent.

Colors: `#18262a`, `#374a54`, `#617b81`, `#97aba6`, `#c8d4c2`, `#e8ebd5`, `#a4785e`

Tags: coast, stone, storm.

### Canyon Echo

`canyon-echo` · 7 colors · original

Plum shadows, rust, clay, and apricot describe canyon walls beneath pale sand and sky.

Colors: `#382d42`, `#755056`, `#b17658`, `#d7a478`, `#e9c799`, `#f6e7c4`, `#7897a1`

Tags: canyon, rust, desert.

### Desert Caravan

`desert-caravan` · 7 colors · original

Indigo night shade, baked ochre and rose sand carry the warm light of a desert crossing.

Colors: `#2e2c4c`, `#66526b`, `#a77467`, `#cc9c62`, `#e2bf82`, `#f5e3b6`, `#71898a`

Tags: desert, ochre, travel.

### Fjord Light

`fjord-light` · 8 colors · original

Navy fjord water, slate cliffs and frosted cyan span mossy land and pink northern light.

Colors: `#142938`, `#365665`, `#6a8791`, `#a6bec0`, `#dfe6d6`, `#627b62`, `#a4b18a`, `#cca8a4`

Tags: fjord, water, north.

### Highland Rain

`highland-rain` · 8 colors · original

Heather purple, wet moss and granite gray form an atmospheric muted highland scene.

Colors: `#29323a`, `#52616a`, `#87928d`, `#b9bda9`, `#e5dfcc`, `#69546f`, `#a38199`, `#7d8c58`

Tags: highland, rain, heather.

### Lake Reflection

`lake-reflection` · 8 colors · original

Lake blues and forest greens meet muted rose sky in an evenly shaded waterside palette.

Colors: `#172d3d`, `#36596e`, `#64909d`, `#a6c0bd`, `#e4e6cb`, `#526c4b`, `#96a16f`, `#c99588`

Tags: lake, reflection, blue.

### Misty Valley

`misty-valley` · 7 colors · original

Layered teal mountains and fog-gray light let distant silhouettes soften into a pale green sky.

Colors: `#183437`, `#38595b`, `#668080`, `#97ad9e`, `#c4d1bb`, `#e8ecda`, `#858b94`

Tags: valley, fog, teal.

### Monsoon Ridge

`monsoon-ridge` · 9 colors · original

Ink-blue rain clouds and lush emerald hills hold yellow-green breaks of light above warm earth.

Colors: `#1b2c3d`, `#39596b`, `#5f8790`, `#96b7b2`, `#d9debf`, `#365e4b`, `#71985a`, `#b4bf77`, `#99715e`

Tags: monsoon, rain, green.

### Redwood Trail

`redwood-trail` · 8 colors · original

Red-brown bark and deep forest shade meet moss, filtered gold and pale trail dust.

Colors: `#262c26`, `#485542`, `#7c8155`, `#a9a774`, `#dbd09a`, `#754737`, `#b87c53`, `#ebd8b0`

Tags: redwood, forest, bark.

### Rolling Prairie

`rolling-prairie` · 8 colors · original

Wide green grassland and mellow yellow fields sit under dusty blue sky with cream cloud light.

Colors: `#263b39`, `#526958`, `#8a9c69`, `#c0bd82`, `#e5d49f`, `#f4ecd0`, `#617f99`, `#a6c1cc`

Tags: prairie, grass, sky.

### Salt Flat

`salt-flat` · 8 colors · original

Blue-gray reflections, dusty mauve and chalk-white salt sit beside pale straw light.

Colors: `#364452`, `#73818d`, `#a9b7bc`, `#d5ded9`, `#f6f1df`, `#947886`, `#c3a59f`, `#d9c398`

Tags: salt, reflection, pale.

### Savanna Dusk

`savanna-dusk` · 7 colors · original

Acacia silhouettes and dusty gold grasses fade into mauve twilight and peach horizon light.

Colors: `#30293a`, `#635362`, `#967d70`, `#c4a173`, `#e0c78f`, `#f3dfb0`, `#555d43`

Tags: savanna, dusk, gold.

### Sunset

`sunset` · 6 colors · original

Plum shadows and amber peach highlights capture the soft color of a fading sky.

Colors: `#211b35`, `#62365b`, `#b45a68`, `#e99074`, `#ffd1a0`, `#fff1d4`

Tags: sunset, warm, sky.

### Volcanic Island

`volcanic-island` · 8 colors · original

Charcoal lava rock, hot clay and tropical teal create strong island silhouettes under warm cloud light.

Colors: `#1d282c`, `#44565a`, `#688778`, `#9bba94`, `#e3dfb1`, `#8f4137`, `#d4744d`, `#f0b974`

Tags: volcanic, island, contrast.

## Mineral

### Agate

`agate` · 8 colors · original

Dark plum, red earth and muted amber alternate with pale gray bands in a richly striped stone study.

Colors: `#332835`, `#68505d`, `#9d6c6e`, `#c69985`, `#dec1a4`, `#f0dfc6`, `#82949a`, `#b8c6c3`

Tags: agate, banded, earth.

### Amethyst

`amethyst` · 7 colors · original

Dark violet facets and lavender crystal light share a cool gray-green mineral base.

Colors: `#29223f`, `#50405f`, `#806184`, `#b291bb`, `#d9c4e2`, `#efe3eb`, `#768e87`

Tags: amethyst, violet, crystal.

### Azurite

`azurite` · 8 colors · original

Vivid blue mineral and small malachite-green flecks contrast with warm pale rock and deep slate.

Colors: `#162943`, `#284e81`, `#497cbd`, `#84add4`, `#cadde3`, `#f1ead2`, `#4d7852`, `#9bb07a`

Tags: azurite, blue, green.

### Citrine

`citrine` · 7 colors · original

Smoky honey and amber crystal rise into lemon light and a bright creamy mineral highlight.

Colors: `#3f3427`, `#80603a`, `#b78a43`, `#d7b55f`, `#eadb92`, `#f8edc8`, `#a2aa8c`

Tags: citrine, yellow, crystal.

### Fluorite

`fluorite` · 8 colors · original

Green, aqua and lavender bands stretch through shaded purple crystal into cool pale highlights.

Colors: `#2b3049`, `#565078`, `#967fb2`, `#c9b7d7`, `#e7e4da`, `#417d70`, `#7bb79c`, `#b5d9bb`

Tags: fluorite, crystal, multicolor.

### Garnet

`garnet` · 7 colors · original

Near-black burgundy rises through ruby facets and warm red light into soft rose highlights.

Colors: `#291a27`, `#592337`, `#8b354d`, `#b85b65`, `#db9090`, `#efbdae`, `#f4e2c9`

Tags: garnet, red, crystal.

### Hematite

`hematite` · 7 colors · original

Cold iron-gray facets carry subtle mauve reflection and warm rust at the stone edge.

Colors: `#242d35`, `#465660`, `#768891`, `#aab6b5`, `#d8dfd4`, `#785b61`, `#b58575`

Tags: iron, gray, rust.

### Lapis Lazuli

`lapis-lazuli` · 8 colors · original

Deep ultramarine stone, cobalt flecks and pyrite gold meet pale mineral grain.

Colors: `#1e254d`, `#334675`, `#526b9e`, `#89a4c5`, `#c7d7df`, `#eeeada`, `#a78b46`, `#d6b866`

Tags: lapis, blue, gold.

### Malachite

`malachite` · 7 colors · original

Black-green veins, vivid malachite bands and pale stone create rich patterned emerald textures.

Colors: `#132d27`, `#225943`, `#43875e`, `#79b27b`, `#b9d2a0`, `#e6e6bb`, `#957d55`

Tags: green, stone, banded.

### Moonstone

`moonstone` · 7 colors · original

Blue-gray shadows and frosted cyan sheen rest against milky pearl and a soft peach edge.

Colors: `#384752`, `#6e8796`, `#a9bdc6`, `#d5e1de`, `#f3efdf`, `#c4aaa3`, `#e7ccc0`

Tags: moonstone, pearl, blue.

### Obsidian

`obsidian` · 7 colors · original

Black volcanic glass moves through blue-gray and plum reflections into a pale sharp edge.

Colors: `#10151e`, `#29303f`, `#4c5365`, `#7d8090`, `#b7b8ba`, `#e0dfd2`, `#5d485b`

Tags: obsidian, black, glass.

### Opal

`opal` · 9 colors · original

Pearl gray and cream hold softly luminous coral, aqua, lavender and pale gold flashes.

Colors: `#374654`, `#75828b`, `#b7c2bd`, `#e4e8d9`, `#f8f1dd`, `#d5a4b5`, `#89c7c1`, `#b9a6d5`, `#e6c88b`

Tags: opal, iridescent, pale.

### Pyrite

`pyrite` · 7 colors · original

Bronze shadow, muted olive-gold and bright pale brass capture small faceted metallic forms.

Colors: `#302d24`, `#61573a`, `#97864a`, `#c2ad64`, `#e0d18d`, `#f1e7bd`, `#7b806e`

Tags: pyrite, gold, metal.

### Rose Quartz

`rose-quartz` · 7 colors · original

Wine-gray shadow, dusty rose and soft cream let crystal facets feel warm and translucent.

Colors: `#4d374b`, `#876778`, `#b598a2`, `#d9bec3`, `#ecd9d7`, `#f8eee1`, `#aca8a0`

Tags: quartz, rose, pale.

### Tiger Eye

`tiger-eye` · 6 colors · original

Espresso bands, warm amber and honey reflections create a compact golden-brown stone palette.

Colors: `#29231d`, `#5f4126`, `#996332`, `#c18d46`, `#e0b76d`, `#f4dba3`

Tags: amber, brown, banded.

### Turquoise Stone

`turquoise-stone` · 7 colors · original

Turquoise veins and blue-green mineral meet warm sandstone, amber and cream.

Colors: `#1e3e43`, `#38777c`, `#6bb2ae`, `#abd5c1`, `#e2e8c5`, `#8a6550`, `#c39c6d`

Tags: turquoise, stone, earth.

## Neon

### Acid Aquarium

`acid-aquarium` · 7 colors · original

Black teal, vivid aquatic cyan and acid chartreuse mix with lavender and glowing pearl.

Colors: `#0b2535`, `#175d77`, `#29a6b5`, `#68ded1`, `#bcf39b`, `#f0efc5`, `#7864b9`

Tags: aqua, lime, electric.

### Arcade Sherbet

`arcade-sherbet` · 8 colors · original

Ink-blue outlines hold vivid cherry, tangerine, mint and lavender for a cheerful neon sprite palette.

Colors: `#20234a`, `#544a89`, `#9c70b5`, `#e792b9`, `#f3ba88`, `#ebd979`, `#7acdbe`, `#a5dbe4`

Tags: arcade, multicolor, bright.

### Chromatic Rain

`chromatic-rain` · 8 colors · original

Navy rain shadows and muted teal puddles reflect intense violet, rose and lemon signs.

Colors: `#0e2438`, `#294860`, `#49778e`, `#78adb3`, `#bed7cd`, `#8461b4`, `#dc7d9a`, `#f3d484`

Tags: rain, reflection, multicolor.

### Neon after midnight

`cyberpunk` · 6 colors · original

Deep violet, hot pink and luminous cyan create a saturated city-night palette.

Colors: `#100c2b`, `#452c70`, `#d82e8a`, `#ffb66c`, `#57e5e1`, `#e9f8ff`

Tags: night, pink, cyan.

### Electric Papaya

`electric-papaya` · 7 colors · original

Deep violet and electric coral contrast with papaya orange, sunny yellow, and mint-green light.

Colors: `#241a3c`, `#603b75`, `#b3577e`, `#ed896d`, `#f8ba69`, `#f3dc90`, `#8ed8bb`

Tags: orange, coral, electric.

### Hologram Rose

`hologram-rose` · 8 colors · original

Deep blue-purple and luminous rose cross icy cyan, lavender and a pale gold holographic sheen.

Colors: `#1c2445`, `#4d4c84`, `#9179bd`, `#cea4dc`, `#f8d8eb`, `#5dbdcc`, `#ace8e5`, `#f4e7a9`

Tags: hologram, pink, ice.

### Hot Pink Carbon

`hot-pink-carbon` · 7 colors · original

Graphite blue, saturated hot pink and pale blush create strong fluorescent fashion-like dot treatments.

Colors: `#172633`, `#3b4b62`, `#788394`, `#b8c0c5`, `#ecddd6`, `#9c3265`, `#ef72a4`

Tags: pink, graphite, contrast.

### Laser Lime

`laser-lime` · 7 colors · original

Navy screen shadows carry electric lime, vivid turquoise and pale laser yellow.

Colors: `#0a202b`, `#224d62`, `#3c8799`, `#60ccbf`, `#a5ed75`, `#e3f49e`, `#f5eed2`

Tags: lime, laser, contrast.

### Neon Monsoon

`neon-monsoon` · 8 colors · original

Blue-black rain and emerald signs meet purple reflection, coral lamps and bright humid cream.

Colors: `#092838`, `#1c5870`, `#38998e`, `#76d6b0`, `#cbefd0`, `#785198`, `#cc7395`, `#f0b995`

Tags: rain, green, city.

### Night Market

`night-market` · 8 colors · original

Petrol night, lantern orange and red-pink signs share teal reflections and pale streetlight cream.

Colors: `#172e39`, `#385b66`, `#6b9290`, `#b8cabb`, `#ede2b8`, `#ac4a63`, `#e27958`, `#f2b56d`

Tags: night, lantern, orange.

### Nightclub Iris

`nightclub-iris` · 7 colors · original

Iris violet and saturated pink contrast with deep indigo, aqua, and pale stage highlights.

Colors: `#14182e`, `#323764`, `#70519e`, `#b278cb`, `#e9a7df`, `#50bfc6`, `#b8e9dc`

Tags: violet, pink, night.

### Poolside Laser

`poolside-laser` · 7 colors · original

Night cobalt, radiant pool cyan and lime highlights contrast with sunny orange and light peach.

Colors: `#102f4c`, `#2c6595`, `#51a3cf`, `#89dede`, `#d3f0cb`, `#f6e0ac`, `#e4976f`

Tags: pool, cyan, laser.

### Signal Flare

`signal-flare` · 8 colors · original

Blue-black and cool steel shadows frame brilliant red-orange, amber and pale emergency light.

Colors: `#102b3a`, `#3a5c6d`, `#7794a1`, `#bfccc6`, `#eeecd1`, `#ab3e4c`, `#ec7956`, `#f5bd68`

Tags: signal, orange, contrast.

### Synthwave Salmon

`synthwave-salmon` · 8 colors · original

Violet dusk, cobalt shadows and warm salmon light create a bright retro-future sunset study.

Colors: `#261b46`, `#52366e`, `#8a528d`, `#cd799c`, `#f7afa8`, `#f8d3a0`, `#4274ab`, `#79c2d2`

Tags: synthwave, salmon, sky.

### Ultraviolet City

`ultraviolet-city` · 8 colors · original

Inky blue, ultraviolet purple and hot magenta combine neon cyan with warm lit-window yellow.

Colors: `#0c1630`, `#28335b`, `#6a4aa1`, `#b164c6`, `#f074b0`, `#39b7c9`, `#9be5d4`, `#f4d886`

Tags: ultraviolet, city, night.

### Ultraviolet Lime

`ultraviolet-lime` · 7 colors · original

A tight saturated violet-and-lime palette moves from deep plum to bright lemon-cream.

Colors: `#221736`, `#54347d`, `#9b62c3`, `#d19ce8`, `#74b66a`, `#b6df76`, `#f1ed9e`

Tags: violet, lime, contrast.

## Neutral

### Blue Black Study

`blue-black-study` · 8 colors · original

An eight-step blue-black ramp opens smoothly into cool porcelain highlights.

Colors: `#0c1724`, `#202d3d`, `#394b5e`, `#5a7183`, `#8499a5`, `#b0bec4`, `#d9e0df`, `#f4f5ea`

Tags: blue, tonal, ink.

### Carbon Fog

`carbon-fog` · 5 colors · original

Dense carbon and pale fog shape a clear five-value range for graphic forms.

Colors: `#101518`, `#424b50`, `#808e94`, `#bbc8c9`, `#ecf1ea`

Tags: carbon, grayscale, graphic.

### Cool Grays 6

`cool-grays-6` · 6 colors · original

Six slate grays carry a cool blue cast through crisp photographic highlights.

Colors: `#18232b`, `#3b4b58`, `#677b88`, `#9bacb6`, `#ced9de`, `#f1f6f4`

Tags: grayscale, cool, tonal.

### Grayscale 16

`grayscale-16` · 16 colors · original

Sixteen evenly spaced gray values give monochrome photographs and diagrams a detailed tonal range.

Colors: `#000000`, `#111111`, `#222222`, `#333333`, `#444444`, `#555555`, `#666666`, `#777777`, `#888888`, `#999999`, `#aaaaaa`, `#bbbbbb`, `#cccccc`, `#dddddd`, `#eeeeee`, `#ffffff`

Tags: grayscale, 16-color, tonal.

### Four grays

`grayscale-4` · 4 colors · original

Four evenly spaced gray levels for compact tonal studies and readable texture.

Colors: `#000000`, `#555555`, `#aaaaaa`, `#ffffff`

Tags: grayscale, tonal, compact.

### Eight grays

`grayscale-8` · 8 colors · original

Eight evenly spaced grays preserve gentle shading in monochrome photographs.

Colors: `#000000`, `#242424`, `#494949`, `#6d6d6d`, `#929292`, `#b6b6b6`, `#dbdbdb`, `#ffffff`

Tags: grayscale, tonal, photography.

### Ivory and Slate

`ivory-and-slate` · 2 colors · original

Deep slate and warm ivory create a clean two-tone pairing for lettering and silhouettes.

Colors: `#283c4b`, `#f5ecd5`

Tags: duotone, ivory, slate.

### Linen Graphite

`linen-graphite` · 5 colors · original

Graphite darks and creamy linen midtones create a soft five-color monochrome study.

Colors: `#272c2c`, `#636662`, `#a7a99a`, `#d6d5be`, `#f5efd8`

Tags: linen, graphite, warm.

### Paper & ink

`mono` · 2 colors · original

Pure black and white for crisp linework, silhouettes and classic one-bit texture.

Colors: `#000000`, `#ffffff`

Tags: monochrome, high-contrast, linework.

### Mushroom Paper

`mushroom-paper` · 5 colors · original

Mushroom taupes and softly tinted paper give quiet still life a natural warmth.

Colors: `#393234`, `#756667`, `#ac9b93`, `#d4c8b9`, `#f3eadd`

Tags: taupe, paper, soft.

### Oat and Ink

`oat-and-ink` · 4 colors · original

Charcoal, oatmeal and paper cream form a versatile warm four-tone palette.

Colors: `#242629`, `#6b6860`, `#b7ad96`, `#f1e7cf`

Tags: oat, warm, paper.

### Olive Charcoal

`olive-charcoal` · 5 colors · original

Green-tinted charcoal and olive gray lend tonal drawings a subdued organic cast.

Colors: `#222d26`, `#4c5947`, `#829072`, `#b9c3a0`, `#e7ead0`

Tags: olive, charcoal, tonal.

### Parchment 12

`parchment-12` · 12 colors · original

Twelve warm parchment tones support detailed tonal work and delicate paper texture.

Colors: `#211c19`, `#362c25`, `#4d3e32`, `#66513f`, `#81684f`, `#9e8161`, `#b89b76`, `#ceb38e`, `#dfc8a7`, `#ecdbbf`, `#f5e9d6`, `#fbf4e8`

Tags: parchment, 12-color, tonal.

### Pewter Rose

`pewter-rose` · 6 colors · original

Pewter grays and restrained rose tones bring a subtle warmth to portraits.

Colors: `#333840`, `#616875`, `#92939b`, `#c3b7b7`, `#e6d1c9`, `#f8ebe1`

Tags: pewter, rose, portrait.

### Silver Halide

`silver-halide` · 6 colors · original

A cool silver ramp with warm-white highlights gives photographs a classic print atmosphere.

Colors: `#1a1e25`, `#39414c`, `#606d78`, `#909da5`, `#c1cace`, `#eceee7`

Tags: silver, photography, tonal.

### Warm Grays 6

`warm-grays-6` · 6 colors · original

Six warm grays preserve a gentle brown cast from shadow to soft paper light.

Colors: `#25211f`, `#514a45`, `#81776e`, `#b0a497`, `#d7ccbd`, `#f6eee2`

Tags: grayscale, warm, tonal.

## Ocean

### Abyssal

`abyssal` · 7 colors · original

Midnight blue and electric cold light make a deep-sea palette with muted violet and pale cyan.

Colors: `#070f23`, `#152d4e`, `#2a5277`, `#547d9c`, `#98b6c5`, `#dde5da`, `#665b8d`

Tags: deep-sea, blue, night.

### Arctic Current

`arctic-current` · 7 colors · original

Polar navy and glacier blue ease into frozen mint, snow cream and a pale violet edge.

Colors: `#122b43`, `#345873`, `#6f99b0`, `#a9cad3`, `#d5e7df`, `#f3f2dd`, `#a6a8c0`

Tags: arctic, ice, blue.

### Coral Reef

`coral-reef` · 8 colors · original

Aquatic navy and turquoise balance hot coral, soft shell and golden reef light.

Colors: `#16364d`, `#2e6e85`, `#53acb3`, `#9ad7cf`, `#e7edce`, `#9e4c68`, `#e78278`, `#e9b76a`

Tags: reef, coral, tropical.

### Deep Coral

`deep-coral` · 8 colors · original

Coral red, wine and warm pink glow within deep blue water and a softly lit cream highlight.

Colors: `#152d43`, `#3a5577`, `#647ba0`, `#a37d9d`, `#d095a5`, `#f0c8ba`, `#f5e8d0`, `#913e51`

Tags: coral, red, underwater.

### Harbor Sunset

`harbor-sunset` · 8 colors · original

Purple harbor silhouettes and cool teal water catch amber, salmon and apricot evening reflections.

Colors: `#252c42`, `#51526d`, `#7e8190`, `#aabeb0`, `#e9dfbb`, `#ad6a76`, `#dd977b`, `#edbf86`

Tags: harbor, sunset, reflection.

### Kelp Forest

`kelp-forest` · 7 colors · original

Kelp green and amber fronds move through blue-green depths into sun-filtered pale water.

Colors: `#0e2c31`, `#2c5351`, `#547e63`, `#8fa174`, `#c8c68e`, `#e7e3b8`, `#9c8054`

Tags: kelp, green, underwater.

### Midnight Swell

`midnight-swell` · 7 colors · original

Dark ocean navy, slate violet, and muted turquoise describe nighttime waves with a moon-white crest.

Colors: `#0b1b2f`, `#273b58`, `#4b5978`, `#748a9c`, `#a8bec4`, `#dfe7df`, `#4a817c`

Tags: waves, night, navy.

### Deep water

`ocean` · 5 colors · original

Navy depth, turquoise water and pale green light create a compact marine ramp.

Colors: `#071f2c`, `#165264`, `#278b9a`, `#63c7b2`, `#d7f3d4`

Tags: water, teal, tonal.

### Octopus Ink

`octopus-ink` · 8 colors · original

Deep plum-black, indigo and warm violet trace tentacle shadows against teal water and pearl light.

Colors: `#201c30`, `#45405a`, `#7a5f82`, `#ab88ab`, `#d5bccc`, `#efe1df`, `#3e7575`, `#85a9a0`

Tags: ink, plum, deep-sea.

### Pelagic

`pelagic` · 7 colors · original

Clear open-water blues range from deep cobalt through blue-green currents to pale ocean light.

Colors: `#112947`, `#214e7f`, `#367caf`, `#5aa5c6`, `#93cbd1`, `#d2e6dd`, `#f2f0d4`

Tags: open-water, cobalt, tonal.

### Saltwater Pearl

`saltwater-pearl` · 7 colors · original

Pearl cream, shell rose and silver teal create soft pale highlights over deep marine shadow.

Colors: `#2f454a`, `#678283`, `#a0b5ae`, `#ced4bd`, `#ece8d3`, `#f7e4d8`, `#bc9796`

Tags: pearl, shell, pale.

### Sea Spray

`sea-spray` · 7 colors · original

Soft green foam and cloudy aqua create an airy coastal ramp rooted in cool slate shadows.

Colors: `#344f57`, `#66878c`, `#9fb7b2`, `#c7d7c7`, `#e7ecd9`, `#f5efdf`, `#b8c7dc`

Tags: foam, aqua, pale.

### Storm Surge

`storm-surge` · 7 colors · original

Navy-black, steel blue and cold pale spray hold a small yellow signal accent for stormy water.

Colors: `#14202e`, `#304455`, `#577282`, `#8da8b0`, `#c6d8d1`, `#e5eedf`, `#c3ad6d`

Tags: storm, steel, contrast.

### Tidal Glass

`tidal-glass` · 7 colors · original

Deep petrol, sea-glass teal and luminous foam hold a gentle shell-pink accent in a clear coastal palette.

Colors: `#123a43`, `#276c78`, `#5aa0a3`, `#94c8bb`, `#d5e6d0`, `#f6efdb`, `#d89c93`

Tags: sea-glass, teal, foam.

### Tidepool

`tidepool` · 8 colors · original

Stone brown, kelp green and blue pool reflections share pink shell and warm pale sand.

Colors: `#273d40`, `#526e6c`, `#91a68d`, `#c8c9a2`, `#ece4bd`, `#826958`, `#b38b76`, `#d8b0a1`

Tags: tidepool, stone, shell.

### Turquoise Lagoon

`turquoise-lagoon` · 7 colors · original

Saturated turquoise water crosses shallow lime-green light and pale sand with warm coral detail.

Colors: `#143d4d`, `#237482`, `#3baaa6`, `#7ed1b4`, `#cce7bf`, `#f5e6b3`, `#db987c`

Tags: lagoon, turquoise, tropical.

## Pastel

### Apricot Mist

`apricot-mist` · 7 colors · original

Muted blue-gray shadows meet dusty apricot, pale sand and gently pink mist.

Colors: `#4b606b`, `#8aa4ac`, `#c0d0cd`, `#e4e5d5`, `#f8edd7`, `#c58e7c`, `#e6b9a5`

Tags: apricot, mist, muted.

### Baby Blue Coral

`baby-blue-coral` · 7 colors · original

Gentle blue-gray and powder blue balance light coral, rosy peach and warm off-white.

Colors: `#41596e`, `#85a6bf`, `#b9d4df`, `#e1eae0`, `#f5e7d6`, `#d99097`, `#efb8aa`

Tags: blue, coral, soft.

### Blush Linen

`blush-linen` · 6 colors · original

Mauve-brown, dusty blush and warm linen make a soft portrait palette with muted sage relief.

Colors: `#5b4d56`, `#9b858b`, `#cab0af`, `#e5d0c4`, `#f4e9da`, `#a7b5a0`

Tags: blush, linen, portrait.

### Chalk Garden

`chalk-garden` · 8 colors · original

Slate-blue outlines hold pale sage, lilac, butter and rose like chalk on softly textured paper.

Colors: `#495960`, `#929f9b`, `#c4d3bc`, `#e8e6c9`, `#f6f0de`, `#b7a9ce`, `#e5bfc6`, `#e2c79f`

Tags: chalk, garden, multicolor.

### Lilac Cloud

`lilac-cloud` · 7 colors · original

Indigo-gray shadows lift lavender, powder blue and warm ivory into softly colored cloud layers.

Colors: `#46405f`, `#88809e`, `#b9afcf`, `#ddd3e8`, `#f4eee6`, `#a7c5d0`, `#d6e3d6`

Tags: lilac, cloud, soft.

### Macaroon Mint

`macaroon-mint` · 7 colors · original

Toasted nut shadows, creamy mint and pale rose compose an airy bakery-window palette.

Colors: `#545447`, `#94977d`, `#c2d0ad`, `#e2ebcb`, `#f6eed7`, `#d2b49f`, `#e7c4cd`

Tags: mint, rose, cream.

### Peach Sorbet

`peach-sorbet` · 7 colors · original

Warm rose-brown grounds peach, apricot and cream alongside a small pale mint accent.

Colors: `#62444a`, `#a67875`, `#dca594`, `#f3c6a9`, `#f8e2bf`, `#f8f0dd`, `#b8ccb2`

Tags: peach, apricot, warm.

### Periwinkle Pearl

`periwinkle-pearl` · 6 colors · original

Cool indigo-gray and periwinkle melt into pearly pink, powder blue and luminous cream.

Colors: `#48526f`, `#8593b9`, `#b8c3df`, `#dde2e8`, `#f4efdf`, `#d5b8ce`

Tags: periwinkle, pearl, blue.

### Petal Paper

`petal-paper` · 7 colors · original

Dusty plum and warm gray carry muted mauve, pale rose and creamy handmade-paper light.

Colors: `#574653`, `#94828a`, `#c2adb1`, `#e0cbd0`, `#f1e2dd`, `#f8f0db`, `#acbaab`

Tags: petal, paper, muted.

### Pistachio Rose

`pistachio-rose` · 7 colors · original

Deep olive anchors pistachio cream, dusty rose and pale peach for a gentle garden confection palette.

Colors: `#39463b`, `#79977a`, `#b7caa0`, `#e4e7c0`, `#f7edda`, `#bc859b`, `#e6b6bc`

Tags: pistachio, rose, soft.

### Porcelain Pink

`porcelain-pink` · 7 colors · original

Cool plum-gray edges, delicate rose and ceramic cream support subtle pale-aqua reflections.

Colors: `#5a566c`, `#9690a4`, `#c9b3ca`, `#e6ccd8`, `#f5e7e7`, `#f7f1df`, `#b4d2cc`

Tags: porcelain, pink, pale.

### Sage Meringue

`sage-meringue` · 6 colors · original

Gray-green shadows and soft sage contrast with buttery meringue and a small peach blush.

Colors: `#435749`, `#839a7c`, `#b9cb9f`, `#dee1b9`, `#f4efd8`, `#cfaa9d`

Tags: sage, cream, muted.

### Seafoam Candy

`seafoam-candy` · 7 colors · original

Deep blue-teal grounds soft aqua and pale mint with a small lavender-sugar accent.

Colors: `#365965`, `#76a6a8`, `#b2d1c3`, `#dcedd1`, `#f5f1df`, `#c0aed5`, `#e9ccd9`

Tags: seafoam, aqua, candy.

### Sherbet Sky

`sherbet-sky` · 7 colors · original

Blue-violet shadow and pale cloud blue share pastel orange, rose and sunlit yellow.

Colors: `#435674`, `#869fbd`, `#bbd4df`, `#e7eddd`, `#f9edbf`, `#efc2a1`, `#dba6bc`

Tags: sky, sherbet, multicolor.

### Soft Confetti

`soft-confetti` · 10 colors · original

Warm charcoal grounds eight pastel pink, aqua, green, blue and yellow notes for playful broad artwork.

Colors: `#4d535b`, `#969fa8`, `#d0d8d1`, `#f3ecdc`, `#e2adc0`, `#efcba3`, `#e6dfad`, `#b6d5bd`, `#a8cbdc`, `#c2b8dc`

Tags: confetti, multicolor, soft.

### Vanilla Lavender

`vanilla-lavender` · 6 colors · original

Violet-brown depth supports lavender cream, vanilla light and a quiet cool-blue accent.

Colors: `#594762`, `#9983a7`, `#c7b0d4`, `#e7d6ea`, `#f6efd9`, `#c5d9dd`

Tags: vanilla, lavender, cream.

## Print

### Blackletter

`blackletter` · 4 colors · original

Dense brown-black ink, taupe and old ivory make letterforms and high-contrast ornament feel tactile.

Colors: `#211b19`, `#61534b`, `#a6927e`, `#e6d8bd`

Tags: ink, paper, ornament.

### Blockprint

`blockprint` · 8 colors · original

Eight strong earth and sea colors share warm paper for playful carved shapes and layered posters.

Colors: `#20272b`, `#445e61`, `#7a9790`, `#b7c2a2`, `#e8dcb6`, `#a04c44`, `#d28c61`, `#e9b66c`

Tags: blockprint, poster, earth.

### Blueprint

`blueprint` · 4 colors · original

Dark navy, cyan wash and pale paper lend drawings a technical print character.

Colors: `#071c40`, `#165995`, `#72b6d3`, `#e8f5eb`

Tags: blue, paper, technical.

### Copperplate

`copperplate` · 7 colors · original

Copper brown, ink-blue and ivory combine fine etched shadows with a small oxidized green accent.

Colors: `#232c31`, `#4b5356`, `#82604a`, `#b38d68`, `#d6ba8b`, `#f3e8cd`, `#779c92`

Tags: etching, copper, paper.

### Cyanotype Sun

`cyanotype-sun` · 7 colors · original

Prussian-blue-inspired shadows and sunlit cream extend a print-blue ramp with a restrained gold accent.

Colors: `#102e49`, `#285b7c`, `#4c88a3`, `#8cbec6`, `#d5e4d9`, `#f2e9c7`, `#d1b168`

Tags: cyanotype, blue, gold.

### Etching Rose

`etching-rose` · 6 colors · original

Rose-brown midtones and parchment highlights keep engraved portraits warm and delicate.

Colors: `#302027`, `#67434e`, `#a76d77`, `#d69e9b`, `#eac8b9`, `#f8ecda`

Tags: etching, rose, paper.

### Letterpress Blue

`letterpress-blue` · 4 colors · original

Four ink-blue and uncoated-paper tones suit bold poster shapes and orderly hatch texture.

Colors: `#1b293d`, `#465d79`, `#9baec2`, `#eee7d5`

Tags: letterpress, blue, paper.

### Linocut Ochre

`linocut-ochre` · 4 colors · original

Near-black olive and strong ochre ink make a graphic four-color relief-print study.

Colors: `#292c22`, `#69704c`, `#b89b43`, `#eee2aa`

Tags: linocut, ochre, bold.

### Lithograph

`lithograph` · 7 colors · original

Blue-black, muted green and red clay meet creamy paper in a broad illustrated-book palette.

Colors: `#202d35`, `#4a6570`, `#7f9d99`, `#b6c6ae`, `#e8e4c8`, `#b2755e`, `#e1ad7a`

Tags: lithography, book, muted.

### Marigold Riso

`marigold-riso` · 5 colors · original

Dark plum, marigold and warm rose suggest a cheerful small-ink poster over pale cream.

Colors: `#34283e`, `#996144`, `#e3af45`, `#e28d9e`, `#f5e8cb`

Tags: poster, marigold, rose.

### Museum Poster

`museum-poster` · 6 colors · original

Deep teal, rust, saffron, and cool paper colors create a palette for modern exhibition prints.

Colors: `#173d42`, `#42757a`, `#9aac98`, `#ca744f`, `#e3b562`, `#f3ebd6`

Tags: poster, teal, rust.

### Newsprint

`newspaper` · 4 colors · original

Charcoal, warm gray and aged paper keep tonal photographs soft and tactile.

Colors: `#29251f`, `#767166`, `#c2bcad`, `#f0ead8`

Tags: newsprint, paper, tonal.

### Riso pink & blue

`risograph` · 5 colors · original

Pink and process blue accents on warm paper suggest a playful layered poster print.

Colors: `#20304a`, `#0078bf`, `#ff48b0`, `#ffd5d5`, `#fff8ea`

Tags: poster, pink, blue.

### Sepia print

`sepia` · 4 colors · original

Warm brown ink and pale paper bring a quiet photographic print character to texture.

Colors: `#24180f`, `#6a4630`, `#b58a5d`, `#efd9b5`

Tags: sepia, warm, paper.

### Velvet Poster

`velvet-poster` · 7 colors · original

Velvety midnight blue, ruby and pink lift warm cream for theatrical screen-print images.

Colors: `#24213b`, `#534866`, `#895770`, `#bf7082`, `#e7aaa3`, `#f5e6ca`, `#859ca8`

Tags: poster, red, night.

### Woodcut

`woodcut` · 7 colors · original

Charcoal, earth brown and forest green give carved illustrations warm shadows and golden grain.

Colors: `#25251e`, `#584d35`, `#7d7954`, `#a8a176`, `#d4c99b`, `#f1e6bd`, `#516a50`

Tags: woodcut, earth, forest.

## Retro

### Apple II Lo-res

`apple-ii` · 15 colors · approximation

An sRGB approximation of the Apple II low-resolution analog colors, with the two equivalent gray indices represented once.

Colors: `#000000`, `#9d0966`, `#2a2ae5`, `#c734ff`, `#007a20`, `#808080`, `#0d9fff`, `#a0afff`, `#555500`, `#f96400`, `#ffa0d0`, `#14f53c`, `#d7ff83`, `#78ffd8`, `#ffffff`

Tags: hardware, analog, apple.

[Apple II Lo-res source](https://apple2history.org/dl/a2refmanorig.pdf).

### Arcade Foundry

`arcade-foundry` · 16 colors · original

Sixteen jewel-bright sprite colors share sturdy ink shadows and warm highlight tones.

Colors: `#141621`, `#34344f`, `#575478`, `#8b82a3`, `#d4cad5`, `#f8efd9`, `#7b243c`, `#c14756`, `#ef7869`, `#f3b263`, `#f6dd78`, `#265b4b`, `#479968`, `#89c887`, `#255c87`, `#58a8c6`

Tags: arcade, pixel-art, jewel.

### CGA cyan/magenta

`cga` · 4 colors · approximation

The high-intensity cyan and magenta CGA combination expressed as conventional sRGB values.

Colors: `#000000`, `#55ffff`, `#ff55ff`, `#ffffff`

Tags: hardware, cyan, magenta.

[CGA cyan/magenta source](https://www.pcjs.org/machines/pcx86/ibm/video/).

### CGA red/green

`cga-warm` · 4 colors · approximation

The bright green, red and yellow CGA combination expressed as conventional sRGB values.

Colors: `#000000`, `#55ff55`, `#ff5555`, `#ffff55`

Tags: hardware, primary, bright.

[CGA red/green source](https://www.pcjs.org/machines/pcx86/ibm/video/).

### Commodore 64 (approximate sRGB)

`commodore-64` · 16 colors · approximation

A widely used sixteen-color sRGB approximation of the analog Commodore 64 display.

Colors: `#000000`, `#ffffff`, `#68372b`, `#70a4b2`, `#6f3d86`, `#588d43`, `#352879`, `#b8c76f`, `#6f4f25`, `#433900`, `#9a6759`, `#444444`, `#6c6c6c`, `#9ad284`, `#6c5eb5`, `#959595`

Tags: hardware, analog, 16-color.

[Commodore 64 (approximate sRGB) source](https://commodore.ca/manuals/c64_users_guide/c64-users_guide.htm).

### EGA 16

`ega` · 16 colors · approximation

The classic sixteen RGBI colors expressed using the conventional bright and dark sRGB levels.

Colors: `#000000`, `#0000aa`, `#00aa00`, `#00aaaa`, `#aa0000`, `#aa00aa`, `#aa5500`, `#aaaaaa`, `#555555`, `#5555ff`, `#55ff55`, `#55ffff`, `#ff5555`, `#ff55ff`, `#ffff55`, `#ffffff`

Tags: hardware, 16-color, primary.

[EGA 16 source](https://www.pcjs.org/machines/pcx86/ibm/video/).

### Game Boy green

`gameboy` · 4 colors · approximation

A familiar green sRGB approximation of the original monochrome handheld display.

Colors: `#0f380f`, `#306230`, `#8bac0f`, `#9bbc0f`

Tags: handheld, green, display-inspired.

[Game Boy green source](https://github.com/gbdev/pandocs/blob/master/src/Palettes.md).

### Game Boy Pocket

`gameboy-pocket` · 4 colors · approximation

A four-tone gray approximation of the reflective monochrome pocket display.

Colors: `#181818`, `#565656`, `#a8a8a8`, `#e0e0e0`

Tags: handheld, grayscale, display-inspired.

[Game Boy Pocket source](https://github.com/gbdev/pandocs/blob/master/src/Palettes.md).

### MSX TMS9918

`msx` · 15 colors · approximation

The palette follows MAME's TMS9928A sRGB approximation with fifteen unique display colors. Transparent and black indices share one opaque black.

Colors: `#000000`, `#21c842`, `#5edc78`, `#5455ed`, `#7d76fc`, `#d4524d`, `#42ebf5`, `#fc5554`, `#ff7978`, `#d4c154`, `#e6ce80`, `#21b03b`, `#c95bba`, `#cccccc`, `#ffffff`

Tags: hardware, msx, analog.

[MSX TMS9918 source](https://github.com/mamedev/mame/blob/master/src/devices/video/tms9928a.cpp).

### NES Classic

`nes` · 55 colors · approximation

The palette approximates NES base colors with a conventional NTSC emulator interpretation in sRGB. It removes repeated black entries. Analog appearance varies by decoder.

Colors, in palette order:

```text
#7c7c7c #0000fc #0000bc #4428bc #940084 #a80020 #a81000 #881400
#503000 #007800 #006800 #005800 #004058 #000000 #bcbcbc #0078f8
#0058f8 #6844fc #d800cc #e40058 #f83800 #e45c10 #ac7c00 #00b800
#00a800 #00a844 #008888 #f8f8f8 #3cbcfc #6888fc #9878f8 #f878f8
#f85898 #f87858 #fca044 #f8b800 #b8f818 #58d854 #58f898 #00e8d8
#787878 #fcfcfc #a4e4fc #b8b8f8 #d8b8f8 #f8b8f8 #f8a4c0 #f0d0b0
#fce0a8 #f8d878 #d8f878 #b8f8b8 #b8f8d8 #00fcfc #f8d8f8
```

Tags: hardware, nes, analog.

[NES Classic source](https://fceux.com/web/help/Palette.html).

### PICO-8

`pico-8` · 16 colors · reference

The sixteen-color base palette of the PICO-8 fantasy console, in its original index order.

Colors: `#000000`, `#1d2b53`, `#7e2553`, `#008751`, `#ab5236`, `#5f574f`, `#c2c3c7`, `#fff1e8`, `#ff004d`, `#ffa300`, `#ffec27`, `#00e436`, `#29adff`, `#83769c`, `#ff77a8`, `#ffccaa`

Tags: fantasy-console, 16-color, pixel-art.

[PICO-8 source](https://www.lexaloffle.com/dl/docs/pico-8_manual.html).

### Plan 9 RGBV 256

`plan9-256` · 256 colors · reference

The exact Go standard-library Plan 9 RGBV table provides 256 colors, with sixteen gray levels and useful continuous-tone shading.

Colors, in palette order:

```text
#000000 #000044 #000088 #0000cc #004400 #004444 #004488 #0044cc
#008800 #008844 #008888 #0088cc #00cc00 #00cc44 #00cc88 #00cccc
#00dddd #111111 #000055 #000099 #0000dd #005500 #005555 #004c99
#0049dd #009900 #00994c #009999 #0093dd #00dd00 #00dd49 #00dd93
#00ee9e #00eeee #222222 #000066 #0000aa #0000ee #006600 #006666
#0055aa #004fee #00aa00 #00aa55 #00aaaa #009eee #00ee00 #00ee4f
#00ff55 #00ffaa #00ffff #333333 #000077 #0000bb #0000ff #007700
#007777 #005dbb #0055ff #00bb00 #00bb5d #00bbbb #00aaff #00ff00
#440044 #440088 #4400cc #444400 #444444 #444488 #4444cc #448800
#448844 #448888 #4488cc #44cc00 #44cc44 #44cc88 #44cccc #440000
#550000 #550055 #4c0099 #4900dd #555500 #555555 #4c4c99 #4949dd
#4c9900 #4c994c #4c9999 #4993dd #49dd00 #49dd49 #49dd93 #49dddd
#4feeee #660000 #660066 #5500aa #4f00ee #666600 #666666 #5555aa
#4f4fee #55aa00 #55aa55 #55aaaa #4f9eee #4fee00 #4fee4f #4fee9e
#55ffaa #55ffff #770000 #770077 #5d00bb #5500ff #777700 #777777
#5d5dbb #5555ff #5dbb00 #5dbb5d #5dbbbb #55aaff #55ff00 #55ff55
#880088 #8800cc #884400 #884444 #884488 #8844cc #888800 #888844
#888888 #8888cc #88cc00 #88cc44 #88cc88 #88cccc #880000 #880044
#99004c #990099 #9300dd #994c00 #994c4c #994c99 #9349dd #999900
#99994c #999999 #9393dd #93dd00 #93dd49 #93dd93 #93dddd #990000
#aa0000 #aa0055 #aa00aa #9e00ee #aa5500 #aa5555 #aa55aa #9e4fee
#aaaa00 #aaaa55 #aaaaaa #9e9eee #9eee00 #9eee4f #9eee9e #9eeeee
#aaffff #bb0000 #bb005d #bb00bb #aa00ff #bb5d00 #bb5d5d #bb5dbb
#aa55ff #bbbb00 #bbbb5d #bbbbbb #aaaaff #aaff00 #aaff55 #aaffaa
#cc00cc #cc4400 #cc4444 #cc4488 #cc44cc #cc8800 #cc8844 #cc8888
#cc88cc #cccc00 #cccc44 #cccc88 #cccccc #cc0000 #cc0044 #cc0088
#dd0093 #dd00dd #dd4900 #dd4949 #dd4993 #dd49dd #dd9300 #dd9349
#dd9393 #dd93dd #dddd00 #dddd49 #dddd93 #dddddd #dd0000 #dd0049
#ee004f #ee009e #ee00ee #ee4f00 #ee4f4f #ee4f9e #ee4fee #ee9e00
#ee9e4f #ee9e9e #ee9eee #eeee00 #eeee4f #eeee9e #eeeeee #ee0000
#ff0000 #ff0055 #ff00aa #ff00ff #ff5500 #ff5555 #ff55aa #ff55ff
#ffaa00 #ffaa55 #ffaaaa #ffaaff #ffff00 #ffff55 #ffffaa #ffffff
```

Tags: historical, 256-color, rgbv.

[Plan 9 RGBV 256 source](https://pkg.go.dev/image/color/palette#Plan9).

### RGB Workbench 64

`rgb-workbench-64` · 64 colors · original

A four-level RGB cube gives pixel artists sixty-four predictable combinations for colorful indexed studies.

Colors, in palette order:

```text
#000000 #000055 #0000aa #0000ff #005500 #005555 #0055aa #0055ff
#00aa00 #00aa55 #00aaaa #00aaff #00ff00 #00ff55 #00ffaa #00ffff
#550000 #550055 #5500aa #5500ff #555500 #555555 #5555aa #5555ff
#55aa00 #55aa55 #55aaaa #55aaff #55ff00 #55ff55 #55ffaa #55ffff
#aa0000 #aa0055 #aa00aa #aa00ff #aa5500 #aa5555 #aa55aa #aa55ff
#aaaa00 #aaaa55 #aaaaaa #aaaaff #aaff00 #aaff55 #aaffaa #aaffff
#ff0000 #ff0055 #ff00aa #ff00ff #ff5500 #ff5555 #ff55aa #ff55ff
#ffaa00 #ffaa55 #ffaaaa #ffaaff #ffff00 #ffff55 #ffffaa #ffffff
```

Tags: color-cube, 64-color, indexed.

### Sprite Adventure

`sprite-adventure` · 12 colors · original

Twelve storybook sprite colors combine forest green, brick red and clear blue for tiny scenes.

Colors: `#202136`, `#44405e`, `#73658b`, `#b3a0b9`, `#edd8cf`, `#fff1d8`, `#7b3048`, `#c45555`, `#ec9d6c`, `#365b57`, `#69a47c`, `#83c2cb`

Tags: pixel-art, storybook, bright.

### Teletext RGB

`teletext` · 8 colors · reference

The eight ideal binary RGB combinations used by classic teletext character generators, from black to white.

Colors: `#000000`, `#ff0000`, `#00ff00`, `#ffff00`, `#0000ff`, `#ff00ff`, `#00ffff`, `#ffffff`

Tags: hardware, teletext, primary.

[Teletext RGB source](https://github.com/mamedev/mame/blob/master/src/devices/video/saa5050.cpp).

### ZX Spectrum

`zx-spectrum` · 15 colors · approximation

The palette approximates fifteen unique normal and bright Spectrum colors in sRGB. Shared black appears once.

Colors: `#000000`, `#0000d7`, `#d70000`, `#d700d7`, `#00d700`, `#00d7d7`, `#d7d700`, `#d7d7d7`, `#0000ff`, `#ff0000`, `#ff00ff`, `#00ff00`, `#00ffff`, `#ffff00`, `#ffffff`

Tags: hardware, primary, spectrum.

[ZX Spectrum source](https://worldofspectrum.org/ZXBasicManual/zxmanchap16.html).

## Seasonal

### April Garden

`april-garden` · 7 colors · original

Fresh lime leaves, spring teal and pale yellow light brighten mauve soil and pink emerging flowers.

Colors: `#294a42`, `#598775`, `#99b788`, `#d3ddad`, `#f5efcf`, `#7c666f`, `#c292a4`

Tags: spring, garden, fresh.

### August Cicada

`august-cicada` · 8 colors · original

Hot ochre sunlight and dry grass cross olive shade, brown branches and the blue of a late-summer sky.

Colors: `#36433c`, `#6c7751`, `#a9a866`, `#d4c37b`, `#f0dfb0`, `#86563d`, `#bd8856`, `#779caa`

Tags: summer, ochre, grass.

### Autumn Orchard

`autumn-orchard` · 8 colors · original

Apple red, olive leaves and ochre grass share plum shadow and cream harvest light.

Colors: `#30262f`, `#674143`, `#a35245`, `#cc8757`, `#e5b578`, `#f4dfad`, `#596143`, `#929258`

Tags: autumn, orchard, earth.

### Cherry Blossom

`cherry-blossom` · 8 colors · original

Dark cherry branches, soft pink petals and blue spring air retain green buds and warm cream.

Colors: `#423341`, `#856374`, `#bd929b`, `#e8bac4`, `#f5dce0`, `#faf0df`, `#809da6`, `#a9bb91`

Tags: spring, blossom, pink.

### December Pine

`december-pine` · 8 colors · original

Deep fir green, icy sage and snow cream share quiet cranberry and gold holiday accents.

Colors: `#153b30`, `#356653`, `#6b9379`, `#adc5a0`, `#e0e6c8`, `#f4edce`, `#8b4050`, `#d3b070`

Tags: winter, pine, green.

### First Snow

`first-snow` · 7 colors · original

Slate forest and dusky violet lighten through powder blue to near-white snow, with a warm bark accent.

Colors: `#243744`, `#516773`, `#8295a4`, `#b6cad2`, `#dfe9e5`, `#f9f4e4`, `#867568`

Tags: winter, snow, slate.

### January Dawn

`january-dawn` · 8 colors · original

Muted indigo and blue-gray snow catch the quiet pink and peach light of a cold morning.

Colors: `#29354f`, `#566985`, `#92a9bd`, `#c8dbe0`, `#eaf0e4`, `#9f829d`, `#d5b0b3`, `#eaceb4`

Tags: winter, dawn, blue.

### Late Harvest

`late-harvest` · 8 colors · original

Wine-purple berries and bronze fields sit against olive leaves and soft autumn paper.

Colors: `#342539`, `#694258`, `#9e685f`, `#c29865`, `#e5c890`, `#f1dfbd`, `#535c38`, `#98995f`

Tags: autumn, harvest, wine.

### May Pollen

`may-pollen` · 7 colors · original

Lime-green leaves, pollen yellow and soft cream meet warm brown stems and pale blue spring air.

Colors: `#38583f`, `#739b57`, `#b8ce7c`, `#e5e899`, `#f9f1c7`, `#8d7154`, `#a9c9c6`

Tags: spring, pollen, yellow.

### Midsummer Field

`midsummer-field` · 8 colors · original

Warm straw, meadow green and clear blue share poppy-red accents for a sunny open-field study.

Colors: `#365446`, `#729b61`, `#b7ca83`, `#e8dc9f`, `#f6edc7`, `#688eb0`, `#abcadd`, `#c87869`

Tags: summer, meadow, straw.

### November Fog

`november-fog` · 7 colors · original

Rain-dark brown and olive fade into gray mist, pale straw and a small muted brick accent.

Colors: `#303735`, `#5f6760`, `#92998b`, `#c0c2ad`, `#e8e4cc`, `#805f54`, `#b18a6c`

Tags: autumn, fog, muted.

### October Lantern

`october-lantern` · 7 colors · original

Charcoal violet, pumpkin orange and golden candlelight glow beside deep pine and warm cream.

Colors: `#282337`, `#574051`, `#98554b`, `#d18742`, `#ecb663`, `#f5df9f`, `#45634b`

Tags: autumn, pumpkin, night.

### September Plum

`september-plum` · 8 colors · original

Ripe purple fruit, dusty rose and gold leaves sit beside cool green shade and creamy daylight.

Colors: `#30273f`, `#665074`, `#9b7193`, `#c8a0b3`, `#e6c8cb`, `#f1e4c9`, `#667a53`, `#b4ad68`

Tags: autumn, plum, orchard.

### Spring Rain

`spring-rain` · 8 colors · original

Fresh green shoots and cool rain-blue shadows blend with lilac petals and pale overcast light.

Colors: `#28414a`, `#527b80`, `#91a9a5`, `#ced5bd`, `#ecedd2`, `#687c52`, `#a6bc77`, `#b999ba`

Tags: spring, rain, green.

### Summer Solstice

`summer-solstice` · 8 colors · original

Leaf green, lake blue and golden midsummer sunshine make a bright long-day palette.

Colors: `#254d48`, `#4e9175`, `#96c489`, `#d4e8af`, `#f8ecc8`, `#3276a2`, `#6bb3d1`, `#e7b15e`

Tags: summer, sun, bright.

### Winter Solstice

`winter-solstice` · 7 colors · original

Blue-black winter night, frozen aqua and pearl snow hold one quiet gold light.

Colors: `#17243c`, `#3b4b69`, `#6a809a`, `#a7bac8`, `#dce4df`, `#f5f0d7`, `#bca36a`

Tags: winter, night, snow.

## Terminal

### Amber terminal

`amber` · 4 colors · original

Four amber phosphor tones contrast with a nearly black warm screen.

Colors: `#160d00`, `#7c4100`, `#e78a12`, `#ffd77c`

Tags: phosphor, amber, monochrome.

### Blue Phosphor

`blue-phosphor` · 4 colors · original

Four electric-blue tones contrast with navy for technical diagrams and nighttime silhouettes.

Colors: `#050d20`, `#163d78`, `#3f8ac7`, `#a9dfff`

Tags: phosphor, blue, tonal.

### Cassette Console

`cassette-console` · 6 colors · original

Warm gray, moss and cream temper a green display ramp with the muted color of old audio hardware.

Colors: `#1b2320`, `#414e43`, `#69775c`, `#98a17c`, `#c5c7a5`, `#ece5c8`

Tags: audio, muted, green.

### Diagnostic Ice

`diagnostic-ice` · 6 colors · original

Six cool signal tones add distinct teal and ice-blue steps to a clear technical image.

Colors: `#091b28`, `#244157`, `#376976`, `#649bad`, `#a0cad0`, `#e6f4ec`

Tags: technical, ice, cyan.

### Marine Console

`marine-console` · 4 colors · original

A deep blue-green screen ramp uses frosted cyan highlights for maritime navigation graphics.

Colors: `#081825`, `#254e65`, `#6197ad`, `#c2e7e9`

Tags: marine, cyan, technical.

### Mission Control

`mission-control` · 8 colors · original

Eight night-console colors mix teal readouts, amber alerts and cool pale labels.

Colors: `#0b1627`, `#27384d`, `#4c5c70`, `#819aa5`, `#c4d7d4`, `#eef3de`, `#47b4a0`, `#e5af5f`

Tags: technical, multicolor, night.

### Orange Phosphor

`orange-phosphor` · 4 colors · original

Burnt orange through apricot creates a warm tonal range inspired by monochrome screens.

Colors: `#1e0b04`, `#813311`, `#d96d25`, `#ffbd76`

Tags: phosphor, orange, warm.

### Paper Terminal

`paper-terminal` · 4 colors · original

Ink, olive-gray and creamy paper turn terminal-inspired artwork into a quiet printed page.

Colors: `#292b25`, `#676c59`, `#a7ad8c`, `#e5e7c5`

Tags: paper, olive, tonal.

### Plasma Terminal

`plasma-terminal` · 4 colors · original

Warm maroon, coral and ivory give text and line art the radiance of a plasma-panel study.

Colors: `#28141b`, `#77363c`, `#c77663`, `#ffceb0`

Tags: plasma, coral, warm.

### Radar Sweep

`radar-sweep` · 4 colors · original

Dark bottle green, olive glow and chartreuse highlights evoke a compact sweep display.

Colors: `#09190e`, `#324e1f`, `#88a52f`, `#d9ed82`

Tags: radar, green, signal.

### Red Phosphor

`red-phosphor` · 4 colors · original

Wine-black, vermilion and pale rose form an urgent red monitor-style tonal ramp.

Colors: `#210810`, `#73253b`, `#cc5166`, `#ffa6ae`

Tags: phosphor, red, tonal.

### Teal Phosphor

`teal-phosphor` · 4 colors · original

Petrol shadows brighten through turquoise and cool mint for clean marine-console texture.

Colors: `#071a1c`, `#1b565a`, `#45a5a4`, `#b7ece0`

Tags: phosphor, teal, cool.

### Green phosphor

`terminal` · 4 colors · original

Deep green through pale mint recreates the mood of a glowing text terminal.

Colors: `#041408`, `#134d24`, `#39a95d`, `#b7f7b0`

Tags: phosphor, green, monochrome.

### Vacuum Tube

`vacuum-tube` · 6 colors · original

Six reddish-brown and amber shades suggest glowing valves behind smoked glass.

Colors: `#1a1112`, `#493039`, `#7b4b48`, `#ae7356`, `#d6a47a`, `#f2d7a3`

Tags: amber, brown, industrial.

### Violet Phosphor

`violet-phosphor` · 4 colors · original

Inky plum and luminous lavender create a soft ultraviolet monitor mood.

Colors: `#160b29`, `#4f327b`, `#9979ce`, `#e1c6fa`

Tags: phosphor, violet, night.

### White Phosphor

`white-phosphor` · 4 colors · original

Cool charcoal and silver-white tones give monochrome artwork the appearance of a laboratory display.

Colors: `#10171b`, `#3e4c53`, `#879a9f`, `#e4f2ef`

Tags: phosphor, cool, grayscale.
