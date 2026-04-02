import {initComboMarkdownEditor} from '../comp/ComboMarkdownEditor.js';

export function initHackathonEditor() {
  const editors = document.querySelectorAll('.page-content .combo-markdown-editor');
  if (!editors.length) return;

  for (const editor of editors) {
    initComboMarkdownEditor(editor);
  }
}
