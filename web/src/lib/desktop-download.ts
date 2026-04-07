import type { DesktopDownloadVariant } from './api';

export type DetectedDesktopOS = 'windows' | 'macos' | 'linux' | 'unknown';
export type DetectedDesktopArch = 'x64' | 'arm64' | 'unknown';

export interface DetectedDesktopClient {
  os: DetectedDesktopOS;
  arch: DetectedDesktopArch;
  key: string;
}

interface MinimalNavigator {
  platform?: string;
  userAgent?: string;
  userAgentData?: {
    platform?: string;
    architecture?: string;
  };
}

function detectOS(platformHint: string, userAgent: string): DetectedDesktopOS {
  if (/win/.test(platformHint) || /windows/.test(userAgent)) return 'windows';
  if (/mac/.test(platformHint) || /darwin/.test(platformHint) || /mac os x|macintosh/.test(userAgent)) return 'macos';
  if (/linux|x11/.test(platformHint) || /linux|x11/.test(userAgent)) return 'linux';
  return 'unknown';
}

function detectArch(archHint: string, userAgent: string): DetectedDesktopArch {
  if (/arm64|aarch64|arm/.test(archHint) || /arm64|aarch64/.test(userAgent)) return 'arm64';
  if (/x64|x86_64|amd64|x86|intel/.test(archHint) || /x64|x86_64|amd64|win64|x86_64|intel/.test(userAgent)) return 'x64';
  return 'unknown';
}

export function detectDesktopClient(nav: MinimalNavigator | null | undefined = globalThis.navigator): DetectedDesktopClient {
  const platformHint = `${nav?.userAgentData?.platform ?? ''} ${nav?.platform ?? ''}`.toLowerCase();
  const archHint = `${nav?.userAgentData?.architecture ?? ''}`.toLowerCase();
  const userAgent = (nav?.userAgent ?? '').toLowerCase();

  const os = detectOS(platformHint, userAgent);
  let arch = detectArch(archHint, userAgent);

  if (os === 'windows' && arch === 'unknown') {
    arch = 'x64';
  }

  if (os === 'macos' && arch === 'unknown') {
    return { os, arch, key: 'macos-arm64' };
  }

  if (os === 'macos' && arch === 'x64') {
    return { os, arch, key: 'macos-amd64' };
  }

  if (os === 'macos' && arch === 'arm64') {
    return { os, arch, key: 'macos-arm64' };
  }

  if (os === 'windows' && arch === 'x64') {
    return { os, arch, key: 'windows-x64' };
  }

  if (os === 'linux' && arch === 'x64') {
    return { os, arch, key: 'linux-x64' };
  }

  return { os, arch, key: 'unknown' };
}

function archMatches(variantArch: string, detectedArch: DetectedDesktopArch): boolean {
  const normalized = variantArch.toLowerCase();
  if (detectedArch === 'x64') {
    return normalized === 'x64' || normalized === 'amd64' || normalized === 'x86_64';
  }
  if (detectedArch === 'arm64') {
    return normalized === 'arm64' || normalized === 'aarch64';
  }
  return false;
}

export function pickDesktopDownload(
  variants: DesktopDownloadVariant[],
  detected: DetectedDesktopClient,
): DesktopDownloadVariant | null {
  const available = variants.filter((variant) => variant.available && !!variant.url);
  if (available.length === 0) return null;

  const exact = available.find((variant) => variant.id === detected.key);
  if (exact) return exact;

  if (detected.os === 'macos' && detected.arch === 'unknown') {
    const preferredMac = available.find((variant) => variant.id === 'macos-arm64');
    if (preferredMac) return preferredMac;
  }

  const sameOSArch = available.find((variant) => variant.os === detected.os && archMatches(variant.arch, detected.arch));
  if (sameOSArch) return sameOSArch;

  const sameOS = available.find((variant) => variant.os === detected.os);
  if (sameOS) return sameOS;

  return available[0];
}

export function desktopClientLabel(detected: DetectedDesktopClient): string {
  if (detected.os === 'windows') return 'Windows';
  if (detected.os === 'macos') return detected.arch === 'x64' ? 'macOS Intel' : 'macOS';
  if (detected.os === 'linux') return 'Linux';
  return 'OS inconnu';
}
