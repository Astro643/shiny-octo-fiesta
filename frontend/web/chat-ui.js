/* Shared helpers for human-support.html and team.html. Everything the visitors type is shown with textContent, never as HTML. */
(function () {
  "use strict";
  var NS = "http://www.w3.org/2000/svg", INK = "#1B1A1C";
  var COLORS = { amber: "#F5B63D", coral: "#F26B5B", mint: "#4FD1A1", sky: "#5AB2F2", violet: "#9A7BF0", rose: "#EE7FB0", lime: "#A6D84B", slate: "#8B93A3" };
  var eyes = function (y, x1, x2, r) { return '<circle cx="' + x1 + '" cy="' + y + '" r="' + r + '" fill="' + INK + '"/><circle cx="' + x2 + '" cy="' + y + '" r="' + r + '" fill="' + INK + '"/>'; };
  var FACES = {
    fox: function (c) { return '<polygon points="5,17 9,2 19,10" fill="' + c + '"/><polygon points="35,17 31,2 21,10" fill="' + c + '"/><circle cx="20" cy="22" r="16" fill="' + c + '"/><ellipse cx="20" cy="29" rx="8" ry="5.5" fill="#fff" fill-opacity=".92"/>' + eyes(21, 14, 26, 2.2) + '<circle cx="20" cy="27" r="1.8" fill="' + INK + '"/>'; },
    owl: function (c) { return '<polygon points="7,12 9,2 17,9" fill="' + c + '"/><polygon points="33,12 31,2 23,9" fill="' + c + '"/><circle cx="20" cy="21" r="17" fill="' + c + '"/><circle cx="13.5" cy="19" r="6" fill="#fff"/><circle cx="26.5" cy="19" r="6" fill="#fff"/>' + eyes(19, 13.5, 26.5, 2.6) + '<polygon points="20,23 17,28 23,28" fill="' + INK + '"/>'; },
    bear: function (c) { return '<circle cx="8" cy="9" r="5.5" fill="' + c + '"/><circle cx="32" cy="9" r="5.5" fill="' + c + '"/><circle cx="20" cy="22" r="16" fill="' + c + '"/><ellipse cx="20" cy="27.5" rx="7" ry="5" fill="#fff" fill-opacity=".92"/>' + eyes(19, 14, 26, 2) + '<ellipse cx="20" cy="25" rx="2.6" ry="1.9" fill="' + INK + '"/>'; },
    cat: function (c) { return '<polygon points="6,16 8,2 19,9" fill="' + c + '"/><polygon points="34,16 32,2 21,9" fill="' + c + '"/><circle cx="20" cy="22" r="16" fill="' + c + '"/>' + eyes(20, 14, 26, 2.2) + '<polygon points="20,25 18,27 22,27" fill="' + INK + '"/><path d="M4 25h9M4 29l9-2M36 25h-9M36 29l-9-2" stroke="#fff" stroke-opacity=".85" stroke-width="1" fill="none"/>'; },
    frog: function (c) { return '<circle cx="20" cy="24" r="15" fill="' + c + '"/><circle cx="11.5" cy="11" r="6.5" fill="' + c + '"/><circle cx="28.5" cy="11" r="6.5" fill="' + c + '"/><circle cx="11.5" cy="11" r="4" fill="#fff"/><circle cx="28.5" cy="11" r="4" fill="#fff"/>' + eyes(11, 11.5, 28.5, 2) + '<path d="M11 28q9 6 18 0" stroke="' + INK + '" stroke-width="1.8" fill="none" stroke-linecap="round"/>'; },
    bunny: function (c) { return '<ellipse cx="13.5" cy="9" rx="4" ry="9" fill="' + c + '"/><ellipse cx="26.5" cy="9" rx="4" ry="9" fill="' + c + '"/><circle cx="20" cy="24" r="14.5" fill="' + c + '"/>' + eyes(22, 14.5, 25.5, 2) + '<ellipse cx="20" cy="27.5" rx="2.2" ry="1.6" fill="' + INK + '"/>'; }
  };
  function h(tag, cls, text) { var e = document.createElement(tag); if (cls) e.className = cls; if (text != null) e.textContent = text; return e; }
  function svg(size, inner, vb) { var s = document.createElementNS(NS, "svg"); s.setAttribute("viewBox", vb || "0 0 40 40"); s.setAttribute("width", size); s.setAttribute("height", size); s.setAttribute("aria-hidden", "true"); s.setAttribute("class", "av"); s.innerHTML = inner; return s; }
  // Avatars: if window.CU_AVATAR_BASE is set (your Neon bucket URL), they load as images from there and fall back to
  // the drawn version if an image is missing. Only whitelisted names are drawn, so a doctored value can never inject markup.
  function faceInner(key) { var p = String(key || "").split("-"); var a = FACES[p[0]] ? p[0] : "fox"; return FACES[a](COLORS[p[1]] || COLORS.slate); }
  var SUPPORT_INNER = '<rect x="2" y="2" width="36" height="36" rx="10" fill="#2D2B2D" stroke="#555" /><path d="M11 14h18L20 28z" fill="#2EFF9B"/>';
  function remote(file, size, fallback) {
    var base = (window.CU_AVATAR_BASE || "").replace(/\/+$/, ""); if (!base) return fallback();
    var img = document.createElement("img"); img.className = "av"; img.alt = ""; img.width = img.height = size; img.decoding = "async";
    img.onerror = function () { img.onerror = null; if (img.parentNode) img.parentNode.replaceChild(fallback(), img); };
    img.src = base + "/" + file; return img;
  }
  function avatar(key, size) { size = size || 30; var p = String(key || "").split("-");
    var safe = FACES[p[0]] && COLORS[p[1]] ? p[0] + "-" + p[1] : "fox-slate";
    return remote("avatars/" + safe + ".svg", size, function () { return svg(size, faceInner(key)); }); }
  function supportAvatar(size) { size = size || 30; return remote("avatars/support.svg", size, function () { return svg(size, SUPPORT_INNER); }); }
  // Full standalone SVG text, used by tools/make-avatars.js to produce the files uploaded to the bucket.
  function avatarFile(key) { return '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 40 40" width="80" height="80">' + (key === "support" ? SUPPORT_INNER : faceInner(key)) + "</svg>\n"; }
  function bell(dot) { var s = h("span", "bell"); s.innerHTML = '<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor" aria-hidden="true"><path d="M12 22a2.5 2.5 0 0 0 2.4-1.8H9.6A2.5 2.5 0 0 0 12 22zm7-5.5-1.5-1.6V10a5.5 5.5 0 0 0-4-5.3V4a1.5 1.5 0 0 0-3 0v.7a5.5 5.5 0 0 0-4 5.3v4.9L5 16.5V18h14z"/></svg>'; if (dot) s.appendChild(h("i")); return s; }
  function time(iso) { var d = new Date(iso); if (isNaN(d)) return ""; var n = new Date();
    return d.toDateString() === n.toDateString() ? d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }) : d.toLocaleDateString([], { month: "short", day: "numeric" }); }
  function api(path, opt) {
    opt = opt || {};
    return fetch(path, { method: opt.method || "GET", credentials: "same-origin", headers: { "Content-Type": "application/json", "X-Requested-With": "support" },
      body: opt.body ? JSON.stringify(opt.body) : undefined }).then(function (res) {
      return res.json().catch(function () { return null; }).then(function (data) {
        if (!res.ok) { var e = new Error((data && data.detail) || "Something went wrong (" + res.status + ")"); e.status = res.status; throw e; }
        return data; });
    });
  }
  // Draws the messages of one thread. who(m) -> {name, av} for visitor/operator messages; system messages are centred notes.
  function renderThread(box, msgs, who) {
    var near = box.scrollHeight - box.scrollTop - box.clientHeight < 80; box.textContent = "";
    msgs.forEach(function (m) {
      if (m.sender === "system") { box.appendChild(h("div", "sys", m.body)); return; }
      var w = who(m), row = h("div", "msg"), tx = h("div", "tx"), nm = h("div", "nm", w.name);
      nm.appendChild(h("time", "", time(m.at))); tx.append(nm, h("div", "bd", m.body)); row.append(w.av, tx); box.appendChild(row);
    });
    if (near) box.scrollTop = box.scrollHeight;
  }
  // The reference scene: dark hatched card, a labelled node, wires into the panel, a tile and a close button.
  // Returns the slot the panel goes in (slot.root is the whole scene; mount it wherever the page wants).
  function scene(opt) {
    var root = h("div", "stage"), sc = h("div", "scene"), slot = h("div", "slot");
    [[0, 128, 40, 1], [179, 128, 124, 1], [268, 72, 1, 114], [268, 72, 35, 1], [268, 185, 35, 1]].forEach(function (r) {
      var w = h("div", "w"), u = "calc(var(--u)*"; w.style.cssText = "left:" + u + r[0] + ");top:" + u + r[1] + ");width:" + (r[2] === 1 ? "1px" : u + r[2] + ")") + ";height:" + (r[3] === 1 ? "1px" : u + r[3] + ")"); sc.appendChild(w); });
    var node = h("div", "node"); node.append(h("i"), document.createTextNode(window.CU_BRAND || "Live support"));
    var tile = h("div", "tile"); tile.innerHTML = '<svg viewBox="0 0 40 40" aria-hidden="true"><g stroke="#ECEBE6" stroke-width="2.2" fill="none" stroke-linecap="round" stroke-linejoin="round"><path d="M8 8l7 7M8 14V8h6"/><path d="M21 8h9v9"/><path d="M8 22v10h8"/></g><rect x="20" y="22" width="13" height="10" rx="3.5" fill="#E4E3DE"/></svg>';
    var x = h("a", "close"); x.href = opt.closeHref || "/"; x.setAttribute("aria-label", opt.closeLabel || "Close");
    x.innerHTML = '<svg viewBox="0 0 20 20" aria-hidden="true"><path d="M2 2l16 16M18 2L2 18" stroke="#6B6A6D" stroke-width="1.6" fill="none"/></svg>';
    sc.append(tile, node, x, slot); root.appendChild(sc); slot.root = root; return slot;
  }
  function composer(opt) {
    var root = h("form", "compose"), err = h("div", "err"), row = h("div", "row"), ta = h("textarea"), btn = h("button", "btn", "Send"), busy = false;
    err.hidden = true; ta.rows = 1; ta.maxLength = opt.max; ta.placeholder = opt.placeholder; ta.setAttribute("aria-label", opt.placeholder); btn.type = "submit";
    row.append(ta, btn); root.append(err, row);
    function setError(t) { err.textContent = t || ""; err.hidden = !t; }
    ta.addEventListener("input", function () { ta.style.height = "auto"; ta.style.height = Math.min(ta.scrollHeight, 120) + "px"; });
    ta.addEventListener("keydown", function (e) { if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); root.requestSubmit(); } });
    root.addEventListener("submit", function (e) {
      e.preventDefault(); var t = ta.value.trim(); if (!t || busy) return; busy = true; btn.disabled = true; setError("");
      Promise.resolve(opt.onSend(t)).then(function () { ta.value = ""; ta.style.height = "auto"; })
        .catch(function (x) { setError(x.message); }).then(function () { busy = false; btn.disabled = false; ta.focus(); });
    });
    return { root: root, focus: function () { ta.focus(); } };
  }
  window.CU = { scene: scene, avatarFile: avatarFile, ANIMALS: Object.keys(FACES), COLOR_NAMES: Object.keys(COLORS), h: h, avatar: avatar, supportAvatar: supportAvatar, bell: bell, time: time, api: api, renderThread: renderThread, composer: composer };
})();
