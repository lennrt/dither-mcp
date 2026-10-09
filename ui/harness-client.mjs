// Development host only. This bundle is not part of the Go resource.
import { AppBridge, PostMessageTransport } from '@modelcontextprotocol/ext-apps/app-bridge';

const frame = document.getElementById('app');
const status = document.getElementById('host-status');
const audit = document.getElementById('audit');
const bootstrap = await fetch('/bootstrap').then(async (response) => {
  if (!response.ok) throw new Error(await response.text());
  return response.json();
});
const capabilities = new URLSearchParams(location.search).has('no-tools') ? {} : { serverTools: {} };
const bridge = new AppBridge(null, { name: 'Dither local test host', version: '0.1.0' }, capabilities, { hostContext: { theme: 'light', displayMode: 'inline', containerDimensions: { maxWidth: 1100 } } });
const calls = [];
bridge.oncalltool = async (params, extra) => {
  if (!['dither_studio', 'dither_render'].includes(params.name)) throw new Error('The test host only allows studio and render tools.');
  calls.push({ name: params.name, arguments: params.arguments });
  audit.textContent = JSON.stringify(calls, null, 2);
  const response = await fetch('/call', { method: 'POST', headers: { 'content-type': 'application/json' }, body: JSON.stringify(params), signal: extra.mcpReq.signal });
  if (!response.ok) throw new Error(await response.text());
  return response.json();
};
bridge.onsandboxready = async () => { await bridge.sendSandboxResourceReady({ html: bootstrap.resource.text, sandbox: 'allow-scripts allow-same-origin' }); };
bridge.oninitialized = async () => {
  await bridge.sendToolInput({ arguments: bootstrap.arguments });
  await bridge.sendToolResult(bootstrap.result);
  status.textContent = 'Connected through the official AppBridge to the Go stdio MCP server.';
};
bridge.onsizechange = ({ height }) => {
  if (typeof height === 'number' && Number.isFinite(height)) frame.style.height = Math.min(Math.max(height, 100), 3000) + 'px';
};
await bridge.connect(new PostMessageTransport(frame.contentWindow, frame.contentWindow));
frame.src = 'http://127.0.0.1:' + location.port + '/sandbox';
let theme = 'light';
document.getElementById('theme').onclick = () => {
  theme = theme === 'light' ? 'dark' : 'light'; document.body.classList.toggle('dark', theme === 'dark');
  document.getElementById('theme').textContent = theme === 'light' ? 'Dark theme' : 'Light theme'; bridge.setHostContext({ theme });
};
let narrow = false;
document.getElementById('width').onclick = () => {
  narrow = !narrow; frame.style.maxWidth = narrow ? '370px' : ''; document.getElementById('width').textContent = narrow ? 'Wide view' : 'Narrow view'; bridge.setHostContext({ containerDimensions: { maxWidth: narrow ? 370 : 1100 } });
};
document.getElementById('cancel').onclick = () => bridge.sendToolCancelled({ reason: 'Test host cancellation' });
document.getElementById('late').onclick = () => bridge.sendToolResult(bootstrap.result);
window.ditherHarness = { bridge, calls, bootstrap };
