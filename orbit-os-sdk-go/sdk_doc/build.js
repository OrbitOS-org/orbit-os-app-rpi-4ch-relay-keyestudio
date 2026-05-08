#!/usr/bin/env node
// Generates a fully static api-reference.html from the JS-rendered source.
// Usage: node build.js
// Output: api-reference.html (overwrites in place)

const fs   = require('fs');
const path = require('path');
const { JSDOM } = require('jsdom');

const SRC = path.join(__dirname, 'api-reference.html');
const src = fs.readFileSync(SRC, 'utf8');

// Execute the page JS to populate #content and #sidebar-nav
const dom = new JSDOM(src, { runScripts: 'dangerously' });
const doc = dom.window.document;

const sidebarHtml = doc.getElementById('sidebar-nav').innerHTML;
const contentHtml = doc.getElementById('content').innerHTML;
const versionHtml = doc.getElementById('version-select').innerHTML;

// Extract everything between <style> tags (CSS)
const cssMatch = src.match(/<style>([\s\S]*?)<\/style>/);
const css = cssMatch ? cssMatch[1] : '';

// Extract <head> meta/link tags (favicon, etc.) but not style/script
const headContent = src
  .replace(/<style>[\s\S]*?<\/style>/, '')
  .replace(/<script>[\s\S]*?<\/script>/, '')
  .match(/<head>([\s\S]*?)<\/head>/)?.[1]
  ?.trim() || '';

// Minimal interactive JS — no data, no rendering, only UI behaviour
const interactiveJs = `
// Sidebar toggle
document.querySelectorAll('.sidebar-section > a').forEach(function(a) {
  a.addEventListener('click', function(e) {
    e.preventDefault();
    var section = a.parentElement;
    var methods = section.nextElementSibling;
    var isOpen  = methods.style.display !== 'none';
    methods.style.display = isOpen ? 'none' : 'block';
    section.classList.toggle('open', !isOpen);
    var id = a.getAttribute('href').slice(1);
    var target = document.getElementById(id);
    if (target) target.scrollIntoView({ behavior: 'instant', block: 'start' });
  });
});

// Scroll-sync sidebar active link
function syncSidebar() {
  var links = Array.from(document.querySelectorAll('#sidebar-nav a[data-id]'));
  var active = null;
  links.forEach(function(a) {
    if (!a.offsetParent) return;
    var el = document.getElementById(a.dataset.id);
    if (el && el.getBoundingClientRect().top <= 72) active = a;
  });
  links.forEach(function(a) { a.classList.remove('active'); });
  if (active) active.classList.add('active');
}

// Language switch
var lang = 'go';
function setLang(l) {
  lang = l;
  document.querySelectorAll('.lang-btn').forEach(function(b) {
    b.classList.toggle('active', b.dataset.lang === l);
  });
  document.querySelectorAll('.code-block, .sig-block').forEach(function(b) {
    b.classList.toggle('visible', b.dataset.lang === l);
  });
}
document.querySelectorAll('.lang-btn').forEach(function(b) {
  b.addEventListener('click', function() { setLang(b.dataset.lang); });
});

document.getElementById('main').addEventListener('scroll', syncSidebar, { passive: true });
window.addEventListener('scroll', syncSidebar, { passive: true });
`.trim();

const out = `<!DOCTYPE html>
<html lang="en">
<head>
${headContent}
<style>${css}</style>
</head>
<body>
<div id="sidebar">
  <div class="sidebar-logo-wrap">
    <img src="logo.png" alt="Orbit OS" class="sidebar-logo">
    <div class="sidebar-version">SDK API 26 Reference</div>
    <div class="sidebar-version-select"><select id="version-select">${versionHtml}</select></div>
  </div>
  <nav id="sidebar-nav">${sidebarHtml}</nav>
</div>

<div id="main">
  <div id="under-construction">&#9888; Under Construction — documentation is incomplete and subject to change</div>
  <div id="lang-bar">
    <span class="label">Language</span>
    <button class="lang-btn active" data-lang="go">Go</button>
  </div>
  <div id="content">${contentHtml}</div>
</div>

<script>${interactiveJs}</script>
</body>
</html>`;

fs.writeFileSync(SRC, out, 'utf8');
console.log('Built: ' + SRC);
