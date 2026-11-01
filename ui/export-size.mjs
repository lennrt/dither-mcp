// Keep the accepted preview ratio. Export validation enforces server limits.
export function proportionalSize(preview, axis, value) {
  if (!['width', 'height'].includes(axis) || !Number.isSafeInteger(value) || value <= 0 ||
      !Number.isSafeInteger(preview?.width) || preview.width <= 0 ||
      !Number.isSafeInteger(preview?.height) || preview.height <= 0) return null;
  const other = axis === 'width' ? 'height' : 'width';
  const paired = Math.max(1, Math.round(value * preview[other] / preview[axis]));
  if (!Number.isSafeInteger(paired)) return null;
  return { [axis]: value, [other]: paired };
}
