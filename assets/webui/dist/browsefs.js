import {$, $$, removeAllChildNodes} from './domhelper.js';
import {rpc, downloadFile, parseOtaruPath, fsLs, fsMv, fsMkdir} from './api.js';
import {formatBlobSize, formatTimestamp} from './format.js';
import {findLongestCommonSubStr} from './commonsubstr.js';
import {preview} from './preview.js';

const kDialogCancelled = Symbol('modal dialog cancelled.');

const kFocusClass = 'hasfocus';
const kModalClass = 'modal';
const kPromptActiveClass = 'promptactive';
const kConfirmActiveClass = 'confirmactive';
const kCursorClass = 'content__entry--cursor';
const kMatchClass = 'browsefs__entry--match';
const kSelectedClass = 'browsefs__entry--selected';

const kFilterUpdateDelayMs = 500;
const kRowHeight = 30;

const reValidFileName = /^[^\/]+$/;
const reFileNameBase = /^[^\d\.]+/;

const colNames = ['type', 'name', 'size', 'uid', 'gid', 'perm_mode', 'modified_time'];
const reTime = /_time$/;
const sortFuncMap = {
  'name': (a, b) => {
    if (a['type'] != b['type']) {
      const atype = a['type'] || "FILE";
      const btype = b['type'] || "FILE";
      return atype.charCodeAt() - btype.charCodeAt();
    }
    return a['name'].localeCompare(b['name']);
  },
  'time_asc': (a, b) => a['modified_time'] - b['modified_time'],
  'time_desc': (a, b) => b['modified_time'] - a['modified_time'],
};
const actionDefMap = {
  'HOST': {
    labels: ['→'],
    action: (browsefs, entry) => {
      const next = `//${entry.name}/`;
      browsefs.path = next;
    },
  },
  'DIR': {
    labels: ['→'],
    action: (browsefs, entry) => {
      const curr = browsefs.path;
      const next = curr.replace(/\/?$/, '/') + entry.name;
      browsefs.path = next;
    },
  },
  'FILE': {
    labels: ['DL'],
    action: (browsefs, entry, host) => {
      downloadFile(host, entry['id'], entry['name'])
    },
  },
};
const getActionDef = (data) => {
  return actionDefMap[data.type || 'FILE'] || {labels: []};
};

const innerHTMLSource =
  `<div class="section__header browsefs__header">
    <a class="button browsefs__parentdir" href="#">↑</a>
    <label class="browsefs__label" for="browsefs-path">Path: </label>
    <input class="browsefs__path" type="text" id="browsefs-path">
    <select class="browsefs__sort">
      <option value="name" selected>Name</option>
      <option value="time_asc">Time ↓</option>
      <option value="time_desc">Time ↑</option>
    </select>
    <input class="browsefs__upload" type="file" id="browsefs-upload" multiple>
    <label class="button browsefs__label--upload" for="browsefs-upload">Upload</label>
  </div>
  <div class="section__header browsefs__header browsefs__header--query">
    <label class="browsefs__label" for="browsefs-query">Query: </label>
    <input class="browsefs__text browsefs__query" type="text" id="browsefs-query" tabindex="1">
  </div>
  <div class="section__header browsefs__header browsefs__confirm">
    <div class="browsefs__confirm--title"></div>
    <div class="browsefs__confirm--detail"></div>
    <div class="browsefs__confirm--btnscont">
      <a class="button browsefs__confirm--ok" tabindex="2" href="#">OK</a>
      <a class="button browsefs__confirm--cancel" tabindex="3" href="#">Cancel</a>
    </div>
  </div>
  <div class="section__header browsefs__header browsefs__header--prompt">
    <label class="browsefs__label browsefs__promptlabel" for="browsefs-prompt">prompt: </label>
    <input class="browsefs__text browsefs__prompt" type="text" id="browsefs-prompt" tabindex="4">
  </div>
  <div class="browsefs__scroll">
    <table class="content__table browsefs__list"><tbody></tbody></table>
  </div>`;

