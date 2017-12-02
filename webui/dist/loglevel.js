import {contentSection, isSectionSelected} from './nav.js';
import {$, $$, removeAllChildNodes} from './domhelper.js';
import {rpc} from './api.js';

const listDiv = $('.loglevel--list');
const levels = ['debug', 'info', 'warning', 'critical'];
Object.freeze(levels);

const triggerUpdate = async () => {
  if (!isSectionSelected('loglevel'))
    return;

  try {
    removeAllChildNodes(listDiv);

    const result = await rpc('api/v1/logger/categories');
    const categories = result['category'].sort(
        (a, b) => a['category'].localeCompare(b['category']));
    for (let category of categories) {
      const name = category['category'];
      const itemDiv = document.createElement('div');
      itemDiv.classList.add('kvview__item');
      listDiv.appendChild(itemDiv);
      
      const labelDiv = document.createElement('div');
      labelDiv.classList.add('kvview__label');
      labelDiv.textContent = name;
      itemDiv.appendChild(labelDiv);

      const valueDiv = document.createElement('div');
      valueDiv.classList.add('kvview__value');
      valueDiv.classList.add('loglevel__level');
      valueDiv.id = `loglevel-${name}`;
      itemDiv.appendChild(valueDiv);

      for (let i = 0; i < levels.length; ++ i) {
        const inputId = `loglevel-${name}-${i}`;

        const input = document.createElement('input');
        input.type = 'radio';
        input.classList.add('loglevel__radio');
        input.name = `loglevel-${name}`;
        input.id = inputId;
        input.value = i;
        input.checked = (category['level'] == i);
        valueDiv.appendChild(input);

        const label = document.createElement('label');
        label.classList.add('loglevel__label');
        label.setAttribute('for', inputId);
        label.textContent = levels[i];
        valueDiv.appendChild(label);
      }
    }
  } catch (e) {
    console.log(e);
  }
}
contentSection('loglevel').addEventListener('shown', e => {
  triggerUpdate();
});
