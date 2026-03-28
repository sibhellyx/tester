export const BASE = "http://localhost:8080/api/v1";

export const api = async (path, opts = {}) => {
  const res = await fetch(BASE + path, opts);
  if (!res.ok) {
    let detail = `${res.status} ${res.statusText}`;
    try {
      const body = await res.json();
      detail = body.error || body.message || body.detail || JSON.stringify(body);
    } catch {
      try { detail = await res.text() || detail; } catch {}
    }
    throw new Error(detail);
  }
  if (res.status === 204) return null;
  return res.json();
};