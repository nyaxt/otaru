import {$, $$} from './util.js';

$$(".nav__item").forEach(menu_item => {
  menu_item.addEventListener("click", ev => {
    // history.pushState(null, document.title, menu_item.getAttribute("href"));
    return false; 
  });
});

const validContents = [
  'blobstore',
  'settings',
];

let showContent = () => {
  let contentId = "";
  let m;
  if (m = window.location.hash.match(/^#(\w+)$/)) {
    contentId = m[1];
  }

  if (!validContents.includes(contentId))
    return;

  const contentHash = `#${contentId}`;
  $$(".nav__tab").forEach(tab => {
    if (tab.getAttribute("href") === contentHash) {
      tab.classList.add("nav__item--selected");
    } else {
      tab.classList.remove("nav__item--selected");
    } 
  });
  const sectionNeedle = `section--${contentId}`;

  $$(".section").forEach(section => {
    if (section.classList.contains(sectionNeedle)) {
      section.classList.add("section--selected");
      section.dispatchEvent(new Event('shown'));
    } else {
      section.classList.remove("section--selected");
      section.dispatchEvent(new Event('hidden'));
    }
  });
};

window.addEventListener("hashchange", ev => {
  showContent();
})
showContent();
