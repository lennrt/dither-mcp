// Browser side. Bundle this module with your application; keep the MCP client in
// your desktop process or backend. No browser access to native files is needed.
import { AppBridge, PostMessageTransport } from '@modelcontextprotocol/ext-apps/app-bridge';

export async function mountDitherStudio({ iframe, sandboxUrl, bootstrap, callTool, hostContext = {}, onReady = () => {}, onError = () => {} }) {
  const sandbox = new URL(sandboxUrl, location.href);
  if (sandbox.origin === location.origin) throw new Error('The sandbox proxy must have a different origin from the host.');
  if (!iframe?.contentWindow || typeof callTool !== 'function') throw new Error('An attached iframe and callTool handler are required.');
  const bridge = new AppBridge(null, { name: 'Dither embedding example', version: '1' }, { serverTools: {} }, { hostContext: { theme: 'light', displayMode: 'inline', ...hostContext } });
  bridge.oncalltool = (params, extra) => {
    if (!['dither_studio', 'dither_render'].includes(params.name)) throw new Error('Unsupported studio tool.');
    return callTool(params, { signal: extra.mcpReq.signal });
  };
  bridge.onsandboxready = () => bridge.sendSandboxResourceReady({ html: bootstrap.resource.text, sandbox: 'allow-scripts', csp: bootstrap.resource._meta?.ui?.csp, permissions: bootstrap.resource._meta?.ui?.permissions });
  bridge.oninitialized = async () => {
    try {
      await bridge.sendToolInput({ arguments: bootstrap.arguments });
      await bridge.sendToolResult(bootstrap.result); // Include content, structuredContent and _meta.
      onReady();
    } catch (error) { onError(error); }
  };
  bridge.onsizechange = ({ height }) => {
    if (Number.isFinite(height)) iframe.style.height = `${Math.min(Math.max(height, 100), 3000)}px`;
  };
  bridge.onerror = onError;
  await bridge.connect(new PostMessageTransport(iframe.contentWindow, iframe.contentWindow));
  iframe.src = sandbox.href;
  return {
    setHostContext: (update) => bridge.setHostContext(update),
    cancel: (reason = 'Host cancelled the studio operation.') => bridge.sendToolCancelled({ reason }),
    async destroy() {
      try { await bridge.teardownResource({}, { timeout: 1000 }); } finally { await bridge.close(); iframe.removeAttribute('src'); }
    },
  };
}