class BrowseFS extends HTMLElement {
  constructor() {
    super();

    this.inflightUpdate_ = false;
    this.path_ = '//';
    this.cursorRow_ = null;
    this.cursorIndex_ = -1;
    this.query_ = null;
    this.renameBefore_ = null;
  }

  get hasFocus() {
    return this.classList.contains(kFocusClass);
  }

  set hasFocus(val) {
    if (val)
      this.classList.add(kFocusClass);
    else
      this.classList.remove(kFocusClass);
  }

  get hasModalDialog() {
    return this.classList.contains(kModalClass);
  }

  get path() {
    return this.path_;
  }

  set path(val) {
    if (this.path_ == val)
      return;

    const e = new Event('pathChanged');
    e.oldPath = this.path_;
    e.newPath = val;

    this.path_ = val;
    this.query = null;
    if (this.cursorIndex_ >= 0) {
      this.cursorIndex_ = 0;
    }

    this.dispatchEvent(e);
    this.triggerUpdate();
  }

  get cursorIndex() {
    return this.cursorIndex_;
  }

  set cursorIndex(val) {
    if (this.cursorIndex_ == val)
      return;

    this.cursorIndex_ = val;
    this.updateCursor();
  }

  clearCursor() {
    this.cursorIndex = -1;
  }

  get cursorRow() {
    return this.cursorRow_;
  }

  get numVisibleRows() {
    let divRect = this.scrollDiv_.getBoundingClientRect();
    return Math.floor(divRect.height / kRowHeight);
  }

  get query() {
    return query_;
  }

  set query(val) {
    this.query_ = val;
    window.setTimeout(() => {
      if (this.query_ === null) {
        this.classList.remove('filtered');
        return;
      }

      this.classList.add('filtered');
      this.queryInput_.value = val;
      this.queryInput_.focus();

      this.restartFilterTimer_();
    }, 0);
  }

  navigateParent() {
    if (this.cursorIndex_ > 0)
      this.cursorIndex_ = 0;

    const curr = this.path;
    const next = curr.replace(/[^\/]+\/?$/, '');
    if (next !== curr)
      this.path = next;
  }

