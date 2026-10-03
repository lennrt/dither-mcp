# Metadata fixtures

`linear-srgb.icc` is an original synthetic ICC v4 RGB matrix/TRC profile.
Its identity curves describe linear samples with sRGB primaries. The D50-adapted
colorant columns are (0.4360747, 0.2225045, 0.0139322),
(0.3850649, 0.7168786, 0.0971045), and (0.1430804, 0.0606169, 0.7141733).
Values use signed 16.16 encoding. It contains no personal or third-party metadata.
It is covered by the project license.

Tests embed these same bytes in independent container layouts. A neutral sample
of 128/255 in this linear profile becomes approximately 188/255 in sRGB.

`oriented.webp` contains an original 2-by-3 RGB grid, orientation 6, and the
linear-sRGB fixture profile. Pillow 12.3.0 encoded the lossless WebP file.
The colors in row order are (20,40,60), (60,80,100), (100,120,140),
(140,160,180), (180,200,220), and (220,240,250).
