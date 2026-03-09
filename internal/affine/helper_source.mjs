import { io } from 'socket.io-client';
import * as Y from 'yjs';

const ACK_TIMEOUT_MS = 10000;
const CONNECT_TIMEOUT_MS = 10000;
const ID_CHARS = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-';

function emitWithAck(socket, event, payload) {
  return new Promise((resolve, reject) => {
    let settled = false;
    const timer = setTimeout(() => {
      if (settled) {
        return;
      }
      settled = true;
      reject(new Error(`${event} timeout after ${ACK_TIMEOUT_MS}ms`));
    }, ACK_TIMEOUT_MS);

    socket.emit(event, payload, ack => {
      if (settled) {
        return;
      }
      settled = true;
      clearTimeout(timer);
      if (ack?.error) {
        const message = typeof ack.error.message === 'string' && ack.error.message.trim()
          ? ack.error.message
          : `${event} failed`;
        reject(new Error(message));
        return;
      }
      resolve(ack?.data ?? {});
    });
  });
}

function wsURLFromBaseURL(baseURL) {
  return baseURL
    .replace(/^https:\/\//, 'wss://')
    .replace(/^http:\/\//, 'ws://')
    .replace(/\/+$/, '');
}

function connectSocket(baseURL, token) {
  return new Promise((resolve, reject) => {
    const socket = io(wsURLFromBaseURL(baseURL), {
      transports: ['websocket'],
      path: '/socket.io/',
      extraHeaders: {
        Authorization: `Bearer ${token}`,
      },
      autoConnect: true,
    });

    let settled = false;
    const timer = setTimeout(() => {
      if (settled) {
        return;
      }
      settled = true;
      socket.disconnect();
      reject(new Error(`socket connect timeout after ${CONNECT_TIMEOUT_MS}ms`));
    }, CONNECT_TIMEOUT_MS);

    const cleanup = () => {
      clearTimeout(timer);
      socket.off('connect', onConnect);
      socket.off('connect_error', onError);
    };

    const onConnect = () => {
      if (settled) {
        return;
      }
      settled = true;
      cleanup();
      resolve(socket);
    };

    const onError = err => {
      if (settled) {
        return;
      }
      settled = true;
      cleanup();
      socket.disconnect();
      reject(err instanceof Error ? err : new Error(String(err)));
    };

    socket.on('connect', onConnect);
    socket.on('connect_error', onError);
  });
}

async function joinWorkspace(socket, workspaceId) {
  await emitWithAck(socket, 'space:join', {
    spaceType: 'workspace',
    spaceId: workspaceId,
    clientVersion: '0.26.0',
  });
}

async function loadDoc(socket, workspaceId, docId) {
  try {
    return await emitWithAck(socket, 'space:load-doc', {
      spaceType: 'workspace',
      spaceId: workspaceId,
      docId,
    });
  } catch (err) {
    if (String(err.message || err).includes('DOC_NOT_FOUND')) {
      return {};
    }
    throw err;
  }
}

async function pushDocUpdate(socket, workspaceId, docId, update) {
  await emitWithAck(socket, 'space:push-doc-update', {
    spaceType: 'workspace',
    spaceId: workspaceId,
    docId,
    update: Buffer.from(update).toString('base64'),
  });
}

function generateId() {
  let out = '';
  for (let i = 0; i < 10; i += 1) {
    out += ID_CHARS[Math.floor(Math.random() * ID_CHARS.length)];
  }
  return out;
}

function blockVersion(flavour) {
  switch (flavour) {
    case 'affine:page':
      return 2;
    case 'affine:surface':
      return 5;
    default:
      return 1;
  }
}

function setSysFields(block, blockId, flavour) {
  block.set('sys:id', blockId);
  block.set('sys:flavour', flavour);
  block.set('sys:version', blockVersion(flavour));
}

function makeText(content) {
  const yText = new Y.Text();
  if (content.length > 0) {
    yText.insert(0, content);
  }
  return yText;
}

function parseMarkdown(markdown) {
  const src = markdown.replace(/\r\n/g, '\n');
  const lines = src.split('\n');
  const ops = [];
  let i = 0;

  const flushParagraph = paragraphLines => {
    const text = paragraphLines.join('\n').trim();
    if (text.length > 0) {
      ops.push({ type: 'paragraph', text });
    }
  };

  while (i < lines.length) {
    const line = lines[i];
    const trimmed = line.trim();

    if (trimmed === '') {
      i += 1;
      continue;
    }

    if (/^(```|~~~)/.test(trimmed)) {
      const fence = trimmed.slice(0, 3);
      const language = trimmed.slice(3).trim();
      const body = [];
      i += 1;
      while (i < lines.length && !lines[i].trim().startsWith(fence)) {
        body.push(lines[i]);
        i += 1;
      }
      if (i < lines.length) {
        i += 1;
      }
      ops.push({ type: 'code', language, text: body.join('\n') });
      continue;
    }

    if (/^([-*_])(?:\s*\1){2,}$/.test(trimmed)) {
      ops.push({ type: 'divider' });
      i += 1;
      continue;
    }

    const headingMatch = /^(#{2,6})\s+(.*)$/.exec(trimmed);
    if (headingMatch) {
      ops.push({
        type: 'heading',
        level: headingMatch[1].length,
        text: headingMatch[2].trim(),
      });
      i += 1;
      continue;
    }

    const todoMatch = /^[-*]\s+\[( |x|X)\]\s+(.*)$/.exec(trimmed);
    if (todoMatch) {
      ops.push({
        type: 'list',
        style: 'todo',
        checked: todoMatch[1].toLowerCase() === 'x',
        text: todoMatch[2].trim(),
      });
      i += 1;
      continue;
    }

    const bulletMatch = /^[-*]\s+(.*)$/.exec(trimmed);
    if (bulletMatch) {
      ops.push({ type: 'list', style: 'bulleted', text: bulletMatch[1].trim() });
      i += 1;
      continue;
    }

    const numberedMatch = /^\d+\.\s+(.*)$/.exec(trimmed);
    if (numberedMatch) {
      ops.push({ type: 'list', style: 'numbered', text: numberedMatch[1].trim() });
      i += 1;
      continue;
    }

    if (trimmed.startsWith('>')) {
      const quoteLines = [];
      while (i < lines.length && lines[i].trim().startsWith('>')) {
        quoteLines.push(lines[i].trim().replace(/^>\s?/, ''));
        i += 1;
      }
      ops.push({ type: 'quote', text: quoteLines.join('\n').trim() });
      continue;
    }

    const paragraphLines = [line];
    i += 1;
    while (i < lines.length) {
      const next = lines[i];
      const nextTrimmed = next.trim();
      if (
        nextTrimmed === '' ||
        /^(```|~~~)/.test(nextTrimmed) ||
        /^([-*_])(?:\s*\1){2,}$/.test(nextTrimmed) ||
        /^(#{2,6})\s+/.test(nextTrimmed) ||
        /^[-*]\s+\[( |x|X)\]\s+/.test(nextTrimmed) ||
        /^[-*]\s+/.test(nextTrimmed) ||
        /^\d+\.\s+/.test(nextTrimmed) ||
        nextTrimmed.startsWith('>')
      ) {
        break;
      }
      paragraphLines.push(next);
      i += 1;
    }
    flushParagraph(paragraphLines);
  }

  return ops;
}

function createBlock(op) {
  const blockId = generateId();
  const block = new Y.Map();

  switch (op.type) {
    case 'heading':
    case 'paragraph':
    case 'quote': {
      setSysFields(block, blockId, 'affine:paragraph');
      block.set('sys:parent', null);
      block.set('sys:children', new Y.Array());
      block.set('prop:type', op.type === 'heading' ? `h${op.level}` : op.type === 'quote' ? 'quote' : 'text');
      block.set('prop:text', makeText(op.text || ''));
      return { blockId, block };
    }
    case 'list': {
      setSysFields(block, blockId, 'affine:list');
      block.set('sys:parent', null);
      block.set('sys:children', new Y.Array());
      block.set('prop:type', op.style);
      block.set('prop:checked', op.style === 'todo' ? !!op.checked : false);
      block.set('prop:text', makeText(op.text || ''));
      return { blockId, block };
    }
    case 'code': {
      setSysFields(block, blockId, 'affine:code');
      block.set('sys:parent', null);
      block.set('sys:children', new Y.Array());
      block.set('prop:language', op.language || '');
      block.set('prop:text', makeText(op.text || ''));
      return { blockId, block };
    }
    case 'divider':
    default: {
      setSysFields(block, blockId, 'affine:divider');
      block.set('sys:parent', null);
      block.set('sys:children', new Y.Array());
      return { blockId, block };
    }
  }
}

function createDocumentUpdate(docId, title, markdown) {
  const doc = new Y.Doc();
  const blocks = doc.getMap('blocks');

  const pageId = generateId();
  const page = new Y.Map();
  setSysFields(page, pageId, 'affine:page');
  page.set('prop:title', makeText(title));
  const pageChildren = new Y.Array();
  page.set('sys:children', pageChildren);
  blocks.set(pageId, page);

  const surfaceId = generateId();
  const surface = new Y.Map();
  setSysFields(surface, surfaceId, 'affine:surface');
  surface.set('sys:parent', null);
  surface.set('sys:children', new Y.Array());
  const elements = new Y.Map();
  elements.set('type', '$blocksuite:internal:native$');
  elements.set('value', new Y.Map());
  surface.set('prop:elements', elements);
  blocks.set(surfaceId, surface);
  pageChildren.push([surfaceId]);

  const noteId = generateId();
  const note = new Y.Map();
  setSysFields(note, noteId, 'affine:note');
  note.set('sys:parent', null);
  note.set('prop:displayMode', 'both');
  note.set('prop:xywh', '[0,0,800,95]');
  note.set('prop:index', 'a0');
  note.set('prop:hidden', false);
  const background = new Y.Map();
  background.set('light', '#ffffff');
  background.set('dark', '#252525');
  note.set('prop:background', background);
  const noteChildren = new Y.Array();
  note.set('sys:children', noteChildren);
  blocks.set(noteId, note);
  pageChildren.push([noteId]);

  for (const op of parseMarkdown(markdown)) {
    const { blockId, block } = createBlock(op);
    blocks.set(blockId, block);
    noteChildren.push([blockId]);
  }

  const meta = doc.getMap('meta');
  meta.set('id', docId);
  meta.set('title', title);
  meta.set('createDate', Date.now());
  meta.set('tags', new Y.Array());

  return Y.encodeStateAsUpdate(doc);
}

function createWorkspacePageDelta(workspaceUpdateBase64, docId, title) {
  const wsDoc = new Y.Doc();
  if (workspaceUpdateBase64) {
    Y.applyUpdate(wsDoc, Buffer.from(workspaceUpdateBase64, 'base64'));
  }
  const prevSV = Y.encodeStateVector(wsDoc);
  const meta = wsDoc.getMap('meta');
  let pages = meta.get('pages');
  if (!(pages instanceof Y.Array)) {
    pages = new Y.Array();
    meta.set('pages', pages);
  }
  const entry = new Y.Map();
  entry.set('id', docId);
  entry.set('title', title);
  entry.set('createDate', Date.now());
  entry.set('tags', new Y.Array());
  pages.push([entry]);
  return Y.encodeStateAsUpdate(wsDoc, prevSV);
}

async function main() {
  const input = JSON.parse(await new Promise((resolve, reject) => {
    let data = '';
    process.stdin.setEncoding('utf8');
    process.stdin.on('data', chunk => {
      data += chunk;
    });
    process.stdin.on('end', () => resolve(data));
    process.stdin.on('error', reject);
  }));

  const socket = await connectSocket(input.baseURL, input.token);
  try {
    await joinWorkspace(socket, input.workspaceID);
    const docId = generateId();
    const docUpdate = createDocumentUpdate(docId, input.title, input.markdown);
    await pushDocUpdate(socket, input.workspaceID, docId, docUpdate);

    const workspaceSnapshot = await loadDoc(socket, input.workspaceID, input.workspaceID);
    const workspaceDelta = createWorkspacePageDelta(workspaceSnapshot.missing, docId, input.title);
    await pushDocUpdate(socket, input.workspaceID, input.workspaceID, workspaceDelta);

    process.stdout.write(JSON.stringify({ ok: true, docId }));
  } finally {
    socket.disconnect();
  }
}

main().catch(err => {
  process.stdout.write(JSON.stringify({
    ok: false,
    error: err instanceof Error ? err.message : String(err),
  }));
  process.exitCode = 1;
});
