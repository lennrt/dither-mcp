import { mountDitherStudio } from './studio-host.mjs';
const status = document.getElementById('status');
const session = document.querySelector('meta[name="studio-session"]').content;
const headers = { 'x-studio-session': session };
try {
  const response = await fetch('/bootstrap', { headers });
  if (!response.ok) throw new Error(await response.text());
  const bootstrap = await response.json();
  const mounted = await mountDitherStudio({
    iframe: document.getElementById('studio'), sandboxUrl: `http://127.0.0.1:${location.port}/sandbox`, bootstrap,
    callTool: async (params, { signal }) => {
      const result = await fetch('/call', { method: 'POST', headers: { ...headers, 'content-type': 'application/json' }, body: JSON.stringify(params), signal });
      if (!result.ok) throw new Error(await result.text());
      return result.json();
    },
    hostContext: { containerDimensions: { maxWidth: 1100 } },
    onReady: () => { status.textContent = 'Connected to the local Go server.'; },
    onError: (error) => { status.textContent = error.message; },
  });
  let theme = 'light';
  document.getElementById('theme').onclick = () => { theme = theme === 'light' ? 'dark' : 'light'; mounted.setHostContext({ theme }); };
  document.getElementById('close').onclick = async () => {
    await mounted.destroy();
    document.getElementById('theme').disabled = true; document.getElementById('close').disabled = true;
    status.textContent = 'Studio closed.';
  };
} catch (error) { status.textContent = error.message; }