  connectedCallback() {
    this.innerHTML = innerHTMLSource;

    this.parentDirBtn_ = this.querySelector('.browsefs__parentdir');
    this.pathInput_ = this.querySelector('.browsefs__path');
    this.sortSelect_ = this.querySelector('.browsefs__sort');
    this.listTBody_ = this.querySelector('.browsefs__list').lastChild;
    this.upload_ = this.querySelector('.browsefs__upload');
    this.queryInput_ = this.querySelector('.browsefs__query');
    this.promptLabel_ = this.querySelector('.browsefs__promptlabel');
    this.promptInput_ = this.querySelector('.browsefs__prompt');
    this.confirmTitle_ = this.querySelector('.browsefs__confirm--title');
    this.confirmDetail_ = this.querySelector('.browsefs__confirm--detail');
    this.confirmOk_ = this.querySelector('.browsefs__confirm--ok');
    this.confirmCancel_ = this.querySelector('.browsefs__confirm--cancel');
    this.scrollDiv_ = this.querySelector('.browsefs__scroll');

    this.parentDirBtn_.addEventListener('click', ev => {
      this.navigateParent();

      ev.preventDefault();
    });
    this.pathInput_.addEventListener('change', () => {
      const path = this.path;
      if (this.pathInput_.value !== path) {
        this.path = this.pathInput_.value;
      }
    });
    this.sortSelect_.addEventListener('change', () => {
      this.triggerUpdate();
    });
    this.upload_.addEventListener('change', async () => {
      const files = this.upload_.files;
      console.log('------') ;
      for (let file of files) {
        console.log(`name: ${file.name} size: ${file.size} type: ${file.type}`) ;
        // FIXME sanitize file.name
        const cfresp = await rpc('api/v1/filesystem/file', {
          method: 'POST',
          body: {
            dir_id: 0,
            name: `${this.path}/${file.name}`,
            uid: 0, gid: 0, perm_mode: 0o644, modified_time: 0
          }});
        const id = cfresp.id;
        console.dir(cfresp);
        const uplresp = await rpc(`file/${id}`, {method: 'PUT', args:{ offset: 0 }, rawBody: file});
      }
    });
    this.queryInput_.addEventListener('keydown', kd => {
      if (kd.key === 'Tab') {
        kd.preventDefault();
        this.queryInput_.blur();
        return true;
      }
      return false;
    });
    const onquerykeypressup = (e) => {
      if (e.type === 'keypress' && e.key === 'Enter') {
        let cr = this.cursorRow;
        if (cr)
          cr.triggerAction();

        return true;
      }
      if (e.key === 'Escape') {
        this.query = null;
        return true;
      }

      this.query_ = this.queryInput_.value;
      this.restartFilterTimer_();
      return false;
    };
    this.queryInput_.addEventListener('keypress', onquerykeypressup);
    this.queryInput_.addEventListener('keyup', onquerykeypressup);

    this.promptInput_.addEventListener('keydown', kd => {
      if (kd.key === 'Tab') {
        kd.preventDefault();
        this.onExitDialog_(false);
        return false;
      }
      return false;
    });
    this.promptInput_.addEventListener('keypress', (e) => {
      if (e.key === 'Enter') {
        this.onExitDialog_(true);
        return false;
      }

      return false;
    });
    this.promptInput_.addEventListener('keyup', (e) => {
      if (e.key === 'Escape') {
        this.onExitDialog_(false);
        return false;
      }
      return true;
    });

    this.confirmOk_.addEventListener('click', ev => {
      this.onExitDialog_(true);
      ev.preventDefault();
    });
    this.confirmCancel_.addEventListener('click', ev => {
      this.onExitDialog_(false);
      ev.preventDefault();
    });

    window.addEventListener("DOMContentLoaded", () => {
      this.triggerUpdate();
    });
    window.addEventListener('keypress', this.onKeyPress_.bind(this));
    window.addEventListener('keyup', this.onKeyUp_.bind(this));
  }

  async onKeyPress_(e) {
    if (preview.isOpen || !this.hasFocus)
      return;
    if (this.hasModalDialog) {
      if (e.key === 'Enter') {
        if (this.onExitDialog_) this.onExitDialog_(true);
      }
      return;
    }

    if (e.key === 'j') {
      ++ this.cursorIndex;
    } else if (e.key === 'k') {
      this.cursorIndex = Math.max(this.cursorIndex - 1, 0);
    } else if (e.key === 'r') {
      this.openRenamePrompt();
    } else if (e.key === 'd') {
      this.openMkdirPrompt();
    } else if (e.key === 'p') {
      let cr = this.cursorRow;
      if (cr)
        preview.open(cr.opath);
    } else if (e.key === 'x') {
      let cr = this.cursorRow;
      if (cr)
        cr.toggleSelection();
    } else if (e.key === ' ') {
      let cr = this.cursorRow;
      if (cr)
        cr.toggleSelection();

      this.cursorIndex = this.cursorIndex + 1;
    } else if (e.key === 'Delete') {
      this.deleteSelection();
    } else if (e.key === 'Enter') {
      let cr = this.cursorRow;
      if (cr)
        cr.triggerAction();
    } else if (e.key === 'u') {
      this.navigateParent();
    } else if (e.key === '?') {
      this.query = '';
    }
  }

  async onKeyUp_(e) {
    if (e.key === 'Escape') {
      if (this.onExitDialog_) this.onExitDialog_(false);
    }
  }

  clear() {
    removeAllChildNodes(this.listTBody_);
  }

  async triggerUpdate() {
    if (this.inflightUpdate_)
      return;

    this.inflightUpdate_ = true;
    for (;;) {
      const updatePath = this.path;
      await this.triggerUpdateLocked_();
      if (updatePath == this.path)
        break;
    }
    this.inflightUpdate_ = false;
  }

