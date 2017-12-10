import {contentSection, isSectionSelected} from './nav.js';
import {$, removeAllChildNodes} from './domhelper.js';
import {rpc} from './api.js';
import {formatTimestamp} from './format.js';
import {levels} from './loglevel.js';

const perQueryLogLimit = 100;

const getLatestId = async () => {
  const resp = await rpc('api/v1/logger/latest_log_entry_id');
  return resp.id;
}

const listDiv = $('.logview__list');
const colNames = ['time', 'level', 'category', 'log', 'location'];
const triggerUpdate = async () => {
  if (!isSectionSelected('logview'))
    return;

  try {
    const latestId = await getLatestId();

    const result = await rpc('api/v1/logger/logs', {args: {
      min_id: latestId - perQueryLogLimit,
      limit: perQueryLogLimit,
    }});
    removeAllChildNodes(listDiv);
    for (let entry of result.entry) {
      const entryDiv = document.createElement('div');
      entryDiv.classList.add('logview__entry');
      listDiv.insertBefore(entryDiv, listDiv.firstChild);

      for (let colName of colNames) {
        var cell = document.createElement('div');
        cell.classList.add('logview__cell');
        cell.classList.add(`logview__cell--${colName}`);

        let val = entry[colName];
        if (colName === 'time') {
          val = formatTimestamp(new Date(val*1000), {relative: false, full: true});
        } else if (colName === 'level') {
          if (val === undefined)
            val = 0;

          val = levels[val][0].toUpperCase();
        }
        cell.textContent = val;

        entryDiv.appendChild(cell);
      }
    }
  } catch (e) {
    console.log(e);
  }
}
$('.logview__refresh').addEventListener('click', ev => {
  ev.preventDefault(); 

  triggerUpdate();
});
contentSection('logview').addEventListener('shown', triggerUpdate);
