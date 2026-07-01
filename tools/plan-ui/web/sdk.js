/* plan-ui annotation SDK — injected into the artifact iframe.
   Vanilla JS, no dependencies. Provides:
     - layout gate: masks the artifact until an overflow/clip/overlap audit passes
     - inline annotation: click an element or select text to attach a comment
     - queue + send: Enter queues, Ctrl/Cmd+Enter sends all queued feedback
     - data-plan-action buttons: one-click structured actions (approve, etc.)
   All requests are same-origin to the plan-ui server. */
(function () {
  "use strict";

  var CFG = window.__PLAN_UI__ || {};
  var KEY = CFG.key;
  if (!KEY) return;
  var API = "/api/" + KEY;

  var queue = []; // pending prompts not yet sent

  // ---- utilities ------------------------------------------------------------

  function api(path, body) {
    return fetch(API + path, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
  }

  function selectorFor(el) {
    if (!el || el === document.body) return "body";
    var parts = [];
    while (el && el.nodeType === 1 && el !== document.body && parts.length < 6) {
      var part = el.tagName.toLowerCase();
      if (el.id) {
        part += "#" + el.id;
        parts.unshift(part);
        break;
      }
      var parent = el.parentElement;
      if (parent) {
        var sibs = Array.prototype.filter.call(
          parent.children,
          function (c) { return c.tagName === el.tagName; }
        );
        if (sibs.length > 1) {
          part += ":nth-of-type(" + (sibs.indexOf(el) + 1) + ")";
        }
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

  // ---- layout gate ----------------------------------------------------------

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
      // Fail open: never leave the artifact permanently masked on an audit bug.
      return { warnings: [], error: String(e) };
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

  // ---- annotation -----------------------------------------------------------

  var card, lastHover;

  function clearHover() {
    if (lastHover) { lastHover.classList.remove("plan-ui-hover-outline"); lastHover = null; }
  }

  function onMouseOver(e) {
    if (card) return;
    var el = e.target;
    if (!el || isUIElement(el) || isInteractive(el) || el === document.body) return;
    clearHover();
    lastHover = el;
    el.classList.add("plan-ui-hover-outline");
  }

  function closeCard() {
    if (card) { card.remove(); card = null; }
    document
      .querySelectorAll(".plan-ui-selected-outline")
      .forEach(function (n) { n.classList.remove("plan-ui-selected-outline"); });
  }

  function openCard(target, label, anchorRect) {
    closeCard();
    clearHover();
    card = document.createElement("div");
    card.className = "plan-ui-card";
    card.setAttribute("data-plan-ui", "");
    card.innerHTML =
      '<div class="plan-ui-card__head"><span>Annotate ' + escapeHtml(label) + "</span></div>" +
      '<div class="plan-ui-card__target">' + escapeHtml(target.summary) + "</div>" +
      '<div class="plan-ui-card__body"><textarea placeholder="Tell the agent what to change…"></textarea></div>' +
      '<div class="plan-ui-hint">Enter to queue · Ctrl/Cmd+Enter to send all</div>' +
      '<div class="plan-ui-card__foot">' +
      '<button class="plan-ui-btn plan-ui-btn--ghost" data-act="cancel">Cancel</button>' +
      '<button class="plan-ui-btn plan-ui-btn--primary" data-act="queue">Queue</button>' +
      '<button class="plan-ui-btn plan-ui-btn--send" data-act="send">Send all</button>' +
      "</div>";
    document.documentElement.appendChild(card);
    positionCard(card, anchorRect);

    var ta = card.querySelector("textarea");
    ta.focus();
    ta.addEventListener("keydown", function (e) {
      if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        queueFrom(ta, target);
        sendQueue();
      } else if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        queueFrom(ta, target);
        closeCard();
      } else if (e.key === "Escape") {
        closeCard();
      }
    });
    card.addEventListener("click", function (e) {
      var act = e.target.getAttribute("data-act");
      if (act === "cancel") closeCard();
      else if (act === "queue") { queueFrom(ta, target); closeCard(); }
      else if (act === "send") { queueFrom(ta, target); sendQueue(); }
    });
  }

  function queueFrom(ta, target) {
    var text = ta.value.trim();
    if (!text) return;
    queue.push({
      text: text,
      action: "comment",
      target: {
        selector: target.selector,
        quoted_text: target.quotedText || "",
        start: target.start || 0,
        end: target.end || 0,
      },
      ts: new Date().toISOString(),
    });
    renderQueue();
  }

  function sendQueue() {
    closeCard();
    if (queue.length === 0) return;
    var prompts = queue.slice();
    queue = [];
    renderQueue();
    api("/feedback", { prompts: prompts }).catch(function () {});
  }

  var queueEl;
  function renderQueue() {
    if (!queueEl) {
      queueEl = document.createElement("div");
      queueEl.className = "plan-ui-queue";
      queueEl.setAttribute("data-plan-ui", "");
      document.documentElement.appendChild(queueEl);
    }
    if (queue.length === 0) { queueEl.innerHTML = ""; return; }
    queueEl.innerHTML =
      '<div class="plan-ui-queue__count" data-act="send">' +
      queue.length + " queued · click to send</div>";
    queueEl.querySelector('[data-act="send"]').onclick = sendQueue;
  }

  function onClick(e) {
    var el = e.target;
    if (isUIElement(el)) return;
    var actionEl = el.closest && el.closest("[data-plan-action]");
    if (actionEl) {
      e.preventDefault();
      var action = actionEl.getAttribute("data-plan-action");
      api("/feedback", {
        prompts: [{
          text: actionEl.getAttribute("data-plan-label") || action,
          action: action,
          target: { selector: selectorFor(actionEl) },
          ts: new Date().toISOString(),
        }],
      }).catch(function () {});
      flash(actionEl);
      return;
    }
    if (isInteractive(el) || el === document.body) return;

    var sel = window.getSelection();
    var selectedText = sel && sel.toString().trim();
    if (selectedText) {
      var range = sel.getRangeAt(0);
      var rect = range.getBoundingClientRect();
      openCard(
        {
          selector: selectorFor(range.startContainer.parentElement || el),
          quotedText: selectedText.slice(0, 400),
          summary: '"' + selectedText.slice(0, 60) + (selectedText.length > 60 ? "…" : "") + '"',
        },
        "text",
        rect
      );
      return;
    }
    e.preventDefault();
    el.classList.add("plan-ui-selected-outline");
    openCard(
      { selector: selectorFor(el), summary: "<" + el.tagName.toLowerCase() + "> element" },
      "<" + el.tagName.toLowerCase() + ">",
      el.getBoundingClientRect()
    );
  }

  // ---- helpers --------------------------------------------------------------

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

  function flash(el) {
    el.classList.add("plan-ui-selected-outline");
    setTimeout(function () { el.classList.remove("plan-ui-selected-outline"); }, 400);
  }

  function escapeHtml(s) {
    return String(s).replace(/[&<>"]/g, function (c) {
      return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c];
    });
  }

  // ---- boot -----------------------------------------------------------------

  function boot() {
    document.addEventListener("mouseover", onMouseOver, true);
    document.addEventListener("mouseout", function () { if (!card) clearHover(); }, true);
    document.addEventListener("click", onClick, true);
    renderQueue();
    runGate();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot);
  } else {
    boot();
  }
})();