  appendRow_(row, host) {
    let tr = document.createElement('tr');
    tr.classList.add('content__entry');
    tr.classList.add('browsefs__entry');
    tr.data = row;
    tr.opath = `${this.path}/${row.name}`;
    this.listTBody_.appendChild(tr);

    tr.toggleSelection = () => {
      tr.classList.toggle(kSelectedClass);
    };
    const actionDef = getActionDef(row);
    tr.triggerAction = () => {
      actionDef.action(this, row, host);
    };
    let cancelClick = false;
    tr.addEventListener('click', (ev) => {
      cancelClick = false;
      window.setTimeout(() => {
        if (cancelClick)
          return;

        tr.toggleSelection();
      }, 200);
    });
    tr.addEventListener('dblclick', (ev) => {
      cancelClick = true;
      tr.triggerAction();
    });

    for (let colName of colNames) {
      const cell = document.createElement('td');
      cell.classList.add('content__cell');
      cell.classList.add('browsefs__cell');
      cell.classList.add(`browsefs__cell--${colName}`);

      if (colName === 'name') {
        this.populateNameCell_(cell, row);
      } else {
        cell.textContent = formatVal(colName, row[colName]);
      }

      tr.appendChild(cell);
    }
  }

  populateNameCell_(cell, data, highlight = null) {
    let name = data.name;
    if (highlight) {
      for (;;) {
        const i = name.indexOf(highlight);
        if (i < 0) {
          break;
        }

        const nohl = name.substr(0, i);
        cell.appendChild(document.createTextNode(nohl));

        const spanHl = document.createElement('span');
        spanHl.classList.add('browsefs__highlight');
        spanHl.textContent = highlight;
        cell.appendChild(spanHl);

        name = name.substr(i + highlight.length);
      }
    }
    cell.appendChild(document.createTextNode(name));

    for (let l of getActionDef(data).labels) {
      const span = document.createElement('span');
      span.classList.add('browsefs__action');
      span.textContent = l;
      cell.appendChild(span);
    }
  }

  async triggerUpdateLocked_() {
    const opath = this.path;
    if (this.pathInput_.value !== opath) {
      this.pathInput_.value = opath;
    }

    this.clear();
    try {
      if (opath === "//") {
        const hosts = await getHostList();

        for (let host of hosts) {
          this.appendRow_({
            type: 'HOST',
            name: host,
          }, null);
        }
      } else {
        const {host, path} = parseOtaruPath(opath);
        const entries = await fsLs(host, path);

        const sortSel = $('.browsefs__sort').value;
        const sortFunc = sortFuncMap[sortSel];

        if (entries.length === 0) {
          this.listTBody_.classList.add('.browsefs__list--empty');

          const tr = document.createElement('tr');
          tr.classList.add('content__entry');
          tr.classList.add('browsefs__entry');
          this.listTBody_.appendChild(tr);

          const cell = document.createElement('td');
          cell.classList.add('content__cell');
          cell.classList.add('browsefs__cell');
          cell.textContent = '<no entries>';
          tr.appendChild(cell);
        } else {
          this.listTBody_.classList.remove('.browsefs__list--empty');

          const rows = entries.sort(sortFunc);
          for (let row of rows) {
            this.appendRow_(row, host);
          }
        }
      }
    } catch (e) {
      this.listTBody_.classList.remove('.browsefs__list--empty');

      const tr = document.createElement('tr');
      tr.classList.add('browsefs__entry');
      tr.classList.add('browsefs__error');
      this.listTBody_.appendChild(tr);

      tr.textContent = e.message;
    }
    this.updateCursor();
  }

  getVisibleRows_(extrasel = '') {
    if (this.query_ !== null) {
      return this.listTBody_.querySelectorAll(`tr.${kMatchClass}${extrasel}`);
    }
    return this.listTBody_.querySelectorAll(`tr${extrasel}`);
  }

