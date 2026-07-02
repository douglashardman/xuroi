export function threadIdFromParam(threadParam: string): string | null {
  const trimmed = threadParam.trim();
  const sep = trimmed.lastIndexOf('--');
  if (sep < 0) return null;
  const id = trimmed.slice(sep + 2).trim();
  return id || null;
}

export async function fetchThreadLLMText(threadParam: string): Promise<Response> {
  const id = threadIdFromParam(threadParam);
  if (!id) {
    return new Response('invalid thread url — expected /t/{slug}--{id}/llm.txt', { status: 400 });
  }
  const { API_URL } = await import('./server-api');
  const res = await fetch(`${API_URL}/v1/threads/${encodeURIComponent(id)}/llm.txt`);
  const body = await res.text();
  if (!res.ok) {
    return new Response(body || 'thread not found', { status: res.status });
  }
  return new Response(body, {
    status: 200,
    headers: { 'Content-Type': 'text/plain; charset=utf-8' },
  });
}