import {contentSection, updateContentIfNeeded} from './nav.js';
import {fillRemoteContent} from './api.js';
import {$} from './domhelper.js';
import './browsefs.js';

const rightfs = $("browse-fs");

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