  updateCursor() {
    const visibleRows = this.getVisibleRows_();

    for (let tr of visibleRows) {
      tr.classList.remove(kCursorClass);
    }

    if (this.cursorIndex_ < 0)
      return;

    if (this.cursorIndex_ >= visibleRows.length) {
      this.cursorIndex_ = visibleRows.length - 1;
    } else if (this.cursorIndex < 0) {
      this.cursorRow_ = null;
      return;
    }

    const cr = visibleRows[this.cursorIndex_];
    this.cursorRow_ = cr;
    if (cr) {
      cr.classList.add(kCursorClass);

      const crRect = cr.getBoundingClientRect();
      const divRect = this.scrollDiv_.getBoundingClientRect();

      if (crRect.bottom > divRect.bottom) {
        cr.scrollIntoView({block: 'end'});
      } else if (crRect.top < divRect.top) {
        cr.scrollIntoView({block: 'start'});
      }
    }
  }

  restartFilterTimer_() {
    if (this.filterTimer_ !== undefined) {
      window.clearTimeout(this.filterTimer_);
    }
    this.filterTimer_ = window.setTimeout(() => this.onFilterTimer_(), kFilterUpdateDelayMs);
  }

  onFilterTimer_() {
    if (this.query_ === null)
      return;

    const query = this.query_;
    if (query === '')
      query = '.';
    const filterRe = new RegExp(query);

    for (let tr of this.listTBody_.querySelectorAll("tr")) {
      const match = tr.data.name.match(filterRe);
      if (!match) {
        tr.classList.remove(kMatchClass);
        continue;
      }
      tr.classList.add(kMatchClass);
    }
    this.cursorIndex_ = 0;
    this.updateCursor();
  }

  getSelectedRows_() {
    return this.getVisibleRows_(`.${kSelectedClass}`);
  }

  async openRenamePrompt() {
    let selectedRows = this.getSelectedRows_();
    if (selectedRows.length == 0) {
      if (!this.cursorRow) {
        console.log("tried to open rename dialog, but no row selected.");
        return;
      }
      this.cursorRow.toggleSelection();
      selectedRows = this.getSelectedRows_();
    }

    if (selectedRows.length == 1) {
      this.renameBefore_ = selectedRows[0].data.name;
    } else {
      const names = Array.from(selectedRows).map(r => r.data.name);
      const lcss = findLongestCommonSubStr(names);
      if (lcss.length == 0) {
        alert("no common substr found.");
        return;
      }
      this.renameBefore_ = lcss;
      this.updateNameHighlight_(lcss);
    }

    try {
      const renameAfter = await this.openPrompt_("Rename: ", this.renameBefore_, reFileNameBase);

      for (let r of selectedRows) {
        const oldFileName = r.data.name;
        const newFileName = oldFileName.replace(this.renameBefore_, renameAfter);
        if (!newFileName.match(reValidFileName)) {
          throw new Error(`New filename "${newFileName}" is not valid.`);
        }
        const pathSrc = this.path + oldFileName;
        const pathDest = this.path + newFileName;

        const result = await fsMv(pathSrc, pathDest);
        r.data.name = newFileName;
      }
    } catch(e) {
      if (e === kDialogCancelled) {
        console.log(`rename cancelled.`);
      } else {
        console.log(`rename failed: ${e}`);
      }
    }
    this.updateNameHighlight_(null);
    if (selectedRows.length === 1) {
      this.cursorRow.toggleSelection();
    }
    this.closePrompt_();
  }

  updateNameHighlight_(highlight) {
    for (let r of this.getSelectedRows_()) {
      const tdName = r.querySelector('td.browsefs__cell--name');
      removeAllChildNodes(tdName);
      this.populateNameCell_(tdName, r.data, highlight);
    }
  }

