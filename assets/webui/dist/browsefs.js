import {$, $$, removeAllChildNodes} from './domhelper.js';
import {rpc, downloadFile} from './api.js';
import {formatBlobSize, formatTimestamp} from './format.js';

const kCursorClass = 'browsefs__entry--cursor';

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

  setCursorIndexBounded(c) { 
    if (c < 0) {
      c = 0;
    } else if (this.numEntries_ <= c) {
      c = this.numEntries_ - 1;
    }

    this.cursorIndex = c;
  }

  get cursorRow() {
    return this.listTBody_.querySelector(`tr:nth-child(${this.cursorIndex_ + 1})`);
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
    tr.classList.add('browsefs__entry');
    this.listTBody_.appendChild(tr);

    tr.toggleSelection = () => {
      tr.classList.toggle('browsefs__entry--selected');
    };

    for (let colName of colNames) {
      var cell = document.createElement('td');
      cell.classList.add('browsefs__cell');
      cell.classList.add(`browsefs__cell--${colName}`);

      cell.textContent = formatVal(colName, row[colName]);
      if (colName === 'name') {
        const actionDef = actionDefMap[row.type || 'FILE'] || {labels: []};
        tr.triggerAction = () => { actionDef.action(this, row, host); };

        for (let l of actionDef.labels) {
          const span = document.createElement('span');
          span.classList.add('browsefs__action');
          span.textContent = l;
          cell.appendChild(span);
        }
        const action = actionDef.action;
        if (action !== undefined) {
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
        }
      }

      tr.appendChild(cell);
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
        const result = await lsPath(host, path);
        const entries = result['listing'][0]['entry'];

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

      var tr = document.createElement('tr');
      tr.classList.add('browsefs__entry');
      tr.classList.add('browsefs__error');
      this.listTBody_.appendChild(tr);

      tr.textContent = e.message;
    }
    this.updateCursor();
  }

  updateCursor() {
    for (var tr of this.listTBody_.querySelectorAll("tr")) {
      tr.classList.remove(kCursorClass);
    }
    
    if (this.cursorIndex_ < 0)
      return;

    let trC = this.cursorRow;
    if (trC)
      trC.classList.add(kCursorClass);
  }
}
window.customElements.define("browse-fs", BrowseFS);

var staticHostList = null;
const getHostList = async () => {
  if (staticHostList !== null) {
    return staticHostList;
  }

  const result = await rpc('api/v1/fe/hosts');
  return result['host'];
};

const reOtaruPath = /^\/\/([\w\[\]]+)(\/.*)$/
const parseOtaruPath = (opath) => {
  const m = opath.match(reOtaruPath);
  if (!m) {
    throw new Error(`Invalid otaru path: ${opath}`)
  }
  const host = m[1];
  const path = m[2];
  return {host, path};
}

const lsPath = async (host, path) => {
  const ep = (host === '[noproxy]') ? 'api/v1/filesystem/ls' :
    `proxy/${host}/api/v1/filesystem/ls`;

  return await rpc(ep, {args: {path: path}});
};

const formatVal = (type, val)=> {
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
