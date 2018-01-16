import {contentSection, isSectionSelected, getBrowsefsPath, setBrowsefsPath} from './nav.js';
import {$, $$, removeAllChildNodes} from './domhelper.js';
import {rpc, downloadFile} from './api.js';
import {formatBlobSize, formatTimestamp} from './format.js';

const colNames = ['type', 'name', 'size', 'uid', 'gid', 'perm_mode', 'modified_time'];
const reTime = /_time$/;
const sortFuncMap = {
  'name': (a, b) => {
    if (a['type'] != b['type']) {
      return a['type'].charCodeAt() - b['type'].charCodeAt();
    }
    return a['name'].localeCompare(b['name']);
  },
  'time_asc': (a, b) => a['modified_time'] - b['modified_time'],
  'time_desc': (a, b) => b['modified_time'] - a['modified_time'],
};
const actionDefMap = {
  'dir': {
    labels: ['→'],
    action: entry => {
      const curr = getBrowsefsPath();
      const next = curr.replace(/\/?$/, '/') + entry.name;
      setBrowsefsPath(next);
    },
  },
  'file': {
    labels: ['DL'],
    action: entry => {
      downloadFile(entry['id'], entry['name'])
    },
  },
};
const pathInput = $('.browsefs__path');
const listTbody = $('.browsefs__list').lastChild;
const upload = $('.browsefs__upload');

const triggerUpdate = async () => {
  if (!isSectionSelected('browsefs'))
    return;

  const path = getBrowsefsPath();
  if (pathInput.value !== path) {
    pathInput.value = path;
  }

  try {
    const result = await rpc('api/v1/filesystem/ls', {args: {path: path}});

    removeAllChildNodes(listTbody);

    const sortSel = $('.browsefs__sort').value;
    const sortFunc = sortFuncMap[sortSel];

    if (result['entry'] === undefined) {
      listTbody.classList.add('.browsefs__list--empty');
    } else {
      const rows = result['entry'].sort(sortFunc);
      for (let row of rows) {
        listTbody.classList.remove('.browsefs__list--empty');

        var tr = document.createElement('tr');
        tr.classList.add('browsefs__entry');
        listTbody.appendChild(tr);

        for (let colName of colNames) {
          var cell = document.createElement('td');
          cell.classList.add('browsefs__cell');
          cell.classList.add(`browsefs__cell--${colName}`);

          let val = row[colName];
          if (colName === 'type') {
            val = val[0];
          } else if (colName === 'perm_mode') {
            val = val.toString(8);
          } else if (colName === 'uid') {
            val = 'u'+val;
          } else if (colName === 'gid') {
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
            const actionDef = actionDefMap[row.type] || {labels: []};

            for (let l of actionDef.labels) {
              const span = document.createElement('span');
              span.classList.add('browsefs__action');
              span.textContent = l;
              cell.appendChild(span);
            }
            const action = actionDef.action;
            if (action !== undefined) {
              tr.addEventListener('click', (ev) => {
                actionDef.action(row);
              });
            }
          }

          tr.appendChild(cell);
        }
      }
    }
  } catch (e) {
    console.log(e);
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