  async openMkdirPrompt() {
    try {
      const dirname = await this.openPrompt_("Mkdir: ", "");

      const result = await fsMkdir(this.path + dirname);
      this.triggerUpdate();
    } catch(e) {
      if (e === kDialogCancelled) {
        console.log(`mkdir cancelled.`);
      } else {
        console.log(`mkdir failed: ${e}`);
      }
    }
    this.closePrompt_();
  }

  openPrompt_(labelText, initValue, selRe = null) {
    this.promptLabel_.textContent = labelText;
    this.promptInput_.value = initValue;
    this.promptInput_.disabled = false;
    this.classList.add(kModalClass);
    this.classList.add(kPromptActiveClass);
    window.requestAnimationFrame(() => {
      this.promptInput_.focus();
      if (selRe) {
        const result = reFileNameBase.exec(this.promptInput_.value);
        if (result) {
          this.promptInput_.setSelectionRange(result.index, result[0].length);
        }
      }
    });

    return new Promise((resolve, reject) => {
      this.onExitDialog_ = (success) => {
        this.onExitDialog_ = null;
        this.promptInput_.disabled = true;

        if (!success) {
          this.closePrompt_();
          reject(kDialogCancelled);
          return;
        }

        resolve(this.promptInput_.value);
      };
    });
  }

  closePrompt_() {
    this.promptLabel_.textContent = "Prompt: ";
    this.promptInput_.blur();
    this.classList.remove(kModalClass);
    this.classList.remove(kPromptActiveClass);
  }

  openConfirm_(title, detail) {
    this.confirmTitle_.textContent = title;

    const lines = detail.split(/\r?\n/g);
    let firstLine = true;
    for (let l of lines) {
      if (firstLine) {
        firstLine = false;
      } else {
        const br = document.createElement('br');
        this.confirmDetail_.appendChild(br);
      }
      const text = document.createTextNode(l);
      this.confirmDetail_.appendChild(text);
    }
    this.classList.add(kModalClass);
    this.classList.add(kConfirmActiveClass);

    return new Promise((resolve, reject) => {
      this.onExitDialog_ = (success) => {
        this.onExitDialog_ = null;
        if (!success) {
          this.closeConfirm_();
          reject(kDialogCancelled);
          return;
        }

        resolve(true);
      };
    });
  }

  closeConfirm_() {
    this.classList.remove(kModalClass);
    this.classList.remove(kConfirmActiveClass);
    removeAllChildNodes(this.confirmTitle_);
    removeAllChildNodes(this.confirmDetail_);
  }

  async deleteSelection() {
    let selectedRows = this.getSelectedRows_();
    if (selectedRows.length == 0) {
      if (!this.cursorRow) {
        console.log("No row selected for deletion.");
        return;
      }
      this.cursorRow.toggleSelection();
      selectedRows = this.getSelectedRows_();
    }

    let details = '';
    for (let r of selectedRows) {
      details += `rm: "${r.data.name}"\n`;
    }
    await this.openConfirm_(`Delete: ${selectedRows.length} item(s)`, details);
    this.closeConfirm_();
  }
}
window.customElements.define("browse-fs", BrowseFS);

let staticHostList = null;
const getHostList = async () => {
  if (staticHostList === null) {
    const result = await rpc('api/v1/fe/hosts');
    staticHostList = result['host'];
  }

  return staticHostList;
};

const formatVal = (type, val) => {
  if (type === 'type') {
    if (val === undefined)
      val = 'FILE';

    return val.toLowerCase()[0];
  } else if (type === 'perm_mode') {
    if (val === undefined)
      val = '000';

    return val.toString(8);
  } else if (type === 'uid') {
    if (val === undefined)
      val = 0;
    return 'u'+val;
  } else if (type === 'gid') {
    if (val === undefined)
      val = 0;
    return 'g'+val;
  } else if (type === 'size') {
    return formatBlobSize(val);
  } else if (type.match(reTime)) {
    if (val === undefined)
      return '--/--/--';

    return formatTimestamp(new Date(val*1000));
  }

  if (val === undefined)
    return '-';
  return val;
};

export {staticHostList};
