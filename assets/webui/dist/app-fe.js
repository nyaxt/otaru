import {contentSection, updateContentIfNeeded} from './nav.js';
import {fillRemoteContent} from './api.js';
import {$} from './domhelper.js';
import './browsefs.js';
import {preview} from './preview.js';
import {infobar} from './infobar.js';

const leftfs = $("#leftfs");
const rightfs = $("#rightfs");

leftfs.path = '//[local]/';

leftfs.counterpart = rightfs;
rightfs.counterpart = leftfs;

contentSection('browsefs').addEventListener('shown', e => {
  if (e.browsefsPath !== undefined)
    rightfs.path = e.browsefsPath;
  rightfs.triggerUpdate();
});
contentSection('browsefs').addEventListener('hidden', () => {
  return rightfs.clear();
});
rightfs.addEventListener('pathChanged', e => {
  updateContentIfNeeded({currBrowsefsPath: e.newPath});
});

const splitbar = $('.splitbar');
const noophandler = () => { return false; };
splitbar.addEventListener('mousedown', md => {
  const pn = splitbar.parentNode;
  const offX = pn.offsetLeft;
  const offW = pn.offsetWidth;
  const leftpane = pn.querySelector('.split--leftpane');
  const rightpane = pn.querySelector('.split--rightpane');

  const mmhandler = mm => {
    const l = (event.pageX - offX) / offW;
    leftpane.style.width = `${l * 100}%`;
    rightpane.style.width =`${(1.0 - l) * 100}%`;
  };
  const muhandler = mu => {
    pn.removeEventListener('mousemove', mmhandler);
    pn.removeEventListener('mouseup', muhandler);
    pn.removeEventListener('selectstart', noophandler);
    pn.classList.remove('drag_parent');
  };
  pn.addEventListener('mousemove', mmhandler);
  pn.addEventListener('mouseup', muhandler);
  pn.addEventListener('selectstart', noophandler);
  pn.classList.add('drag_parent');
});

window.addEventListener('DOMContentLoaded', () => {
  let focusfs = leftfs;
  focusfs.cursorIndex = 0;
  focusfs.hasFocus = true;

  document.addEventListener('keydown', e => {
    if (e.key === 'PageDown') {
      focusfs.cursorIndex = focusfs.cursorIndex + focusfs.numVisibleRows;
    } else if (e.key === 'PageUp') {
      focusfs.cursorIndex = Math.max(focusfs.cursorIndex - focusfs.numVisibleRows, 0);
    } else {
      console.log(`keydown ${e.key}`);
    }
  });
  document.addEventListener('keypress', e => {
    if (e.target instanceof HTMLInputElement)
      return true;

    if (preview.isOpen) {
      if (e.key === 'j') {
        ++ preview.cursorIndex;
      } else if (e.key === 'k') {
        preview.cursorIndex = Math.max(preview.cursorIndex - 1, 0);
      } else if (e.key === 'Enter') {
        preview.toggleView();
      }
      return false;
    }

    if (e.key === 'l') {
      rightfs.cursorIndex = leftfs.cursorIndex;
      leftfs.clearCursor();
      focusfs = rightfs;
      leftfs.hasFocus = false;
      rightfs.hasFocus = true;
    } else if (e.key === 'h') {
      leftfs.cursorIndex = rightfs.cursorIndex;
      rightfs.clearCursor();
      focusfs = leftfs;
      rightfs.hasFocus = false;
      leftfs.hasFocus = true;
    } else {
      console.log(`keypress ${e.key}`);
    }
  });
  document.addEventListener('keyup', e => {
    if (e.key === 'Escape') {
      if (preview.isOpen)
        preview.close();
    } else {
      console.log(`keyup ${e.key}`);
    }
  });

  infobar.showMessage("hoge");  
});
