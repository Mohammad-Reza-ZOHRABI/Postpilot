import { writable } from 'svelte/store';

export const route = writable('/');

function update() {
  route.set(window.location.hash.slice(1) || '/');
}

window.addEventListener('hashchange', update);
update();

export function navigate(path) {
  window.location.hash = '#' + path;
}
