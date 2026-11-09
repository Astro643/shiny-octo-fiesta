(() => {
  "use strict";
  const $ = id => document.getElementById(id);
  const root = $("sw"), launch = $("sw-launch"), panel = $("sw-panel"), dot = $("sw-dot");
  const thread = $("sw-thread"), form = $("sw-form"), input = $("sw-input"), sendBtn = $("sw-send");
  const STORE = "support_chat_v1", MAX_KEPT = 30;
  const CHECK = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><path d="M5.5 12.5l4.2 4.2L18.5 7.8"/></svg>';
  const WARN = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round"><path d="M12 6v8M12 18h.01"/></svg>';
  const CHAT = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linejoin="round" stroke-linecap="round"><path d="M5 5h14a1 1 0 0 1 1 1v9a1 1 0 0 1-1 1h-7l-4 3.5V16H5a1 1 0 0 1-1-1V6a1 1 0 0 1 1-1z"/><path d="M8 9.5h8M8 12.5h5"/></svg>';
  const SUGGESTIONS = ["Is everything working right now?", "How do I get my magic link?", "Why did my request fail?"];

  let open = false, tab = "home", busy = false, status = null, statusTimer = null;
  let messages = [];

  // Chat history lives only in this browser; storage can be unavailable, so every access is guarded.
  const load = () => { try { const v = JSON.parse(localStorage.getItem(STORE) || "[]"); return Array.isArray(v) ? v : []; } catch { return []; } };
  const save = () => { try { localStorage.setItem(STORE, JSON.stringify(messages.slice(-MAX_KEPT))); } catch {} };

  function el(tag, cls, text) {
    const n = document.createElement(tag);
    if (cls) n.className = cls;
    if (text !== undefined) n.textContent = text;   // text only, never HTML
    return n;
  }

  /* ---------- open / close / tabs ---------- */
  function setOpen(v) {
    open = v;
    root.classList.toggle("sw-open", v);
    launch.setAttribute("aria-expanded", String(v));
    launch.setAttribute("aria-label", v ? "Close support" : "Open support and system status");
    panel.setAttribute("aria-hidden", String(!v));
    if (v) { fetchStatus(); if (tab === "messages") setTimeout(() => input.focus(), 120); }
    schedule();
  }
  function setTab(t) {
    tab = t;
    $("sw-home").classList.toggle("on", t === "home");
    $("sw-messages").classList.toggle("on", t === "messages");
    $("sw-tab-home").classList.toggle("on", t === "home");
    $("sw-tab-messages").classList.toggle("on", t === "messages");
    if (t === "messages") { renderThread(); setTimeout(() => input.focus(), 60); }
  }
  function openChat() { setTab("messages"); setOpen(true); }

  launch.addEventListener("click", () => setOpen(!open));
  document.querySelectorAll("[data-sw-close]").forEach(b => b.addEventListener("click", () => setOpen(false)));
  document.querySelectorAll(".sw-tab").forEach(b => b.addEventListener("click", () => setTab(b.dataset.tab)));
  $("sw-ask").addEventListener("click", openChat);
  document.querySelectorAll("[data-open-support]").forEach(b => b.addEventListener("click", e => { e.preventDefault(); openChat(); }));
  document.addEventListener("keydown", e => { if (e.key === "Escape" && open) { setOpen(false); launch.focus(); } });

  /* ---------- live status ---------- */
  function ago(iso) {
    const s = Math.max(0, Math.round((Date.now() - new Date(iso).getTime()) / 1000));
    if (!isFinite(s)) return "";
    if (s < 10) return "Updated just now";
    if (s < 90) return "Updated " + s + "s ago";
    return "Updated " + Math.round(s / 60) + " min ago";
  }
  function renderStatus() {
    const list = $("sw-list"), ico = $("sw-ico"), label = $("sw-label"), sub = $("sw-sub");
    list.replaceChildren();
    if (!status) {
      dot.className = "sw-dot";
      ico.className = "sw-ico"; ico.innerHTML = WARN;
      label.textContent = "Status unavailable";
      sub.textContent = "We could not load the live status. Try again shortly.";
      $("sw-updated").textContent = "";
      return;
    }
    dot.className = "sw-dot " + status.overall;
    ico.className = "sw-ico " + status.overall;
    ico.innerHTML = status.overall === "operational" ? CHECK : WARN;
    label.textContent = status.label;
    sub.textContent = status.overall === "operational" ? "Everything is running normally" : "We are aware and working through it";
    $("sw-updated").textContent = ago(status.updated);
    for (const c of status.components) {
      const row = el("div", "sw-comp");
      row.appendChild(el("span", "", c.name));
      const right = el("span");
      right.appendChild(el("span", "", c.detail));
      const d = el("i", c.status);
      right.appendChild(d);
      row.appendChild(right);
      list.appendChild(row);
    }
  }
  async function fetchStatus() {
    try {
      const r = await fetch("/api/public/status", {cache: "no-store"});
      if (!r.ok) throw new Error();
      status = await r.json();
    } catch { if (!status) status = null; }
    renderStatus();
  }
  // Refresh often while the panel is open, and now and then in the background so the launcher dot stays honest.
  function schedule() {
    clearInterval(statusTimer);
    statusTimer = setInterval(() => { if (!document.hidden) fetchStatus(); }, open ? 20000 : 60000);
  }

  /* ---------- AI chat ---------- */
  function renderThread() {
    thread.replaceChildren();
    if (!messages.length) {
      const box = el("div", "sw-empty");
      box.innerHTML = CHAT;
      box.appendChild(el("h4", "", "No messages yet"));
      box.appendChild(el("p", "", "Ask anything about the bot. Answers come from an AI assistant, available around the clock."));
      const chips = el("div", "sw-chips");
      SUGGESTIONS.forEach(s => {
        const c = el("button", "sw-chip", s); c.type = "button";
        c.addEventListener("click", () => send(s));
        chips.appendChild(c);
      });
      box.appendChild(chips);
      thread.appendChild(box);
      return;
    }
    messages.forEach(m => thread.appendChild(el("div", "sw-msg " + (m.role === "user" ? "user" : "bot") + (m.error ? " err" : ""), m.text)));
    if (busy) { const t = el("div", "sw-typing"); t.append(el("i"), el("i"), el("i")); thread.appendChild(t); }
    thread.scrollTop = thread.scrollHeight;
  }
  async function send(text) {
    text = (text || "").trim();
    if (!text || busy) return;
    messages.push({role: "user", text});
    busy = true; sendBtn.disabled = true; input.value = ""; syncReady(); renderThread();
    const history = messages.slice(0, -1).filter(m => !m.error).slice(-8).map(m => ({role: m.role, text: m.text.slice(0, 800)}));
    try {
      const r = await fetch("/api/support/chat", {
        method: "POST", headers: {"Content-Type": "application/json"},
        body: JSON.stringify({message: text.slice(0, 600), history})
      });
      const data = await r.json().catch(() => ({}));
      if (r.ok && data.reply) messages.push({role: "assistant", text: data.reply});
      else messages.push({role: "assistant", error: true,
        text: typeof data.detail === "string" ? data.detail : "Sorry, something went wrong. Please try again."});
    } catch {
      messages.push({role: "assistant", error: true, text: "Network error. Please check your connection and try again."});
    }
    busy = false; sendBtn.disabled = false; save(); renderThread(); input.focus();
  }
  const syncReady = () => form.classList.toggle("ready", input.value.trim().length > 0);
  input.addEventListener("input", syncReady);
  form.addEventListener("submit", e => { e.preventDefault(); send(input.value); });
  $("sw-new").addEventListener("click", () => { if (busy) return; messages = []; save(); renderThread(); input.focus(); });

  /* ---------- start ---------- */
  messages = load();
  schedule();
  setTimeout(fetchStatus, 800);
  if (location.hash === "#support") openChat();
})();
