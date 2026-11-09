/* Shows a green dot on the Human support button when a reply has arrived since the visitor last looked. */
(function () {
  "use strict";
  var dot = document.getElementById("hs-dot"), seen;
  if (!dot) return;
  try { seen = localStorage.getItem("hs_seen"); } catch (e) { return; }
  fetch("/api/human/messages", { credentials: "same-origin", headers: { "X-Requested-With": "support" } })
    .then(function (r) { return r.ok ? r.json() : null; })
    .then(function (d) {
      if (!d || !d.messages) return;
      var op = d.messages.filter(function (m) { return m.sender === "operator"; }).pop();
      if (op && String(op.id) !== seen) dot.hidden = false;
    }).catch(function () {});
})();
