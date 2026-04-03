export function isMermaidLanguage(lang?: string): boolean {
  const normalized = (lang ?? '').trim().toLowerCase().split(/\s+/)[0];
  return normalized === 'mermaid';
}

export function hasMermaidFence(content: string): boolean {
  return /^[\t ]*(?:```|~~~)\s*mermaid\b/im.test(content);
}