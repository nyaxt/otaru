import {$, $$, removeAllChildNodes} from './domhelper.js';
import {rpc, downloadFile, parseOtaruPath, fsLs, fsMv} from './api.js';
import {formatBlobSize, formatTimestamp} from './format.js';
import {findLongestCommonSubStr} from './commonsubstr.js';

const kCursorClass = 'content__entry--cursor';
const kMatchClass = 'browsefs__entry--match';
const kSelectedClass = 'browsefs__entry--selected';

const kFilterUpdateDelayMs = 500;
const kRowHeight = 30;

const reValidFileName = /^[^\/]+$/;

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
  <div class="section__header browsefs__header browsefs__header--rename">
    <label class="browsefs__label" for="browsefs-rename">Rename: </label>
    <input class="browsefs__text browsefs__rename" type="text" id="browsefs-rename" tabindex="1">
  </div>
  <div class="browsefs__scroll">
    <table class="content__table browsefs__list"><tbody></tbody></table>
  </div>`;

class BrowseFS extends HTMLElement {
  constructor() {
    super();

    this.inflightUpdate_ = false;
    this.path_ = '//';
    this.cursorIndex_ = -1;
    this.numEntries_ = 0;
    this.query_ = null;
    this.renameBefore_ = null;
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
    this.renameInput_ = this.querySelector('.browsefs__rename');
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

    this.renameInput_.addEventListener('keydown', kd => {
      if (kd.key === 'Tab') {
        kd.preventDefault();
        this.renameInput_.blur();
        return true;
      }
      return false;
    });
    this.renameInput_.addEventListener('keypress', (e) => {
      if (e.key === 'Enter') {
        this.executeRename();
        return false;
      }

      return false;
    });
    this.renameInput_.addEventListener('keyup', (e) => {
      if (e.key === 'Escape') {
        console.log("rename escape");
        this.closeRenameDialog();
        return false;
      }
      return true;
    });

    window.addEventListener("DOMContentLoaded", () => {
      this.triggerUpdate();
    });
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
      actionDef.action(this, data, host);
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
      let cell = document.createElement('td');
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
        let i = name.indexOf(highlight);
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
        this.numEntries_ = hosts.length;
      } else {
        const {host, path} = parseOtaruPath(opath);
        const result = await fsLs(host, path);
        const entries = result['entry'] || result['listing'][0]['entry'];

        const sortSel = $('.browsefs__sort').value;
        const sortFunc = sortFuncMap[sortSel];

        if (entries === undefined) {
          this.listTBody_.classList.add('.browsefs__list--empty');
          this.numEntries_ = 0;
        } else {
          this.listTBody_.classList.remove('.browsefs__list--empty');

          const rows = entries.sort(sortFunc);
          for (let row of rows) {
            this.appendRow_(row, host);
          }
          this.numEntries_ = entries.length;
        }
      }
    } catch (e) {
      this.listTBody_.classList.remove('.browsefs__list--empty');

      let tr = document.createElement('tr');
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
    let visibleRows = this.getVisibleRows_();

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

    let cr = visibleRows[this.cursorIndex_];
    this.cursorRow_ = cr;
    if (cr) {
      cr.classList.add(kCursorClass);

      let crRect = cr.getBoundingClientRect();
      let divRect = this.scrollDiv_.getBoundingClientRect();

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

    let query = this.query_;
    if (query === '')
      query = '.';
    let filterRe = new RegExp(query);

    for (let tr of this.listTBody_.querySelectorAll("tr")) {
      let match = tr.data.name.match(filterRe);
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

  openRenameDialog() {
    const selectedRows = this.getSelectedRows_();
    if (selectedRows.length == 0) {
      if (this.cursorRow === null) {
        console.log("tried to open rename dialog, but no row.");
        return;
      }
      this.cursorRow.toggleSelection();
      selectedRows = this.getSelectedRows_();
    } else if (selectedRows.length == 1) {
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

    this.classList.add('modal');
    this.renameInput_.value = this.renameBefore_;
    this.renameInput_.disabled = false;
    window.setTimeout(() => {
      this.renameInput_.focus();
    }, 10);
  }

  async executeRename() {
    this.renameInput_.disabled = true;
    const renameAfter = this.renameInput_.value;

    const selectedRows = this.getSelectedRows_();
    for (let r of selectedRows) {
      const newFileName = r.data.name.replace(this.renameBefore_, renameAfter);
      if (!newFileName.match(reValidFileName)) {
        throw new Error(`New filename "${newFileName}" is not valid.`);
      }
      let pathSrc = this.path + selectedRows[0].data.name;
      let pathDest = this.path + newFileName;

      const result = await fsMv(pathSrc, pathDest);
      r.data.name = newFileName;
    }

    if (selectedRows.length === 1) {
      this.cursorRow.toggleSelection();
    }
    this.closeRenameDialog();
  }

  closeRenameDialog() {
    this.updateNameHighlight_(null);

    this.renameInput_.blur();
    this.classList.remove('modal');
  }

  updateNameHighlight_(highlight) {
    for (let r of this.getSelectedRows_()) {
      const tdName = r.querySelector('td.browsefs__cell--name');
      removeAllChildNodes(tdName);
      this.populateNameCell_(tdName, r.data, highlight);
    }
  }
}
window.customElements.define("browse-fs", BrowseFS);

let staticHostList = null;
const getHostList = async () => {
  if (staticHostList !== null) {
    return staticHostList;
  }

  const result = await rpc('api/v1/fe/hosts');
  return result['host'];
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
