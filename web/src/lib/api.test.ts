import { describe, expect, it } from 'vitest';

import { api } from './api';

describe('api export URL builders', () => {
  it('builds pdf export URL with margin, header and footer', () => {
    const url = api.exportFormat('file-123', 'pdf', {
      margin: 'standard',
      header: '  Confidential Report  ',
      footer: 'Internal only',
      headerAlign: 'right',
      footerAlign: 'center',
      h1UnderlineColor: '#10b981',
      headingTextColor: '#111111',
      headingFont: 'serif',
    });

    expect(url).toContain('/api/files/file-123/export/pdf?');
    expect(url).toContain('margin=standard');
    expect(url).toContain('header=Confidential+Report');
    expect(url).toContain('footer=Internal+only');
    expect(url).toContain('header_align=right');
    expect(url).toContain('footer_align=center');
    expect(url).toContain('h1_underline_color=%2310b981');
    expect(url).toContain('heading_text_color=%23111111');
    expect(url).toContain('heading_font=serif');
  });

  it('passes heading style options to non-pdf exports', () => {
    const url = api.exportFormat('file-1', 'docx', {
      headingTextColor: '#000000',
      headingFont: 'mono',
    });
    expect(url).toContain('/api/files/file-1/export/docx?');
    expect(url).toContain('heading_text_color=%23000000');
    expect(url).toContain('heading_font=mono');
  });

  it('keeps non-pdf export URLs unchanged', () => {
    expect(api.exportFormat('file-1', 'docx', { margin: 'wide' })).toBe('/api/files/file-1/export/docx');
    expect(api.exportRawFormat('epub', { margin: 'narrow', header: 'ignored' })).toBe('/api/export/raw/epub');
  });

  it('supports legacy margin string for backward compatibility', () => {
    expect(api.exportFormat('legacy', 'pdf', 'narrow')).toBe('/api/files/legacy/export/pdf?margin=narrow');
    expect(api.exportRawFormat('pdf', 'wide')).toBe('/api/export/raw/pdf?margin=wide');
  });

  it('truncates long header and footer values', () => {
    const long = 'x'.repeat(140);
    const url = api.exportRawFormat('pdf', { header: long, footer: long });
    const params = new URLSearchParams(url.split('?')[1]);

    expect(params.get('header')?.length).toBe(120);
    expect(params.get('footer')?.length).toBe(120);
  });
});
