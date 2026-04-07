import { describe, expect, it } from 'vitest';

import { detectDesktopClient, pickDesktopDownload } from './desktop-download';
import type { DesktopDownloadVariant } from './api';

describe('desktop download detection', () => {
  it('detects windows x64 from common user agent', () => {
    const detected = detectDesktopClient({
      platform: 'Win32',
      userAgent: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64)',
    });

    expect(detected.os).toBe('windows');
    expect(detected.key).toBe('windows-x64');
  });

  it('detects macOS arm64 from userAgentData', () => {
    const detected = detectDesktopClient({
      userAgentData: { platform: 'macOS', architecture: 'arm64' },
      userAgent: 'Mozilla/5.0',
    });

    expect(detected.os).toBe('macos');
    expect(detected.arch).toBe('arm64');
    expect(detected.key).toBe('macos-arm64');
  });

  it('detects macOS intel from user agent fallback', () => {
    const detected = detectDesktopClient({
      platform: 'MacIntel',
      userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)',
    });

    expect(detected.os).toBe('macos');
    expect(detected.key).toBe('macos-amd64');
  });
});

describe('desktop download recommendation', () => {
  const variants: DesktopDownloadVariant[] = [
    {
      id: 'windows-x64',
      os: 'windows',
      arch: 'x64',
      label: 'Windows 11 x64',
      url: 'https://downloads.example.com/md/windows-x64.exe',
      available: true,
    },
    {
      id: 'macos-arm64',
      os: 'macos',
      arch: 'arm64',
      label: 'macOS Apple Silicon',
      url: 'https://downloads.example.com/md/macos-arm64.zip',
      available: true,
    },
    {
      id: 'macos-amd64',
      os: 'macos',
      arch: 'x64',
      label: 'macOS Intel',
      url: 'https://downloads.example.com/md/macos-amd64.zip',
      available: true,
    },
  ];

  it('picks exact match when available', () => {
    const recommended = pickDesktopDownload(variants, { os: 'windows', arch: 'x64', key: 'windows-x64' });

    expect(recommended?.id).toBe('windows-x64');
  });

  it('falls back to same OS when exact key is missing', () => {
    const windowsUnavailable = variants.map((variant) =>
      variant.id === 'windows-x64' ? { ...variant, available: false, url: '' } : variant,
    );

    const recommended = pickDesktopDownload(windowsUnavailable, { os: 'windows', arch: 'x64', key: 'windows-x64' });

    expect(recommended?.id).toBe('macos-arm64');
  });

  it('returns null when no download is available', () => {
    const none = variants.map((variant) => ({ ...variant, available: false, url: '' }));

    const recommended = pickDesktopDownload(none, { os: 'linux', arch: 'x64', key: 'linux-x64' });

    expect(recommended).toBeNull();
  });
});
