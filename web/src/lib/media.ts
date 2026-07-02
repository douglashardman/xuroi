/** Matches Go media.ValidMediaName — post images + avatars. */
export const MEDIA_NAME_RE = /^(med_|avt_)[0-9a-z]{26}(_thumb|_sm)?\.webp$/;

export function isValidMediaFilename(name: string): boolean {
  return MEDIA_NAME_RE.test(name);
}