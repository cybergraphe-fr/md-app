import { describe, expect, it } from 'vitest';

import { resizeColumns } from './table-resize';

describe('resizeColumns', () => {
  it('déplace la largeur entre les deux colonnes en gardant le total constant', () => {
    const r = resizeColumns(200, 200, 50);
    expect(r.width).toBe(250);
    expect(r.nextWidth).toBe(150);
    expect(r.width + r.nextWidth).toBe(400);
  });

  it('fonctionne avec un delta négatif', () => {
    const r = resizeColumns(200, 200, -60);
    expect(r.width).toBe(140);
    expect(r.nextWidth).toBe(260);
  });

  it('borne la colonne déplacée au minimum', () => {
    const r = resizeColumns(200, 200, -500, 48);
    expect(r.width).toBe(48);
    expect(r.nextWidth).toBe(352);
  });

  it('borne la colonne voisine au minimum', () => {
    const r = resizeColumns(200, 200, 500, 48);
    expect(r.width).toBe(352);
    expect(r.nextWidth).toBe(48);
  });

  it('préserve toujours la somme des deux largeurs', () => {
    for (const delta of [-1000, -10, 0, 37, 1000]) {
      const r = resizeColumns(120, 300, delta, 48);
      expect(r.width + r.nextWidth).toBeCloseTo(420);
      expect(r.width).toBeGreaterThanOrEqual(48);
      expect(r.nextWidth).toBeGreaterThanOrEqual(48);
    }
  });

  it('partage à parts égales si le total est trop petit pour deux minimums', () => {
    const r = resizeColumns(30, 30, 100, 48);
    expect(r.width).toBe(30);
    expect(r.nextWidth).toBe(30);
  });
});
