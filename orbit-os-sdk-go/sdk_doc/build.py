#!/usr/bin/env python3
"""
Generates a fully static HTML from the JS-rendered source.

  Source (editable):  sdk/api-reference.html   ← edit data here
  Output (deploy):    sdk/api-reference-static.html

Usage:  python3 build.py
"""

import subprocess
import sys
import os
import re
from datetime import datetime

DIR = os.path.dirname(os.path.abspath(__file__))
SRC = os.path.join(DIR, 'api-reference.html')
OUT = os.path.join(DIR, 'api-reference-static.html')

# ── Find Chrome ───────────────────────────────────────────────────────────────

CHROME = next(
    (c for c in ('google-chrome', 'chromium-browser', 'chromium', 'google-chrome-stable')
     if subprocess.run(['which', c], capture_output=True).returncode == 0),
    None
)
if not CHROME:
    sys.exit('Chrome/Chromium not found. Install it and retry.')

# ── Render with headless Chrome ───────────────────────────────────────────────

print(f'Rendering {SRC} with {CHROME}...')
result = subprocess.run(
    [
        CHROME,
        '--headless=new',
        '--disable-gpu',
        '--no-sandbox',
        '--dump-dom',
        f'file://{SRC}',
    ],
    capture_output=True, text=True, timeout=30
)
if result.returncode != 0:
    sys.exit(f'Chrome failed:\n{result.stderr}')

rendered = result.stdout
if not rendered.strip().startswith('<!DOCTYPE'):
    rendered = '<!DOCTYPE html>\n' + rendered

# ── Replace <script> blocks with minimal interactive JS ──────────────────────

INTERACTIVE_JS = """
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

// Hamburger / mobile sidebar
(function() {
  var btn = document.getElementById('hamburger');
  var sb  = document.getElementById('sidebar');
  var ov  = document.getElementById('overlay');
  if (!btn) return;
  btn.addEventListener('click', function() {
    sb.classList.toggle('open');
    ov.classList.toggle('visible');
  });
  ov.addEventListener('click', function() {
    sb.classList.remove('open');
    ov.classList.remove('visible');
  });
  document.getElementById('sidebar-nav').addEventListener('click', function(e) {
    if (e.target.tagName === 'A' && window.innerWidth <= 768) {
      sb.classList.remove('open');
      ov.classList.remove('visible');
    }
  });
})();
""".strip()

# Remove all script blocks from the rendered output, then re-add only interactive JS
rendered = re.sub(r'<script\b[^>]*>[\s\S]*?</script>', '', rendered)
rendered = rendered.replace('</body>', f'<script>\n{INTERACTIVE_JS}\n</script>\n</body>', 1)

# ── Stamp doc version ─────────────────────────────────────────────────────────

doc_version = 'v' + datetime.now().strftime('%y.%m%d.%H%M')
rendered = re.sub(r'v\d{2}\.\d{4}\.\d{4}', doc_version, rendered, count=1)

# ── Write output ──────────────────────────────────────────────────────────────

with open(OUT, 'w', encoding='utf-8') as f:
    f.write(rendered)

print(f'Done → {OUT}  ({len(rendered):,} bytes)')
print('Deploy api-reference-static.html — the source api-reference.html is unchanged.')
