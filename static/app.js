const input = document.querySelector('#file-input');
const list = document.querySelector('#file-list');
const target = document.querySelector('#target');
const convert = document.querySelector('#convert');
const status = document.querySelector('#status');
const download = document.querySelector('#download');
const dropzone = document.querySelector('#dropzone');
let files = [];
let busy = false;

const sizeLabel = bytes => bytes < 1024 * 1024 ? `${(bytes / 1024).toFixed(0)} KB` : `${(bytes / 1024 / 1024).toFixed(1)} MB`;

function render() {
  list.replaceChildren(...files.map((file, index) => {
    const row = document.createElement('li');
    row.className = 'file-row';
    const type = document.createElement('span');
    type.className = 'file-type';
    type.textContent = file.name.split('.').pop().slice(0, 5);
    const meta = document.createElement('span');
    meta.className = 'file-meta';
    const name = document.createElement('span');
    name.className = 'file-name';
    name.textContent = file.name;
    const size = document.createElement('span');
    size.className = 'file-size';
    size.textContent = sizeLabel(file.size);
    meta.append(name, size);
    const remove = document.createElement('button');
    remove.className = 'remove';
    remove.type = 'button';
    remove.setAttribute('aria-label', `Remove ${file.name}`);
    remove.textContent = '×';
    remove.disabled = busy;
    remove.addEventListener('click', () => { files.splice(index, 1); refresh(); });
    row.append(type, meta, remove);
    return row;
  }));
  convert.disabled = busy || !files.length || !target.value;
  input.disabled = busy;
}

async function refresh() {
  render();
  download.replaceChildren();
  status.className = 'status';
  status.textContent = '';
  const extensions = [...new Set(files.map(file => file.name.split('.').pop().toLowerCase()))];
  if (!extensions.length) {
    target.replaceChildren(new Option('Select files first', ''));
    target.disabled = true;
    render();
    return;
  }
  const query = new URLSearchParams();
  extensions.forEach(ext => query.append('ext', ext));
  try {
    const response = await fetch(`/formats?${query}`);
    const data = await response.json();
    target.replaceChildren(...data.formats.map(ext => new Option(ext.toUpperCase(), ext)));
    if (!data.formats.length) target.add(new Option('No supported output', ''));
    target.disabled = busy || !data.formats.length;
  } catch {
    target.replaceChildren(new Option('Formats unavailable', ''));
    target.disabled = true;
  }
  render();
}

function addFiles(selected) {
  status.className = 'status';
  status.textContent = '';
  download.replaceChildren();
  const incoming = [...selected];
  if (files.length + incoming.length > 10 || files.reduce((sum, f) => sum + f.size, 0) + incoming.reduce((sum, f) => sum + f.size, 0) > 25 * 1024 * 1024) {
    status.className = 'status error';
    status.textContent = 'Choose up to 10 files with a combined size under 25 MB.';
    return;
  }
  files = [...files, ...incoming];
  refresh();
}

input.addEventListener('change', event => { addFiles(event.target.files); input.value = ''; });
dropzone.addEventListener('dragover', event => { event.preventDefault(); dropzone.classList.add('dragover'); });
dropzone.addEventListener('dragleave', () => dropzone.classList.remove('dragover'));
dropzone.addEventListener('drop', event => { event.preventDefault(); dropzone.classList.remove('dragover'); addFiles(event.dataTransfer.files); });
target.addEventListener('change', render);

convert.addEventListener('click', async () => {
  if (busy || !files.length || !target.value) return;
  busy = true;
  render();
  download.replaceChildren();
  status.className = 'status';
  status.textContent = 'Processing your files…';
  const body = new FormData();
  files.forEach(file => body.append('files', file, file.name));
  body.append('target', target.value);
  try {
    const response = await fetch('/convert', { method: 'POST', body });
    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Conversion failed.');
    }
    const blob = await response.blob();
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.className = 'download-link';
    link.href = url;
    link.download = files.length > 1 ? 'converted-files.zip' : `${files[0].name.replace(/\.[^.]+$/, '')}.${target.value}`;
    link.textContent = 'Download converted files';
    link.addEventListener('click', () => setTimeout(() => URL.revokeObjectURL(url), 1000), { once: true });
    download.append(link);
    status.textContent = 'Conversion complete.';
  } catch (error) {
    status.className = 'status error';
    status.textContent = error.message;
  } finally {
    busy = false;
    refresh();
  }
});
