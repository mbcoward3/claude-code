/* plan-ui annotation SDK — injected into the plan document.
   Vanilla JS, no dependencies. The document IS the whole UI:
     - layout gate: masks the plan until an overflow/clip audit passes
     - inline annotation: click an element or select text to attach a comment
     - persistent markers: every queued annotation is visibly numbered in place;
       click a marker to edit or delete it before sending
     - floating toolbar: queued count, Send annotations / Approve plan, and a
       status line (sent → agent working → plan updated)
   Two modes (window.__PLAN_UI__.mode):
     - "artifact": Send posts feedback to the polling agent; the loop continues.
     - "plan" (ExitPlanMode hook): Send = Request changes (deny + compiled
       feedback) and Approve = allow; both are terminal.
   All requests are same-origin to the local plan-ui server. */
(function () {
  "use strict";

  var CFG = window.__PLAN_UI__ || {};
  var KEY = CFG.key;
  if (!KEY) return;
  var MODE = CFG.mode === "plan" ? "plan" : "artifact";
  var API = "/api/" + KEY;
  var STORE_KEY = "plan-ui:" + KEY;

  // queue items: { id, text, action, target: {selector, quoted_text}, ts }
  var queue = [];
  var nextId = 1;
  var terminal = false; // plan mode: decision made

  // ---- utilities ------------------------------------------------------------

  function api(path, body) {
    return fetch(API + path, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
  }

  function escapeHtml(s) {
    return String(s).replace(/[&<>"]/g, function (c) {
      return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c];
    });
  }

  function selectorFor(el) {
    if (!el || el === document.body) return "body";
    var parts = [];
    while (el && el.nodeType === 1 && el !== document.body && parts.length < 6) {
      var part = el.tagName.toLowerCase();
      if (el.id) {
        parts.unshift(part + "#" + el.id);
        break;
      }
      var parent = el.parentElement;
      if (parent) {
        var sibs = Array.prototype.filter.call(parent.children, function (c) {
          return c.tagName === el.tagName;
        });
        if (sibs.length > 1) part += ":nth-of-type(" + (sibs.indexOf(el) + 1) + ")";
      }
      parts.unshift(part);
      el = el.parentElement;
    }
    return parts.join(" > ");
  }

  function isUIElement(el) {
    return !!(el.closest && el.closest("[data-plan-ui]"));
  }

  function isInteractive(el) {
    return /^(a|button|input|select|textarea|label|option)$/i.test(el.tagName) ||
      el.hasAttribute("data-plan-action");
  }

  // ---- persistence (queue survives live reloads) ------------------------------

  function saveQueue() {
    try {
      var items = queue.map(function (i) {
        return { id: i.id, text: i.text, action: i.action, target: i.target, ts: i.ts };
      });
      sessionStorage.setItem(STORE_KEY, JSON.stringify({ queue: items, nextId: nextId }));
    } catch (e) {}
  }

  function loadQueue() {
    try {
      var raw = sessionStorage.getItem(STORE_KEY);
      if (!raw) return;
      var data = JSON.parse(raw);
      queue = data.queue || [];
      nextId = data.nextId || queue.length + 1;
    } catch (e) { queue = []; }
  }

  function setFlag(name) {
    try { sessionStorage.setItem(STORE_KEY + ":" + name, "1"); } catch (e) {}
  }
  function takeFlag(name) {
    try {
      var v = sessionStorage.getItem(STORE_KEY + ":" + name);
      sessionStorage.removeItem(STORE_KEY + ":" + name);
      return v === "1";
    } catch (e) { return false; }
  }

  // ---- markers ----------------------------------------------------------------

  var overlay; // fixed layer holding badges for element annotations

  function ensureOverlay() {
    if (overlay) return overlay;
    overlay = document.createElement("div");
    overlay.className = "plan-ui-overlay";
    overlay.setAttribute("data-plan-ui", "");
    document.documentElement.appendChild(overlay);
    window.addEventListener("scroll", repositionBadges, { passive: true });
    window.addEventListener("resize", repositionBadges, { passive: true });
    return overlay;
  }

  function markItem(item) {
    // Text annotation: wrap the exact range when we still have it live.
    if (item._range) {
      try {
        var mark = makeMark(item);
        item._range.surroundContents(mark);
        item._mark = mark;
        delete item._range;
        return;
      } catch (e) { delete item._range; /* range crossed elements; fall through */ }
    }
    // Re-marking after reload: find the quoted text inside its element.
    if (item.target && item.target.quoted_text && !item._mark && !item._el) {
      var host = safeQuery(item.target.selector) || document.body;
      if (wrapTextIn(host, item)) return;
    }
    // Element annotation (or text fallback): outline + overlay badge.
    var el = item._el || safeQuery(item.target && item.target.selector);
    if (!el) return;
    item._el = el;
    el.classList.add("plan-ui-annotated");
    var badge = document.createElement("button");
    badge.className = "plan-ui-badge";
    badge.setAttribute("data-plan-ui", "");
    badge.textContent = item.id;
    badge.title = item.text;
    badge.addEventListener("click", function (e) {
      e.preventDefault(); e.stopPropagation();
      openCard({ edit: item });
    });
    ensureOverlay().appendChild(badge);
    item._badge = badge;
    positionBadge(item);
  }

  function makeMark(item) {
    var mark = document.createElement("mark");
    mark.className = "plan-ui-mark";
    mark.setAttribute("data-plan-ui", "");
    mark.setAttribute("data-n", item.id);
    mark.addEventListener("click", function (e) {
      e.preventDefault(); e.stopPropagation();
      openCard({ edit: item });
    });
    return mark;
  }

  function safeQuery(sel) {
    if (!sel) return null;
    try { return document.querySelector(sel); } catch (e) { return null; }
  }

  function wrapTextIn(host, item) {
    var text = item.target.quoted_text;
    var walker = document.createTreeWalker(host, NodeFilter.SHOW_TEXT, null);
    var node;
    while ((node = walker.nextNode())) {
      if (node.parentElement && node.parentElement.closest("[data-plan-ui]")) continue;
      var idx = node.nodeValue.indexOf(text);
      if (idx >= 0) {
        try {
          var range = document.createRange();
          range.setStart(node, idx);
          range.setEnd(node, idx + text.length);
          var mark = makeMark(item);
          range.surroundContents(mark);
          item._mark = mark;
          return true;
        } catch (e) { return false; }
      }
    }
    return false;
  }

  function positionBadge(item) {
    if (!item._badge || !item._el) return;
    var r = item._el.getBoundingClientRect();
    item._badge.style.top = Math.max(2, r.top - 10) + "px";
    item._badge.style.left = Math.min(window.innerWidth - 26, r.right - 10) + "px";
  }

  function repositionBadges() {
    queue.forEach(positionBadge);
  }

  function unmarkItem(item) {
    if (item._mark && item._mark.parentNode) {
      var m = item._mark, parent = m.parentNode;
      while (m.firstChild) parent.insertBefore(m.firstChild, m);
      parent.removeChild(m);
      parent.normalize();
    }
    if (item._el) item._el.classList.remove("plan-ui-annotated");
    if (item._badge && item._badge.parentNode) item._badge.parentNode.removeChild(item._badge);
    delete item._mark; delete item._el; delete item._badge;
  }

  function setMarkersSent() {
    queue.forEach(function (item) {
      if (item._mark) item._mark.classList.add("plan-ui-mark--sent");
      if (item._el) item._el.classList.remove("plan-ui-annotated");
      if (item._badge) item._badge.classList.add("plan-ui-badge--sent");
    });
  }

  // ---- annotation card --------------------------------------------------------

  var card, lastHover;

  function clearHover() {
    if (lastHover) { lastHover.classList.remove("plan-ui-hover"); lastHover = null; }
  }

  function closeCard() {
    if (card) { card.remove(); card = null; }
    document.querySelectorAll(".plan-ui-selected").forEach(function (n) {
      n.classList.remove("plan-ui-selected");
    });
  }

  // openCard({ create: {selector, quotedText, summary, label, range, el}, anchorRect })
  // openCard({ edit: item })
  function openCard(opts) {
    if (terminal) return;
    closeCard();
    clearHover();
    var editing = opts.edit || null;
    var label = editing ? "Edit annotation #" + editing.id : "Annotate " + opts.create.label;
    var summary = editing
      ? (editing.target && (editing.target.quoted_text || editing.target.selector)) || ""
      : opts.create.summary;

    card = document.createElement("div");
    card.className = "plan-ui-card";
    card.setAttribute("data-plan-ui", "");
    card.innerHTML =
      '<div class="plan-ui-card__head"><span>' + escapeHtml(label) + "</span></div>" +
      '<div class="plan-ui-card__target">' + escapeHtml(summary) + "</div>" +
      '<div class="plan-ui-card__body"><textarea placeholder="Tell the agent what to change…"></textarea></div>' +
      '<div class="plan-ui-card__foot">' +
      (editing
        ? '<button class="plan-ui-btn plan-ui-btn--danger" data-act="delete">Delete</button>' +
          '<span class="plan-ui-card__spacer"></span>' +
          '<button class="plan-ui-btn plan-ui-btn--ghost" data-act="cancel">Cancel</button>' +
          '<button class="plan-ui-btn plan-ui-btn--primary" data-act="save">Save</button>'
        : '<button class="plan-ui-btn plan-ui-btn--ghost" data-act="cancel">Cancel</button>' +
          '<button class="plan-ui-btn plan-ui-btn--primary" data-act="add">Add annotation</button>') +
      "</div>";
    document.documentElement.appendChild(card);
    positionCard(card, editing ? rectFor(editing) : opts.anchorRect);

    var ta = card.querySelector("textarea");
    if (editing) ta.value = editing.text;
    ta.focus();

    function commit() {
      var text = ta.value.trim();
      if (editing) {
        if (!text) return removeItem(editing);
        editing.text = text;
        if (editing._badge) editing._badge.title = text;
        saveQueue();
        renderToolbar();
        closeCard();
      } else {
        if (!text) { closeCard(); return; }
        addItem(text, opts.create);
        closeCard();
      }
    }

    ta.addEventListener("keydown", function (e) {
      if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); commit(); }
      else if (e.key === "Escape") closeCard();
    });
    card.addEventListener("click", function (e) {
      var act = e.target.getAttribute("data-act");
      if (act === "cancel") closeCard();
      else if (act === "add" || act === "save") commit();
      else if (act === "delete" && editing) removeItem(editing);
    });
  }

  function rectFor(item) {
    var el = item._mark || item._el;
    return el ? el.getBoundingClientRect() : null;
  }

  function positionCard(el, rect) {
    var top = (rect ? rect.bottom : 40) + 8;
    var left = rect ? rect.left : 20;
    var vw = window.innerWidth, vh = window.innerHeight;
    if (left + 320 > vw - 12) left = vw - 332;
    if (left < 12) left = 12;
    if (top + 220 > vh - 12) top = Math.max(12, (rect ? rect.top : 40) - 228);
    el.style.top = top + "px";
    el.style.left = left + "px";
  }

  // ---- queue management --------------------------------------------------------

  function addItem(text, create) {
    var item = {
      id: nextId++,
      text: text,
      action: "comment",
      target: {
        selector: create.selector,
        quoted_text: create.quotedText || "",
      },
      ts: new Date().toISOString(),
      _range: create.range || null,
      _el: create.el || null,
    };
    queue.push(item);
    markItem(item);
    saveQueue();
    renderToolbar();
  }

  function removeItem(item) {
    unmarkItem(item);
    var i = queue.indexOf(item);
    if (i >= 0) queue.splice(i, 1);
    saveQueue();
    renderToolbar();
    closeCard();
  }

  function serializeQueue() {
    return queue.map(function (i) {
      return { text: i.text, action: i.action, target: i.target, ts: i.ts };
    });
  }

  function compileFeedback() {
    return queue.map(function (a) {
      var loc = a.target && (a.target.quoted_text || a.target.selector);
      return loc ? "- (re: " + loc + ") " + a.text : "- " + a.text;
    }).join("\n");
  }

  // ---- toolbar -------------------------------------------------------------------

  var toolbar, statusEl, agentEl, sendBtn, approveBtn;
  var sentState = ""; // "" | "sent" | "working"
  var sentItems = []; // markers kept visible (greyed) after send, until reload

  function buildToolbar() {
    toolbar = document.createElement("div");
    toolbar.className = "plan-ui-toolbar";
    toolbar.setAttribute("data-plan-ui", "");
    toolbar.innerHTML =
      '<div class="plan-ui-toolbar__agent" hidden></div>' +
      '<div class="plan-ui-toolbar__row">' +
      '<span class="plan-ui-toolbar__status"></span>' +
      '<button class="plan-ui-btn plan-ui-btn--primary" data-act="send"></button>' +
      '<button class="plan-ui-btn plan-ui-btn--approve" data-act="approve">Approve plan</button>' +
      "</div>";
    document.documentElement.appendChild(toolbar);
    statusEl = toolbar.querySelector(".plan-ui-toolbar__status");
    agentEl = toolbar.querySelector(".plan-ui-toolbar__agent");
    sendBtn = toolbar.querySelector('[data-act="send"]');
    approveBtn = toolbar.querySelector('[data-act="approve"]');
    sendBtn.addEventListener("click", onSend);
    approveBtn.addEventListener("click", onApprove);
    renderToolbar();
  }

  function renderToolbar() {
    if (!toolbar) return;
    var n = queue.length;
    sendBtn.textContent = (MODE === "plan" ? "Request changes" : "Send annotations") +
      (n ? " (" + n + ")" : "");
    sendBtn.disabled = terminal || n === 0;
    approveBtn.disabled = terminal;
    if (terminal) return;
    if (sentState === "working") {
      setStatus("Agent is working on your feedback…", "busy");
    } else if (sentState === "sent") {
      setStatus("Feedback sent — waiting for the agent…", "busy");
    } else if (n > 0) {
      setStatus(n + " annotation" + (n > 1 ? "s" : "") + " queued — click a marker to edit", "");
    } else {
      setStatus("Click any element or select text to annotate", "");
    }
  }

  function setStatus(text, kind) {
    statusEl.textContent = text;
    statusEl.className = "plan-ui-toolbar__status" +
      (kind ? " plan-ui-toolbar__status--" + kind : "");
  }

  function showAgentMessage(text) {
    agentEl.hidden = false;
    agentEl.textContent = "Agent: " + text;
  }

  function finalize(text, kind) {
    terminal = true;
    closeCard();
    queue.forEach(unmarkItem);
    queue = [];
    saveQueue();
    sendBtn.disabled = true;
    approveBtn.disabled = true;
    setStatus(text, kind);
  }

  function onSend() {
    if (queue.length === 0 || terminal) return;
    closeCard();
    if (MODE === "plan") {
      api("/decision", { decision: "deny", feedback: compileFeedback() });
      finalize("Changes requested — the agent is revising the plan. You can close this tab.", "done");
    } else {
      api("/feedback", { prompts: serializeQueue() });
      setMarkersSent();
      sentItems = queue;
      queue = [];
      saveQueue();
      setFlag("pending");
      sentState = "sent";
      renderToolbar();
    }
  }

  function onApprove() {
    if (terminal) return;
    if (MODE === "plan") {
      api("/decision", { decision: "approve", feedback: "" });
      finalize("Plan approved — you can close this tab.", "done");
    } else {
      api("/feedback", { prompts: [{ text: "Approved.", action: "approve", ts: new Date().toISOString() }] });
      finalize("Approval sent to the agent.", "done");
    }
  }

  // ---- layout gate ------------------------------------------------------------------

  function auditLayout() {
    var warnings = [];
    try {
      var docW = document.documentElement.clientWidth;
      if (document.documentElement.scrollWidth > docW + 1) {
        warnings.push({
          type: "overflow",
          selector: "html",
          detail: "page content is " + document.documentElement.scrollWidth +
            "px wide but the viewport is " + docW + "px (horizontal overflow)",
        });
      }
      var els = document.body ? document.body.querySelectorAll("*") : [];
      var cap = Math.min(els.length, 4000);
      for (var i = 0; i < cap; i++) {
        var el = els[i];
        if (isUIElement(el)) continue;
        var style = getComputedStyle(el);
        var ox = style.overflowX;
        if ((ox === "hidden" || ox === "clip") && el.scrollWidth > el.clientWidth + 2 && el.clientWidth > 0) {
          warnings.push({
            type: "clipped",
            selector: selectorFor(el),
            detail: "content is clipped horizontally (scrollWidth " + el.scrollWidth +
              " > clientWidth " + el.clientWidth + ")",
          });
          if (warnings.length > 40) break;
        }
      }
    } catch (e) {
      return { warnings: [] }; // fail open: never leave the plan permanently masked
    }
    return { warnings: warnings.slice(0, 40) };
  }

  var mask;
  function showMask(state, title, detail) {
    if (!mask) {
      mask = document.createElement("div");
      mask.className = "plan-ui-mask";
      mask.setAttribute("data-plan-ui", "");
      mask.innerHTML =
        '<div class="plan-ui-mask__spinner"></div>' +
        '<div class="plan-ui-mask__title"></div>' +
        '<div class="plan-ui-mask__detail"></div>';
      document.documentElement.appendChild(mask);
    }
    mask.className = "plan-ui-mask" + (state === "error" ? " plan-ui-mask--error" : "");
    mask.querySelector(".plan-ui-mask__title").textContent = title || "";
    mask.querySelector(".plan-ui-mask__detail").textContent = detail || "";
    mask.style.display = "flex";
  }
  function hideMask() { if (mask) mask.style.display = "none"; }

  function runGate() {
    showMask("checking", "Checking layout…", "Making sure the plan renders cleanly before review.");
    // Let layout settle (fonts, late styles) before measuring.
    setTimeout(function () {
      var result = auditLayout();
      api("/gate", { warnings: result.warnings }).catch(function () {});
      if (result.warnings.length === 0) {
        hideMask();
      } else {
        showMask(
          "error",
          "Layout needs fixing",
          result.warnings.length + " layout issue(s) found. The agent has been notified and will revise the plan."
        );
      }
    }, 350);
  }

  // ---- document event handlers ---------------------------------------------------------

  function onMouseOver(e) {
    if (card || terminal) return;
    var el = e.target;
    if (!el || isUIElement(el) || isInteractive(el) || el === document.body) return;
    clearHover();
    lastHover = el;
    el.classList.add("plan-ui-hover");
  }

  function onClick(e) {
    if (terminal) return;
    var el = e.target;
    if (isUIElement(el)) return;
    var actionEl = el.closest && el.closest("[data-plan-action]");
    if (actionEl) {
      e.preventDefault();
      api("/feedback", {
        prompts: [{
          text: actionEl.getAttribute("data-plan-label") || actionEl.getAttribute("data-plan-action"),
          action: actionEl.getAttribute("data-plan-action"),
          target: { selector: selectorFor(actionEl) },
          ts: new Date().toISOString(),
        }],
      }).catch(function () {});
      actionEl.classList.add("plan-ui-selected");
      setTimeout(function () { actionEl.classList.remove("plan-ui-selected"); }, 400);
      return;
    }
    if (isInteractive(el) || el === document.body) return;

    var sel = window.getSelection();
    var selectedText = sel && sel.toString().trim();
    if (selectedText) {
      var range = sel.getRangeAt(0).cloneRange();
      openCard({
        create: {
          selector: selectorFor(range.startContainer.parentElement || el),
          quotedText: selectedText.slice(0, 400),
          summary: '"' + selectedText.slice(0, 60) + (selectedText.length > 60 ? "…" : "") + '"',
          label: "text",
          range: range,
        },
        anchorRect: range.getBoundingClientRect(),
      });
      return;
    }
    e.preventDefault();
    el.classList.add("plan-ui-selected");
    openCard({
      create: {
        selector: selectorFor(el),
        summary: "<" + el.tagName.toLowerCase() + "> element",
        label: "<" + el.tagName.toLowerCase() + ">",
        el: el,
      },
      anchorRect: el.getBoundingClientRect(),
    });
  }

  // ---- live updates -----------------------------------------------------------------

  function connectSSE() {
    var es = new EventSource("/events/" + KEY);
    es.addEventListener("reload", function () {
      saveQueue();
      location.reload();
    });
    es.addEventListener("presence", function (e) {
      if (MODE === "plan" || terminal) return;
      if (e.data === "working" && sentState === "sent") {
        sentState = "working";
        renderToolbar();
      }
    });
    es.addEventListener("agent-reply", function (e) {
      showAgentMessage(e.data);
    });
    es.addEventListener("ended", function () {
      finalize("Session ended by the agent.", "done");
    });
  }

  // ---- boot ---------------------------------------------------------------------------

  function boot() {
    loadQueue();
    buildToolbar();
    queue.forEach(markItem);
    if (takeFlag("pending")) {
      setStatus("Plan updated — review the changes", "done");
    }
    document.addEventListener("mouseover", onMouseOver, true);
    document.addEventListener("mouseout", function () { if (!card) clearHover(); }, true);
    document.addEventListener("click", onClick, true);
    connectSSE();
    runGate();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot);
  } else {
    boot();
  }
})();
