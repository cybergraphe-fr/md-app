<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { EditorView, keymap, lineNumbers, drawSelection, dropCursor } from '@codemirror/view';
  import { EditorState, Compartment, type Extension } from '@codemirror/state';
  import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands';
  import { searchKeymap, highlightSelectionMatches } from '@codemirror/search';
  import { markdown, markdownLanguage } from '@codemirror/lang-markdown';
  import { languages } from '@codemirror/language-data';
  import { syntaxHighlighting, HighlightStyle, indentOnInput, foldGutter } from '@codemirror/language';
  import { tags as t } from '@lezer/highlight';
  import { autocompletion, completionKeymap } from '@codemirror/autocomplete';
  import {
    activeContent,
    setContent,
    theme,
    formatAction,
    type FormatActionKind,
  } from '$lib/stores/files';

  const darkHighlightStyle = HighlightStyle.define([
    { tag: t.keyword, color: '#c678dd' },
    { tag: [t.name, t.deleted, t.character, t.macroName], color: '#e06c75' },
    { tag: [t.function(t.variableName), t.labelName], color: '#61afef' },
    { tag: [t.color, t.constant(t.name), t.standard(t.name)], color: '#d19a66' },
    { tag: [t.definition(t.name), t.separator], color: '#abb2bf' },
    { tag: [t.typeName, t.className, t.number, t.changed, t.annotation, t.modifier, t.self, t.namespace], color: '#e5c07b' },
    { tag: [t.operator, t.operatorKeyword, t.url, t.escape, t.regexp, t.link, t.special(t.string)], color: '#56b6c2' },
    { tag: [t.meta, t.comment], color: '#7f848e' },
    { tag: t.strong, fontWeight: 'bold', color: '#e5c07b' },
    { tag: t.emphasis, fontStyle: 'italic', color: '#c678dd' },
    { tag: t.strikethrough, textDecoration: 'line-through' },
    { tag: t.link, color: '#61afef', textDecoration: 'underline' },
    { tag: t.heading, fontWeight: 'bold', color: '#e06c75' },
    { tag: [t.atom, t.bool, t.special(t.variableName)], color: '#d19a66' },
    { tag: [t.processingInstruction, t.string, t.inserted], color: '#98c379' },
    { tag: t.invalid, color: '#ffffff', backgroundColor: '#e06c75' },
  ]);

  const lightHighlightStyle = HighlightStyle.define([
    { tag: t.keyword, color: '#a626a4' },
    { tag: [t.name, t.deleted, t.character, t.macroName], color: '#e45649' },
    { tag: [t.function(t.variableName), t.labelName], color: '#4078f2' },
    { tag: [t.color, t.constant(t.name), t.standard(t.name)], color: '#986801' },
    { tag: [t.definition(t.name), t.separator], color: '#383a42' },
    { tag: [t.typeName, t.className, t.number, t.changed, t.annotation, t.modifier, t.self, t.namespace], color: '#c18401' },
    { tag: [t.operator, t.operatorKeyword, t.url, t.escape, t.regexp, t.link, t.special(t.string)], color: '#0184bc' },
    { tag: [t.meta, t.comment], color: '#a0a1a7' },
    { tag: t.strong, fontWeight: 'bold', color: '#c18401' },
    { tag: t.emphasis, fontStyle: 'italic', color: '#a626a4' },
    { tag: t.strikethrough, textDecoration: 'line-through' },
    { tag: t.link, color: '#4078f2', textDecoration: 'underline' },
    { tag: t.heading, fontWeight: 'bold', color: '#e45649' },
    { tag: [t.atom, t.bool, t.special(t.variableName)], color: '#986801' },
    { tag: [t.processingInstruction, t.string, t.inserted], color: '#50a14f' },
    { tag: t.invalid, color: '#ffffff', backgroundColor: '#e45649' },
  ]);

  let container: HTMLDivElement;
  let view: EditorView | undefined;
  const themeCompartment = new Compartment();
  const highlightCompartment = new Compartment();

  function buildTheme(dark: boolean): Extension {
    return EditorView.theme(
      {
        '&': {
          backgroundColor: 'transparent',
          color: dark ? '#e4e4e7' : '#18181b',
          height: '100%',
        },
        '.cm-content': {
          caretColor: dark ? '#8b5cf6' : '#7c3aed',
          fontFamily: 'var(--font-mono)',
          fontSize: '14px',
          lineHeight: '1.7',
          padding: '0.75rem 0',
        },
        '.cm-cursor': { borderLeftColor: dark ? '#8b5cf6' : '#7c3aed', borderLeftWidth: '2px' },
        '.cm-gutters': {
          backgroundColor: 'transparent',
          color: dark ? '#3f3f46' : '#a1a1aa',
          borderRight: `1px solid ${dark ? 'rgba(255,255,255,0.04)' : 'rgba(0,0,0,0.06)'}`,
          minWidth: '3.2rem',
        },
        '.cm-gutter': { fontSize: '12px' },
        '.cm-activeLine': {
          backgroundColor: dark ? 'rgba(255,255,255,0.03)' : 'rgba(0,0,0,0.02)',
        },
        '.cm-activeLineGutter': {
          backgroundColor: dark ? 'rgba(255,255,255,0.03)' : 'rgba(0,0,0,0.02)',
          color: dark ? '#71717a' : '#52525b',
        },
        '.cm-selectionBackground, ::selection': {
          backgroundColor: dark ? 'rgba(139,92,246,0.2) !important' : 'rgba(124,58,237,0.12) !important',
        },
        '.cm-matchingBracket': {
          color: dark ? '#c4b5fd' : '#7c3aed',
          fontWeight: '600',
          backgroundColor: dark ? 'rgba(139,92,246,0.15)' : 'rgba(124,58,237,0.1)',
          borderRadius: '2px',
        },
        '.cm-foldGutter': { color: dark ? '#3f3f46' : '#d4d4d8' },
        '.cm-tooltip': {
          backgroundColor: dark ? '#18181b' : '#ffffff',
          border: `1px solid ${dark ? 'rgba(255,255,255,0.08)' : 'rgba(0,0,0,0.08)'}`,
          borderRadius: '8px',
          boxShadow: dark ? '0 8px 32px rgba(0,0,0,0.5)' : '0 8px 32px rgba(0,0,0,0.12)',
        },
        '.cm-tooltip-autocomplete ul li[aria-selected]': {
          backgroundColor: dark ? 'rgba(139,92,246,0.15)' : 'rgba(124,58,237,0.08)',
        },
        '.cm-panels': {
          backgroundColor: dark ? '#18181b' : '#fafafa',
          borderBottom: `1px solid ${dark ? 'rgba(255,255,255,0.06)' : 'rgba(0,0,0,0.06)'}`,
        },
        '.cm-search': {
          fontSize: '13px',
        },
        '.cm-button': {
          backgroundImage: 'none',
          backgroundColor: dark ? 'rgba(255,255,255,0.06)' : 'rgba(0,0,0,0.04)',
          border: `1px solid ${dark ? 'rgba(255,255,255,0.1)' : 'rgba(0,0,0,0.1)'}`,
          borderRadius: '4px',
          color: dark ? '#e4e4e7' : '#18181b',
        },
        '.cm-textfield': {
          backgroundColor: dark ? 'rgba(255,255,255,0.05)' : '#ffffff',
          border: `1px solid ${dark ? 'rgba(255,255,255,0.1)' : 'rgba(0,0,0,0.1)'}`,
          borderRadius: '4px',
          color: dark ? '#e4e4e7' : '#18181b',
        },
      },
      { dark }
    );
  }

  function createExtensions(dark: boolean): Extension[] {
    return [
      lineNumbers(),
      foldGutter(),
      drawSelection(),
      dropCursor(),
      history(),
      indentOnInput(),
      highlightCompartment.of(
        syntaxHighlighting(dark ? darkHighlightStyle : lightHighlightStyle, { fallback: true })
      ),
      markdown({
        base: markdownLanguage,
        codeLanguages: languages,
        addKeymap: true,
      }),
      highlightSelectionMatches(),
      autocompletion(),
      themeCompartment.of(buildTheme(dark)),
      keymap.of([
        indentWithTab,
        ...defaultKeymap,
        ...historyKeymap,
        ...searchKeymap,
        ...completionKeymap,
      ]),
      keymap.of([
        {
          key: 'Ctrl-b',
          run: (v) => wrapSelection(v, '**'),
        },
        {
          key: 'Ctrl-i',
          run: (v) => wrapSelection(v, '_'),
        },
        {
          key: 'Ctrl-k',
          run: (v) => {
            const sel = v.state.sliceDoc(
              v.state.selection.main.from,
              v.state.selection.main.to
            );
            v.dispatch({
              changes: {
                from: v.state.selection.main.from,
                to: v.state.selection.main.to,
                insert: `[${sel}](url)`,
              },
            });
            return true;
          },
        },
      ]),
      EditorView.updateListener.of((update) => {
        if (update.docChanged) {
          setContent(update.state.doc.toString());
        }
      }),
      EditorView.lineWrapping,
    ];
  }

  function wrapSelection(v: EditorView, wrapper: string): boolean {
    const { from, to } = v.state.selection.main;
    const sel = v.state.sliceDoc(from, to);
    const hasSelection = from !== to;
    const fallbackText = wrapper === '`' ? 'code' : 'text';
    const value = hasSelection ? sel : fallbackText;
    v.dispatch({
      changes: { from, to, insert: `${wrapper}${value}${wrapper}` },
      selection: {
        anchor: from + wrapper.length,
        head: from + wrapper.length + value.length,
      },
    });
    return true;
  }

  function updateCurrentLine(v: EditorView, updater: (lineText: string) => { text: string; selectionOffset?: number }): void {
    const { from, to } = v.state.selection.main;
    const line = v.state.doc.lineAt(from);
    const updated = updater(line.text);
    const selectionOffset = updated.selectionOffset ?? 0;
    v.dispatch({
      changes: {
        from: line.from,
        to: line.to,
        insert: updated.text,
      },
      selection: {
        anchor: Math.max(line.from, from + selectionOffset),
        head: Math.max(line.from, to + selectionOffset),
      },
    });
  }

  function prefixLine(v: EditorView, prefix: string): void {
    updateCurrentLine(v, (lineText) => ({
      text: `${prefix}${lineText}`,
      selectionOffset: prefix.length,
    }));
  }

  function replaceHeading(v: EditorView, level: 1 | 2 | 3): void {
    const prefix = `${'#'.repeat(level)} `;
    updateCurrentLine(v, (lineText) => {
      const baseText = lineText.replace(/^#{1,6}\s+/, '');
      const text = baseText.length === 0 ? prefix : `${prefix}${baseText}`;
      return {
        text,
        selectionOffset: text.length - lineText.length,
      };
    });
  }

  function asParagraph(v: EditorView): void {
    updateCurrentLine(v, (lineText) => {
      const text = lineText
        .replace(/^#{1,6}\s+/, '')
        .replace(/^>\s+/, '')
        .replace(/^[-*+]\s+/, '')
        .replace(/^\d+\.\s+/, '')
        .replace(/^-\s+\[[ xX]\]\s+/, '');
      return {
        text,
        selectionOffset: text.length - lineText.length,
      };
    });
  }

  function insertLink(v: EditorView): void {
    const { from, to } = v.state.selection.main;
    const selection = v.state.sliceDoc(from, to).trim();
    const label = selection || 'link text';
    const insert = `[${label}](https://)`;
    const urlStart = from + insert.indexOf('https://');
    v.dispatch({
      changes: { from, to, insert },
      selection: { anchor: urlStart, head: urlStart + 'https://'.length },
    });
  }

  function insertCodeBlock(v: EditorView): void {
    const { from, to } = v.state.selection.main;
    const selection = v.state.sliceDoc(from, to);
    const hasSelection = selection.trim().length > 0;
    const body = hasSelection ? `\n${selection}\n` : '\ncode\n';
    const insert = `\`\`\`\n${body}\`\`\``;
    const anchor = from + 4;
    const head = hasSelection ? anchor + selection.length : anchor + 4;
    v.dispatch({
      changes: { from, to, insert },
      selection: { anchor, head },
    });
  }

  function applyFormat(kind: FormatActionKind): void {
    if (!view) return;
    switch (kind) {
      case 'bold':
        wrapSelection(view, '**');
        break;
      case 'italic':
        wrapSelection(view, '_');
        break;
      case 'underline': {
        const { from, to } = view.state.selection.main;
        const selection = view.state.sliceDoc(from, to);
        const value = selection || 'text';
        const insert = `<u>${value}</u>`;
        view.dispatch({
          changes: { from, to, insert },
          selection: { anchor: from + 3, head: from + 3 + value.length },
        });
        break;
      }
      case 'strike':
        wrapSelection(view, '~~');
        break;
      case 'heading1':
        replaceHeading(view, 1);
        break;
      case 'heading2':
        replaceHeading(view, 2);
        break;
      case 'heading3':
        replaceHeading(view, 3);
        break;
      case 'paragraph':
        asParagraph(view);
        break;
      case 'unorderedList':
        prefixLine(view, '- ');
        break;
      case 'orderedList':
        prefixLine(view, '1. ');
        break;
      case 'taskList':
        prefixLine(view, '- [ ] ');
        break;
      case 'quote':
        prefixLine(view, '> ');
        break;
      case 'codeInline':
        wrapSelection(view, '`');
        break;
      case 'codeBlock':
        insertCodeBlock(view);
        break;
      case 'link':
        insertLink(view);
        break;
    }
    view.focus();
  }

  let currentTheme: 'light' | 'dark' = 'light';
  const unsubTheme = theme.subscribe((t) => {
    currentTheme = t;
    if (view) recreateExtensions(t === 'dark');
  });

  let lastExternalContent = '';
  const unsubContent = activeContent.subscribe((c) => {
    if (!view) return;
    const current = view.state.doc.toString();
    if (c !== current && c !== lastExternalContent) {
      lastExternalContent = c;
      view.dispatch({
        changes: { from: 0, to: view.state.doc.length, insert: c },
      });
    }
  });

  let lastFormatActionId = 0;
  const unsubFormatAction = formatAction.subscribe((action) => {
    if (action.id === 0 || action.id === lastFormatActionId) return;
    lastFormatActionId = action.id;
    applyFormat(action.kind);
  });

  function recreateExtensions(dark: boolean): void {
    if (!view) return;
    view.dispatch({
      effects: [
        themeCompartment.reconfigure(buildTheme(dark)),
        highlightCompartment.reconfigure(
          syntaxHighlighting(dark ? darkHighlightStyle : lightHighlightStyle, { fallback: true })
        ),
      ],
    });
  }

  onMount(() => {
    const state = EditorState.create({
      doc: $activeContent,
      extensions: createExtensions(currentTheme === 'dark'),
    });
    view = new EditorView({ state, parent: container });
    lastExternalContent = $activeContent;
  });

  onDestroy(() => {
    unsubTheme();
    unsubContent();
    unsubFormatAction();
    view?.destroy();
  });
</script>

<div class="editor-wrapper" bind:this={container}></div>

<style>
  .editor-wrapper {
    flex: 1;
    height: 100%;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  :global(.editor-wrapper .cm-editor) {
    height: 100%;
    width: 100%;
  }

  :global(.editor-wrapper .cm-scroller) {
    overflow: auto;
  }

  :global(.editor-wrapper .cm-editor.cm-focused) {
    outline: none;
  }
</style>
