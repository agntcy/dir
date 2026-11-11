/* Copyright AGNTCY Contributors (https://github.com/agntcy) */
/* SPDX-License-Identifier: Apache-2.0 */

/* Interactive "choose your setup" wizard for the Configurations page.
   Only two things decide the configuration: how far your records need to be
   discoverable, and (when they don't) whether others may pull them. So this is
   a 1-2 step conditional flow that maps 1:1 to the four configurations and
   deep-links to each section on the same page. Progressive enhancement:
   replaces the static fallback list inside #setup-chooser. */
(function () {
  var SECTION_IDS = ["private-node", "public-store-node", "networked-node", "federated"];
  var activeWrap = null;
  var activeTimer = null;

  function getSectionNodes(heading) {
    var nodes = [heading];
    var el = heading.nextElementSibling;
    while (el && el.tagName !== "H3") {
      nodes.push(el);
      el = el.nextElementSibling;
    }
    return nodes;
  }

  function unwrapSection(wrap) {
    if (!wrap || !wrap.parentNode) {
      return;
    }
    var parent = wrap.parentNode;
    while (wrap.firstChild) {
      parent.insertBefore(wrap.firstChild, wrap);
    }
    wrap.remove();
  }

  function clearSectionHighlight() {
    if (activeTimer) {
      clearTimeout(activeTimer);
      activeTimer = null;
    }
    if (activeWrap) {
      unwrapSection(activeWrap);
      activeWrap = null;
    }
  }

  function highlightSection(id) {
    clearSectionHighlight();
    var heading = document.getElementById(id);
    if (!heading) {
      return;
    }

    var nodes = getSectionNodes(heading);
    var wrap = document.createElement("div");
    wrap.className = "sc-section-target sc-section-flash";
    heading.parentNode.insertBefore(wrap, heading);
    nodes.forEach(function (node) {
      wrap.appendChild(node);
    });
    activeWrap = wrap;

    wrap.scrollIntoView({ behavior: "smooth", block: "start" });

    activeTimer = setTimeout(function () {
      wrap.classList.remove("sc-section-flash");
      activeTimer = null;
    }, 2600);
  }

  function flashFromHash() {
    var id = location.hash.slice(1);
    if (SECTION_IDS.indexOf(id) === -1) {
      return;
    }
    requestAnimationFrame(function () {
      highlightSection(id);
    });
  }

  if (!window.__dirSetupSectionFlash) {
    window.__dirSetupSectionFlash = true;
    window.addEventListener("hashchange", flashFromHash);
  }

  document$.subscribe(function () {
    flashFromHash();

    var mount = document.getElementById("setup-chooser");
    if (!mount) {
      return;
    }

  var DISCOVER = {
    key: "discover",
    title: "Who needs to discover your records?",
    hint: "Discovery = others finding out what records you have, and you discovering what others have, over the network.",
    options: [
      { v: "onlyme", label: "Only me", sub: "No one else needs to find them." },
      { v: "team", label: "My team or organization", sub: "Nodes and teammates connected to a shared DHT." },
      { v: "orgs", label: "Other organizations", sub: "Cross-organization discovery and exchange." },
    ],
  };
  var STORE = {
    key: "store",
    title: "Should others be able to pull your records?",
    hint: "Directly from a public registry (GHCR / Docker Hub), by CID — no network needed.",
    options: [
      { v: "local", label: "No — keep them local", sub: "Bundled local store; not network-reachable." },
      { v: "public", label: "Yes — via a public registry", sub: "Anyone with the registry can pull them." },
    ],
  };

  var CONFIGS = {
    privatenode: { n: 1, name: "Private node", href: "#private-node", why: "Records stay on your machine — no one else can discover or retrieve them." },
    publicstore: { n: 2, name: "Public-store node", href: "#public-store-node", why: "A public registry lets anyone pull your records by CID — without a network." },
    networked: { n: 3, name: "Networked node", href: "#networked-node", why: "On the DHT you announce your records and can discover others across the network." },
    federated: { n: 4, name: "Federated", href: "#federated", why: "Peered nodes exchange records across organizations under a shared trust root." },
  };

  var state = {};
  var screen = "discover"; // discover | store | result

  function pick(s) {
    if (s.discover === "orgs") return "federated";
    if (s.discover === "team") return "networked";
    return s.store === "public" ? "publicstore" : "privatenode";
  }

  function esc(s) {
    return String(s).replace(/[&<>"]/g, function (c) {
      return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c];
    });
  }

  function optionsHTML(q) {
    var chosen = state[q.key];
    return q.options
      .map(function (o) {
        var on = chosen === o.v;
        return (
          '<button type="button" class="sc-opt" role="radio" aria-checked="' + on + '" data-v="' + esc(o.v) + '">' +
          '<span class="sc-radio" aria-hidden="true"></span>' +
          '<span class="sc-otext"><span class="sc-olabel">' + esc(o.label) + "</span>" +
          '<span class="sc-osub">' + esc(o.sub) + "</span></span>" +
          "</button>"
        );
      })
      .join("");
  }

  function questionHTML(q, count, nextLabel, canBack) {
    return (
      '<div class="sc-card">' +
      '<div class="sc-count">' + esc(count) + "</div>" +
      '<div class="sc-title" id="sc-title-' + esc(q.key) + '">' + esc(q.title) + "</div>" +
      '<p class="sc-hint">' + esc(q.hint) + "</p>" +
      '<div class="sc-options" role="radiogroup" aria-labelledby="sc-title-' + esc(q.key) + '">' + optionsHTML(q) + "</div>" +
      '<div class="sc-nav">' +
      '<button type="button" class="sc-btn sc-ghost" data-act="back"' + (canBack ? "" : " disabled") + ">← Back</button>" +
      '<button type="button" class="sc-btn sc-primary" data-act="next"' + (state[q.key] ? "" : " disabled") + ">" + esc(nextLabel) + "</button>" +
      "</div></div>"
    );
  }

  function render() {
    if (screen === "discover") {
      var toResult = state.discover && state.discover !== "onlyme";
      mount.innerHTML = questionHTML(DISCOVER, "Question 1", toResult ? "See recommendation" : "Next", false);
      return;
    }
    if (screen === "store") {
      mount.innerHTML = questionHTML(STORE, "Question 2", "See recommendation", true);
      return;
    }
    // result — just the recommendation and two actions
    var c = CONFIGS[pick(state)];
    mount.innerHTML =
      '<div class="sc-card sc-result">' +
      '<div class="sc-badge">Recommended</div>' +
      '<div class="sc-rec-name">' + c.n + ". " + esc(c.name) + "</div>" +
      '<p class="sc-why">' + esc(c.why) + "</p>" +
      '<div class="sc-cta">' +
      '<a class="sc-btn sc-primary sc-jump" href="' + c.href + '">Jump to ' + esc(c.name) + " ↓</a>" +
      '<button type="button" class="sc-btn sc-ghost" data-act="restart">↺ Start over</button>' +
      "</div></div>";
  }

  mount.addEventListener("click", function (e) {
    var jump = e.target.closest ? e.target.closest(".sc-jump") : null;
    if (jump && mount.contains(jump)) {
      e.preventDefault();
      var href = jump.getAttribute("href") || "";
      if (href.charAt(0) === "#") {
        history.pushState(null, "", href);
        highlightSection(href.slice(1));
      }
      jump.blur();
      return;
    }

    var opt = e.target.closest ? e.target.closest(".sc-opt") : null;
    if (opt && mount.contains(opt)) {
      var key = screen === "store" ? "store" : "discover";
      state[key] = opt.getAttribute("data-v");
      render();
      return;
    }
    var btn = e.target.closest ? e.target.closest("[data-act]") : null;
    if (!btn || !mount.contains(btn)) return;
    var act = btn.getAttribute("data-act");
    if (act === "next") {
      if (screen === "discover" && state.discover) {
        screen = state.discover === "onlyme" ? "store" : "result";
        render();
      } else if (screen === "store" && state.store) {
        screen = "result";
        render();
      }
    } else if (act === "back") {
      if (screen === "store") { screen = "discover"; render(); }
    } else if (act === "restart") {
      clearSectionHighlight();
      if (location.hash) {
        history.replaceState(null, "", location.pathname + location.search);
      }
      state = {};
      screen = "discover";
      render();
    }
  });

  state = {};
  screen = "discover";
  render();
  });
})();
