// Small behaviours shared by every page: fill in configured links, footer year, landing scroll cue.
(() => {
  "use strict";
  const cfg = window.SITE_CONFIG || {};
  const https = v => typeof v === "string" && /^https:\/\/[^\s"'<>]+$/.test(v.trim());
  const email = v => typeof v === "string" && /^[^@\s<>"',;]{1,64}@[A-Za-z0-9.-]{1,190}\.[A-Za-z]{2,}$/.test(v.trim());
  const all = sel => document.querySelectorAll(sel);

  if (https(cfg.inviteUrl)) {
    all("[data-invite]").forEach(a => { a.href = cfg.inviteUrl.trim(); a.rel = "noopener"; a.textContent = "Add to Discord"; });
    all("[data-invite-only]").forEach(a => { a.href = cfg.inviteUrl.trim(); a.hidden = false; });
  }
  const hasUrl = https(cfg.supportUrl), hasMail = email(cfg.supportEmail);
  if (hasUrl) all("[data-support-url]").forEach(a => { a.href = cfg.supportUrl.trim(); a.hidden = false; });
  if (hasMail) all("[data-support-email]").forEach(a => { a.href = "mailto:" + cfg.supportEmail.trim(); a.textContent = cfg.supportEmail.trim(); a.hidden = false; });
  all("[data-contact-or]").forEach(s => { s.hidden = !(hasUrl && hasMail); });
  all("[data-contact-none]").forEach(s => { s.hidden = hasUrl || hasMail; });
  all("[data-year]").forEach(s => { s.textContent = String(new Date().getFullYear()); });

  const cue = document.getElementById("scroll-cue");
  if (cue) {
    const sync = () => cue.classList.toggle("gone", window.scrollY > 60);
    window.addEventListener("scroll", sync, {passive: true});
    sync();
  }
})();
