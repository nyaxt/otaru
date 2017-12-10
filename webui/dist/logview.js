import {contentSection, isSectionSelected} from './nav.js';
import {$, removeAllChildNodes} from './domhelper.js';
import {rpc} from './api.js';
import {formatTimestamp} from './format.js';
import {levels} from './loglevel.js';

const scrollDocumentToTopAsync = () => {
  requestAnimationFrame(() => {
    document.documentElement.scrollTop = 0;
  });
};

const scrollDocumentToBottomAsync = () => {
  requestAnimationFrame(() => {
    document.documentElement.scrollTop = document.documentElement.scrollHeight;
  });
};

const perQueryLogLimit = 100;
const moreQueryLogLimit = 10;

const getLatestId = async () => {
  const resp = await rpc('api/v1/logger/latest_log_entry_id');
  return resp.id;
}

let oldestEntryId = 0;
const listDiv = $('.logview__list');
const colNames = ['time', 'level', 'category', 'log', 'location'];
const divFromEntry = (entry) => {
  const entryDiv = document.createElement('div');
  entryDiv.classList.add('logview__entry');

  for (let colName of colNames) {
    var cell = document.createElement('div');
    cell.classList.add('logview__cell');
    cell.classList.add(`logview__cell--${colName}`);

    let val = entry[colName];
    if (colName === 'time') {
      val = formatTimestamp(new Date(val*1000), {relative: false, full: true});
      // val = entry['id'];
    } else if (colName === 'level') {
      if (val === undefined)
        val = 0;

      val = levels[val][0].toUpperCase();
    }
    cell.textContent = val;

    entryDiv.appendChild(cell);
  }

  return entryDiv;
}

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
    if (!result.entry)
      return;

    for (let entry of result.entry) {
      const entryDiv = divFromEntry(entry);
      listDiv.appendChild(entryDiv);
    }
    oldestEntryId = result.entry[0].id;
    scrollDocumentToBottomAsync();
  } catch (e) {
    console.log(e);
  }
}
$('.logview__more').addEventListener('click', async ev => {
  ev.preventDefault(); 
  if (oldestEntryId == 0)
    return;

  const result = await rpc('api/v1/logger/logs', {args: {
    min_id: oldestEntryId - moreQueryLogLimit,
    limit: moreQueryLogLimit,
  }});
  if (!result.entry)
    return;

  for (let i = result.entry.length-1; i >= 0; i--) {
    let entry = result.entry[i];

    if (oldestEntryId <= entry.id)
      continue;

    const entryDiv = divFromEntry(entry);
    listDiv.insertBefore(entryDiv, listDiv.firstChild);
  }
  oldestEntryId = result.entry[0].id;
  scrollDocumentToTopAsync();
});
$('.logview__refresh').addEventListener('click', ev => {
  ev.preventDefault(); 

  triggerUpdate();
});
contentSection('logview').addEventListener('shown', triggerUpdate);
