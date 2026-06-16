/* Copyright AGNTCY Contributors (https://github.com/agntcy) */
/* SPDX-License-Identifier: Apache-2.0 */

/* Operations workflow: horizontal pipeline SVG with flowing particles.
   Stages, left to right: Record Sources -> Ingest -> Directory -> Discover
   -> Trust & Provenance -> Consume. Journey buttons highlight paths per process. */
document$.subscribe(function () {
  var BLUE = "#4d8fd4";
  var AMBER = "#f0a830";
  var TEAL = "#2dd4bf";
  var PURPLE = "#a78bfa";
  var SLATE = "#7c8aa0";

  var VIEW_W = 1330;
  var VIEW_H = 540;

  /* Column x-centres, one per stage. */
  var COL = { src: 100, ing: 330, dir: 560, dis: 790, trust: 1010, con: 1230 };
  /* Row y-centres. */
  var ROW = { top: 145, mid: 300, bot: 455, up: 222, dn: 378, ctr: 300 };

  var CW = 152;
  var CH = 92;

  var ICON_PATHS = {
    file: '<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 2v6h6"/>',
    registries:
      '<rect width="20" height="8" x="2" y="2" rx="2"/><rect width="20" height="8" x="2" y="14" rx="2"/><path d="M6 6h.01"/><path d="M6 18h.01"/>',
    globe:
      '<circle cx="12" cy="12" r="10"/><path d="M2 12h20"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>',
    push: '<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><path d="m17 8-5-5-5 5"/><path d="M12 3v12"/>',
    import:
      '<path d="M12 17V3"/><path d="m6 11 6 6 6-6"/><path d="M19 21H5"/>',
    sync: '<path d="m2 9 3-3 3 3"/><path d="M13 18H7a2 2 0 0 1-2-2V6"/><path d="m22 15-3 3-3-3"/><path d="M11 6h6a2 2 0 0 1 2 2v10"/>',
    store:
      '<rect width="20" height="5" x="2" y="3" rx="1"/><path d="M4 8v11a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8"/><path d="M10 12h4"/>',
    routing:
      '<rect x="16" y="16" width="6" height="6" rx="1"/><rect x="2" y="16" width="6" height="6" rx="1"/><rect x="9" y="2" width="6" height="6" rx="1"/><path d="M5 16v-3a1 1 0 0 1 1-1h12a1 1 0 0 1 1 1v3"/><path d="M12 12V8"/>',
    search: '<circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/>',
    rsearch:
      '<circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/><path d="M11 8v6"/><path d="M8 11h6"/>',
    verify: '<path d="M20 6 9 17l-5-5"/>',
    pull: '<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><path d="m7 10 5 5 5-5"/><path d="M12 15V3"/>',
    export:
      '<path d="m18 9-6-6-6 6"/><path d="M12 3v14"/><path d="M5 21h14"/>',
    mcp: '<path d="M12 8V4H8"/><rect width="16" height="12" x="4" y="8" rx="2"/><path d="M2 14h2"/><path d="M20 14h2"/><path d="M15 13v2"/><path d="M9 13v2"/>',
  };

  var NODES = [
    { id: "oasf", label: "OASF JSON", desc: "Manually built", icon: "file", color: SLATE, cx: COL.src, cy: ROW.top },
    { id: "registries", label: "External Registries", desc: "MCP registry, GitHub repos", icon: "registries", color: SLATE, cx: COL.src, cy: ROW.mid },
    { id: "remote", label: "Other Directories", desc: "Directory Peers, OCI registries", icon: "globe", color: SLATE, cx: COL.src, cy: ROW.bot },

    { id: "push", label: "Push", desc: "upload OASF record", icon: "push", color: BLUE, cx: COL.ing, cy: ROW.top },
    { id: "import", label: "Import", desc: "translate & enrich to OASF", icon: "import", color: BLUE, cx: COL.ing, cy: ROW.mid },
    { id: "sync", label: "Sync", desc: "replicate", icon: "sync", color: BLUE, cx: COL.ing, cy: ROW.bot },

    { id: "store", label: "Store", desc: "store OCI artifact", icon: "store", color: BLUE, cx: COL.dir, cy: ROW.up },
    { id: "routing", label: "Routing", desc: "announce record to DHT network", icon: "routing", color: BLUE, cx: COL.dir, cy: ROW.dn },

    { id: "search", label: "Search", desc: "local query", icon: "search", color: AMBER, cx: COL.dis, cy: ROW.up },
    { id: "rsearch", label: "Routing Search", desc: "discover records in peers", icon: "rsearch", color: AMBER, cx: COL.dis, cy: ROW.dn },

    { id: "verify", label: "Verify", desc: "validate signature (Cosign)", icon: "verify", color: TEAL, cx: COL.trust, cy: ROW.ctr },

    { id: "pull", label: "Pull", desc: "raw OASF record", icon: "pull", color: PURPLE, cx: COL.con, cy: ROW.top },
    { id: "export", label: "Export", desc: "to A2A, SKILL.md, Copilot\u2026", icon: "export", color: PURPLE, cx: COL.con, cy: ROW.mid },
    { id: "mcp", label: "MCP Serve", desc: "AI tools / IDE", icon: "mcp", color: PURPLE, cx: COL.con, cy: ROW.bot },
  ];

  var NODE_BY_ID = {};
  NODES.forEach(function (n) {
    NODE_BY_ID[n.id] = n;
  });

  var GROUP_LABELS = [
    { label: "Record Sources", x: COL.src, color: SLATE },
    { label: "Ingest", x: COL.ing, color: BLUE },
    { label: "Directory", x: COL.dir, color: BLUE },
    { label: "Discover", x: COL.dis, color: AMBER },
    { label: "Trust & Provenance", x: COL.trust, color: TEAL },
    { label: "Consume", x: COL.con, color: PURPLE },
  ];

  /* from -> to edges; colour follows the downstream stage. */
  var EDGES = [
    { from: "oasf", to: "push", color: BLUE },
    { from: "registries", to: "import", color: BLUE },
    { from: "remote", to: "sync", color: BLUE },
    { from: "import", to: "push", color: BLUE },
    { from: "push", to: "store", color: BLUE },
    { from: "sync", to: "store", color: BLUE },
    { from: "store", to: "routing", color: BLUE },
    { from: "store", to: "search", color: AMBER },
    { from: "routing", to: "rsearch", color: AMBER },
    { from: "search", to: "verify", color: TEAL },
    { from: "rsearch", to: "verify", color: TEAL },
    { from: "verify", to: "pull", color: PURPLE },
    { from: "verify", to: "export", color: PURPLE },
    { from: "verify", to: "mcp", color: PURPLE },
  ];

  /* Core process journeys: nodes and edges to highlight when selected. */
  var JOURNEYS = {
    publish: {
      label: "Publish",
      nodes: ["oasf", "push", "store", "routing"],
      edges: ["oasf-push", "push-store", "store-routing"],
    },
    discover: {
      label: "Discover",
      nodes: ["store", "routing", "search", "rsearch", "verify", "pull", "export", "mcp"],
      edges: [
        "store-search",
        "store-routing",
        "routing-rsearch",
        "search-verify",
        "rsearch-verify",
        "verify-pull",
        "verify-export",
        "verify-mcp",
      ],
    },
    aggregate: {
      label: "Aggregate",
      nodes: ["registries", "remote", "import", "sync", "push", "store"],
      edges: ["registries-import", "import-push", "push-store", "remote-sync", "sync-store"],
    },
    federation: {
      label: "Federation",
      nodes: ["oasf", "remote", "push", "sync", "store", "routing", "rsearch"],
      edges: [
        "oasf-push",
        "remote-sync",
        "push-store",
        "sync-store",
        "store-routing",
        "routing-rsearch",
      ],
    },
  };

  var FLOW_DUR = 3.0;

  function edgeId(edge) {
    return edge.from + "-" + edge.to;
  }

  function toLookup(items) {
    var lookup = {};
    items.forEach(function (item) {
      lookup[item] = true;
    });
    return lookup;
  }

  function edgePath(a, b) {
    if (a.cx === b.cx) {
      /* Same column: vertical curve bowing to the left. */
      var down = b.cy > a.cy;
      var sy = a.cy + (down ? CH / 2 : -CH / 2);
      var ty = b.cy + (down ? -CH / 2 : CH / 2);
      var bow = a.cx - 52;
      var span = ty - sy;
      return (
        "M" + a.cx + "," + sy +
        " C" + bow + "," + (sy + span * 0.3) +
        " " + bow + "," + (ty - span * 0.3) +
        " " + b.cx + "," + ty
      );
    }
    /* Different columns: horizontal S-curve, right edge -> left edge. */
    var sx = a.cx + CW / 2;
    var tx = b.cx - CW / 2;
    var dx = (tx - sx) * 0.5;
    return (
      "M" + sx + "," + a.cy +
      " C" + (sx + dx) + "," + a.cy +
      " " + (tx - dx) + "," + b.cy +
      " " + tx + "," + b.cy
    );
  }

  function buildParticle(path, color, delay, dur) {
    return (
      '<g class="dir-graph-particle" data-flow="outbound">' +
      '<animateMotion dur="' + dur + 's" repeatCount="indefinite" begin="' +
      delay + 's" path="' + path + '"/>' +
      '<animate attributeName="opacity" values="0;0.9;0.9;0" dur="' +
      dur + 's" repeatCount="indefinite" begin="' + delay + 's"/>' +
      '<circle r="2.6" fill="' + color +
      '" filter="url(#dir-graph-particle-glow)"/>' +
      "</g>"
    );
  }

  function wrapText(text, maxChars) {
    var words = text.split(" ");
    var lines = [];
    var line = "";
    words.forEach(function (word) {
      var candidate = line ? line + " " + word : word;
      if (candidate.length > maxChars && line) {
        lines.push(line);
        line = word;
      } else {
        line = candidate;
      }
    });
    if (line) {
      lines.push(line);
    }
    return lines.slice(0, 2);
  }

  function buildCard(node) {
    var halfW = CW / 2;
    var halfH = CH / 2;
    var top = node.cy - halfH;
    var iconSize = 20;
    var scale = iconSize / 24;
    var tx = node.cx - iconSize / 2;
    var ty = node.cy - 28 - iconSize / 2;

    var lines = wrapText(node.desc, 24);
    var descStartY = lines.length > 1 ? node.cy + 14 : node.cy + 17;
    var desc = lines
      .map(function (line, i) {
        return (
          '<text class="dir-graph-card-desc" x="' + node.cx +
          '" y="' + (descStartY + i * 13) +
          '" text-anchor="middle">' + line + "</text>"
        );
      })
      .join("");

    return (
      '<g class="dir-graph-card" data-node-id="' + node.id + '">' +
      '<rect class="dir-graph-card-fill" x="' + (node.cx - halfW) +
      '" y="' + top + '" width="' + CW + '" height="' + CH + '" rx="12"/>' +
      '<rect class="dir-graph-card-tint" x="' + (node.cx - halfW) +
      '" y="' + top + '" width="' + CW + '" height="' + CH +
      '" rx="12" fill="' + node.color + '"/>' +
      '<rect class="dir-graph-card-border" x="' + (node.cx - halfW) +
      '" y="' + top + '" width="' + CW + '" height="' + CH +
      '" rx="12" stroke="' + node.color + '"/>' +
      '<g class="dir-graph-card-icon" transform="translate(' + tx + "," + ty +
      ") scale(" + scale.toFixed(4) + ')" fill="none" stroke="' + node.color +
      '" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">' +
      ICON_PATHS[node.icon] + "</g>" +
      '<text class="dir-graph-card-label" x="' + node.cx +
      '" y="' + (node.cy - 2) + '" text-anchor="middle">' + node.label +
      "</text>" +
      desc +
      "</g>"
    );
  }

  function buildDotGrid() {
    var dots = [];
    for (var x = 20; x <= VIEW_W - 20; x += 40) {
      for (var y = 20; y <= VIEW_H - 20; y += 40) {
        dots.push(
          '<circle cx="' + x + '" cy="' + y + '" r="1" class="dir-graph-dot"/>',
        );
      }
    }
    return dots.join("");
  }

  function buildSvg() {
    var edges = EDGES.map(function (edge, i) {
      var a = NODE_BY_ID[edge.from];
      var b = NODE_BY_ID[edge.to];
      var path = edgePath(a, b);
      return (
        '<g class="dir-graph-branch" data-edge-id="' + edgeId(edge) + '">' +
        '<path class="dir-graph-branch-line" d="' + path +
        '" stroke="' + edge.color + '"/>' +
        buildParticle(path, edge.color, (i * 0.32).toFixed(2), FLOW_DUR.toFixed(2)) +
        "</g>"
      );
    }).join("");

    var groups = GROUP_LABELS.map(function (group) {
      return (
        '<text class="dir-graph-group-label" x="' + group.x +
        '" y="58" text-anchor="middle" fill="' + group.color + '">' +
        group.label.toUpperCase() + "</text>"
      );
    }).join("");

    /* Faint backdrop highlighting the Directory core. */
    var coreTop = ROW.up - CH / 2 - 22;
    var coreBottom = ROW.dn + CH / 2 + 22;
    var core =
      '<rect x="' + (COL.dir - CW / 2 - 16) + '" y="' + coreTop +
      '" width="' + (CW + 32) + '" height="' + (coreBottom - coreTop) +
      '" rx="18" fill="' + BLUE + '" fill-opacity="0.05" stroke="' + BLUE +
      '" stroke-opacity="0.22" stroke-width="1.2"/>';

    var cards = NODES.map(buildCard).join("");

    return (
      '<svg class="dir-graph-root" viewBox="0 0 ' + VIEW_W + " " + VIEW_H +
      '" preserveAspectRatio="xMidYMid meet" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">' +
      "<defs>" +
      '<radialGradient id="dir-graph-wash" cx="50%" cy="50%" r="60%">' +
      '<stop offset="0%" stop-color="' + BLUE + '" stop-opacity="0.07"/>' +
      '<stop offset="100%" stop-color="' + BLUE + '" stop-opacity="0"/>' +
      "</radialGradient>" +
      '<filter id="dir-graph-particle-glow" x="-100%" y="-100%" width="300%" height="300%">' +
      '<feGaussianBlur in="SourceGraphic" stdDeviation="1.6" result="blur"/>' +
      "<feMerge><feMergeNode in=\"blur\"/><feMergeNode in=\"SourceGraphic\"/></feMerge>" +
      "</filter>" +
      "</defs>" +
      buildDotGrid() +
      '<rect width="' + VIEW_W + '" height="' + VIEW_H +
      '" fill="url(#dir-graph-wash)"/>' +
      core +
      groups +
      edges +
      cards +
      "</svg>"
    );
  }

  function buildJourneyButtons() {
    return (
      '<div class="dir-graph-journeys" role="tablist" aria-label="Highlight a process">' +
      Object.keys(JOURNEYS)
        .map(function (key) {
          return (
            '<button type="button" class="dir-graph-journey-btn" role="tab" ' +
            'aria-selected="false" data-journey="' + key + '">' +
            JOURNEYS[key].label +
            "</button>"
          );
        })
        .join("") +
      "</div>"
    );
  }

  function setActiveJourney(wrap, journeyId) {
    var journey = journeyId ? JOURNEYS[journeyId] : null;
    var nodeLookup = journey ? toLookup(journey.nodes) : null;
    var edgeLookup = journey ? toLookup(journey.edges) : null;

    wrap.setAttribute("data-active-journey", journeyId || "");
    wrap.classList.toggle("dir-graph-wrap--journey-active", !!journey);

    wrap.querySelectorAll(".dir-graph-card").forEach(function (card) {
      var id = card.getAttribute("data-node-id");
      var inJourney = !journey || nodeLookup[id];
      card.classList.toggle("dir-graph-card--active", !!journey && inJourney);
      card.classList.toggle("dir-graph-card--dimmed", !!journey && !inJourney);
    });

    wrap.querySelectorAll(".dir-graph-branch").forEach(function (branch) {
      var id = branch.getAttribute("data-edge-id");
      var inJourney = !journey || edgeLookup[id];
      branch.classList.toggle("dir-graph-branch--active", !!journey && inJourney);
      branch.classList.toggle("dir-graph-branch--dimmed", !!journey && !inJourney);
    });

    wrap.querySelectorAll(".dir-graph-journey-btn").forEach(function (btn) {
      var selected = btn.getAttribute("data-journey") === journeyId;
      btn.classList.toggle("is-active", selected);
      btn.setAttribute("aria-selected", selected ? "true" : "false");
    });
  }

  function renderGraph(wrap) {
    wrap.innerHTML =
      '<div class="dir-graph-stage">' + buildSvg() + "</div>" + buildJourneyButtons();

    wrap.querySelectorAll(".dir-graph-journey-btn").forEach(function (btn) {
      btn.addEventListener("click", function () {
        var journey = btn.getAttribute("data-journey");
        var active = wrap.getAttribute("data-active-journey");
        setActiveJourney(wrap, active === journey ? null : journey);
      });
    });
  }

  document
    .querySelectorAll(".dir-graph-wrap[data-dir-graph]")
    .forEach(renderGraph);
});
