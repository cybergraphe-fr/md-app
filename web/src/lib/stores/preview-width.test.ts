import { describe, expect, it } from 'vitest';

import {
  normalizePreviewWidth,
  previewWidthPresets,
  PREVIEW_WIDTH_MIN,
  PREVIEW_WIDTH_MAX,
} from './files';

describe('normalizePreviewWidth', () => {
  it('garde la valeur sentinelle "full"', () => {
    expect(normalizePreviewWidth('full')).toBe('full');
  });

  it('borne en bas à PREVIEW_WIDTH_MIN', () => {
    expect(normalizePreviewWidth(100)).toBe(PREVIEW_WIDTH_MIN);
  });

  it('borne en haut à PREVIEW_WIDTH_MAX', () => {
    expect(normalizePreviewWidth(99999)).toBe(PREVIEW_WIDTH_MAX);
  });

  it('conserve une valeur dans la plage et arrondit', () => {
    expect(normalizePreviewWidth(1024)).toBe(1024);
    expect(normalizePreviewWidth(1023.6)).toBe(1024);
  });

  it('accepte une chaîne numérique', () => {
    expect(normalizePreviewWidth('800')).toBe(800);
  });

  it('retombe sur le défaut (780) pour une entrée invalide', () => {
    expect(normalizePreviewWidth('abc')).toBe(780);
    expect(normalizePreviewWidth(NaN)).toBe(780);
    expect(normalizePreviewWidth(null)).toBe(780);
    expect(normalizePreviewWidth(undefined)).toBe(780);
  });

  it('les presets sont tous valides après normalisation', () => {
    for (const preset of previewWidthPresets) {
      expect(normalizePreviewWidth(preset.value)).toBe(preset.value);
    }
  });
});
