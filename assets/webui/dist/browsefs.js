import {contentSection, isSectionSelected, getBrowsefsPath, setBrowsefsPath} from './nav.js';
import {$, $$, removeAllChildNodes} from './domhelper.js';
import {rpc, downloadFile} from './api.js';
import {formatBlobSize, formatTimestamp} from './format.js';

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
  'DIR': {
    labels: ['→'],
    action: entry => {
      const curr = getBrowsefsPath();
      const next = curr.replace(/\/?$/, '/') + entry.name;
      setBrowsefsPath(next);
    },
  },
  'FILE': {
    labels: ['DL'],
    action: (entry, host) => {
      downloadFile(host, entry['id'], entry['name'])
    },
  },
};
const pathInput = $('.browsefs__path');
const listTbody = $('.browsefs__list').lastChild;
const upload = $('.browsefs__upload');

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

const triggerUpdate = async () => {
  if (!isSectionSelected('browsefs'))
    return;

  const opath = getBrowsefsPath();
  if (pathInput.value !== opath) {
    pathInput.value = opath;
  }

  removeAllChildNodes(listTbody);
  try {
    if (opath === "//") {
      const hosts = await getHostList();

      for (let host of hosts) {
        var tr = document.createElement('tr');
        tr.classList.add('browsefs__entry');
        listTbody.appendChild(tr);

        var cell = document.createElement('td');
        cell.classList.add('browsefs__cell');
        cell.classList.add('browsefs__cell--name');
        cell.textContent = host;

        const span = document.createElement('span');
        span.classList.add('browsefs__action');
        span.textContent = '→';
        cell.appendChild(span);

        tr.addEventListener('click', (ev) => {
          const next = `//${host}/`;
          setBrowsefsPath(next);
        });

        tr.appendChild(cell);
      }
    } else {
      const {host, path} = parseOtaruPath(opath);
      const result = await lsPath(host, path);
      const entries = result['listing'][0]['entry'];

      const sortSel = $('.browsefs__sort').value;
      const sortFunc = sortFuncMap[sortSel];

      if (entries === undefined) {
        listTbody.classList.add('.browsefs__list--empty');
      } else {
        listTbody.classList.remove('.browsefs__list--empty');

        const rows = entries.sort(sortFunc);
        for (let row of rows) {
          var tr = document.createElement('tr');
          tr.classList.add('browsefs__entry');
          listTbody.appendChild(tr);

          for (let colName of colNames) {
            var cell = document.createElement('td');
            cell.classList.add('browsefs__cell');
            cell.classList.add(`browsefs__cell--${colName}`);

            let val = row[colName];
            if (colName === 'type') {
              if (val === undefined)
                val = 'FILE';

              val = val.toLowerCase()[0];
            } else if (colName === 'perm_mode') {
              val = val.toString(8);
            } else if (colName === 'uid') {
              if (val === undefined)
                val = 0;
              val = 'u'+val;
            } else if (colName === 'gid') {
              if (val === undefined)
                val = 0;
              val = 'g'+val;
            } else if (colName === 'size') {
              val = formatBlobSize(val);
            } else if (colName.match(reTime)) {
              val = formatTimestamp(new Date(val*1000));
            }
            if (val === undefined)
              val = '-';

            cell.textContent = val;
            if (colName === 'name') {
              const actionDef = actionDefMap[row.type || 'FILE'] || {labels: []};

              for (let l of actionDef.labels) {
                const span = document.createElement('span');
                span.classList.add('browsefs__action');
                span.textContent = l;
                cell.appendChild(span);
              }
              const action = actionDef.action;
              if (action !== undefined) {
                tr.addEventListener('click', (ev) => {
                  actionDef.action(row, host);
                });
              }
            }

            tr.appendChild(cell);
          }
        }
      }
    }
  } catch (e) {
    listTbody.classList.remove('.browsefs__list--empty');

    var tr = document.createElement('tr');
    tr.classList.add('browsefs__entry');
    tr.classList.add('browsefs__error');
    listTbody.appendChild(tr);

    tr.textContent = e.message;
  }
}

contentSection('browsefs').addEventListener('shown', triggerUpdate);
$('.browsefs__sort').addEventListener('change', triggerUpdate);
$('.browsefs__parentdir').addEventListener('click', ev => {
  const curr = getBrowsefsPath();
  const next = curr.replace(/[^\/]+\/?$/, '');
  if (next !== curr)
    setBrowsefsPath(next);

  ev.preventDefault();
});
pathInput.addEventListener('change', () => {
  const path = getBrowsefsPath();
  if (pathInput.value !== path) {
    setBrowsefsPath(pathInput.value);
  }
});
upload.addEventListener('change', async () => {
  const files = upload.files;
  console.log('------') ;
  for (let file of files) {
    console.log(`name: ${file.name} size: ${file.size} type: ${file.type}`) ;
    // FIXME sanitize file.name
    const cfresp = await rpc('api/v1/filesystem/file', {
      method: 'POST',
      body: {
        dir_id: 0,
        name: `${getBrowsefsPath()}/${file.name}`,
        uid: 0, gid: 0, perm_mode: 0o644, modified_time: 0
      }});
    const id = cfresp.id;
    console.dir(cfresp);
    const uplresp = await rpc(`file/${id}`, {method: 'PUT', args:{ offset: 0 }, rawBody: file});
  }
});
contentSection('browsefs').addEventListener('hidden', () => {
  removeAllChildNodes(listTbody);
});

export {staticHostList};
