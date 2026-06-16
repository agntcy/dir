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
  var VIEW_H = 590;

  /* Column x-centres, one per stage. */
  var COL = { src: 100, ing: 330, dir: 560, dis: 790, trust: 1010, con: 1230 };
  /* Row y-centres (four source rows + directory/discover split). */
  var ROW = {
    src1: 118,
    src2: 228,
    src3: 338,
    src4: 448,
    up: 250,
    dn: 400,
    verify: 307,
    con1: 195,
    con2: 325,
    con3: 455,
  };

  var CW = 152;
  var CH = 92;

  /* Lucide icons — aligned with Interactive Product Feature Graph (Figma Make). */
  var ICON_PATHS = {
    filetext:
      '<path d="M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z"/>' +
      '<path d="M14 2v4a2 2 0 0 0 2 2h4"/><path d="M10 9H8"/><path d="M16 13H8"/><path d="M16 17H8"/>',
    files:
      '<path d="M20 7h-3a2 2 0 0 1-2-2V2"/>' +
      '<path d="M9 18a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h7l4 4v10a2 2 0 0 1-2 2Z"/>' +
      '<path d="M3 7.6v12.8A2 2 0 0 0 5 22h9.7a2 2 0 0 0 1.7-1"/>',
    globe:
      '<circle cx="12" cy="12" r="10"/>' +
      '<path d="M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20"/><path d="M2 12h20"/>',
    layers:
      '<path d="M12.83 2.18a2 2 0 0 0-1.66 0L2.6 6.08a1 1 0 0 0 0 1.83l8.58 3.91a2 2 0 0 0 1.66 0l8.58-3.9a1 1 0 0 0 0-1.83z"/>' +
      '<path d="M2 12a1 1 0 0 0 .58.91l8.6 3.91a2 2 0 0 0 1.65 0l8.58-3.9A1 1 0 0 0 22 12"/>' +
      '<path d="M2 17a1 1 0 0 0 .58.91l8.6 3.91a2 2 0 0 0 1.65 0l8.58-3.9A1 1 0 0 0 22 17"/>',
    upload:
      '<path d="M12 3v12"/><path d="m17 8-5-5-5 5"/>' +
      '<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>',
    download:
      '<path d="M12 15V3"/><path d="m7 10 5 5 5-5"/>' +
      '<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>',
    refreshcw:
      '<path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8"/>' +
      '<path d="M21 3v5h-5"/><path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16"/>' +
      '<path d="M8 16H3v5"/>',
    database:
      '<ellipse cx="12" cy="5" rx="9" ry="3"/>' +
      '<path d="M3 5V19A9 3 0 0 0 21 19V5"/><path d="M3 12A9 3 0 0 0 21 12"/>',
    radio:
      '<path d="M16.247 7.761a6 6 0 0 1 0 8.478"/>' +
      '<path d="M19.075 4.933a10 10 0 0 1 0 14.134"/>' +
      '<path d="M4.925 19.067a10 10 0 0 1 0-14.134"/>' +
      '<path d="M7.753 16.239a6 6 0 0 1 0-8.478"/><circle cx="12" cy="12" r="2"/>',
    search: '<circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/>',
    checkcircle: '<circle cx="12" cy="12" r="10"/><path d="m9 12 2 2 4-4"/>',
    arrowdown: '<path d="M12 5v14"/><path d="m19 12-7 7-7-7"/>',
    server:
      '<rect width="20" height="8" x="2" y="2" rx="2" ry="2"/>' +
      '<rect width="20" height="8" x="2" y="14" rx="2" ry="2"/>' +
      '<line x1="6" x2="6.01" y1="6" y2="6"/><line x1="6" x2="6.01" y1="18" y2="18"/>',
  };

  var NODES = [
    { id: "oasf", label: "OASF JSON", desc: "Manually built", icon: "filetext", color: SLATE, cx: COL.src, cy: ROW.src1 },
    {
      id: "files",
      label: "Files",
      desc: "MCP JSON, A2A JSON, Agent Skill Markdown",
      icon: "files",
      color: SLATE,
      cx: COL.src,
      cy: ROW.src2,
      href: "dir/dir-features-scenarios/#import",
    },
    { id: "registries", label: "External Registries", desc: "MCP registry, GitHub repos", icon: "globe", color: SLATE, cx: COL.src, cy: ROW.src3 },
    { id: "remote", label: "Other Directories", desc: "Directory Peers, OCI registries", icon: "layers", color: SLATE, cx: COL.src, cy: ROW.src4 },

    { id: "push", label: "Push", desc: "upload OASF record", icon: "upload", color: BLUE, cx: COL.ing, cy: ROW.src1 },
    { id: "import", label: "Import", desc: "translate & enrich to OASF", icon: "download", color: BLUE, cx: COL.ing, cy: 283 },
    { id: "sync", label: "Sync", desc: "replicate", icon: "refreshcw", color: BLUE, cx: COL.ing, cy: ROW.src4 },

    { id: "store", label: "Store", desc: "store OCI artifact", icon: "database", color: BLUE, cx: COL.dir, cy: ROW.up },
    { id: "routing", label: "Routing", desc: "announce record to DHT network", icon: "radio", color: BLUE, cx: COL.dir, cy: ROW.dn },

    { id: "search", label: "Search", desc: "local query", icon: "search", color: AMBER, cx: COL.dis, cy: ROW.up },
    { id: "rsearch", label: "Routing Search", desc: "discover records in peers", icon: "search", color: AMBER, cx: COL.dis, cy: ROW.dn },

    { id: "verify", label: "Verify", desc: "validate signature", icon: "checkcircle", color: TEAL, cx: COL.trust, cy: ROW.verify },

    { id: "pull", label: "Pull", desc: "raw OASF record", icon: "arrowdown", color: PURPLE, cx: COL.con, cy: ROW.con1 },
    { id: "export", label: "Export", desc: "Export to A2A, SKILL.md, GitHub Copilot MCP", icon: "upload", color: PURPLE, cx: COL.con, cy: ROW.con2 },
    { id: "mcp", label: "MCP Serve", desc: "AI tools / IDE", icon: "server", color: PURPLE, cx: COL.con, cy: ROW.con3 },
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
    { from: "files", to: "import", color: BLUE },
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

  var EDGE_BY_ID = {};
  EDGES.forEach(function (edge) {
    EDGE_BY_ID[edgeId(edge)] = edge;
  });

  /* Core process journeys: edges to highlight when selected. Nodes are derived. */
  var JOURNEYS = {
    publish: {
      label: "Publish",
      edges: [
        "oasf-push",
        "files-import",
        "registries-import",
        "import-push",
        "remote-sync",
        "push-store",
        "sync-store",
        "store-routing",
      ],
    },
    discover: {
      label: "Discover",
      edges: ["store-search", "routing-rsearch"],
    },
    verify: {
      label: "Verify",
      edges: [
        "search-verify",
        "rsearch-verify",
        "verify-pull",
        "verify-export",
        "verify-mcp",
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

  function nodeLookupForJourney(journey) {
    var nodes = {};
    journey.edges.forEach(function (edgeKey) {
      var edge = EDGE_BY_ID[edgeKey];
      if (!edge) {
        return;
      }
      nodes[edge.from] = true;
      nodes[edge.to] = true;
    });
    return nodes;
  }

  function clearJourneyVisuals(wrap) {
    wrap.querySelectorAll(".dir-graph-card[data-node-id], .dir-graph-card-link[data-node-id]").forEach(function (card) {
      card.classList.remove("dir-graph-card--active", "dir-graph-card--dimmed");
    });

    wrap.querySelectorAll(".dir-graph-branch").forEach(function (branch) {
      branch.classList.remove("dir-graph-branch--active");
    });
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
    return lines.slice(0, 3);
  }

  function buildCard(node) {
    var halfW = CW / 2;
    var halfH = CH / 2;
    var top = node.cy - halfH;
    var iconSize = 20;
    var scale = iconSize / 24;
    var tx = node.cx - iconSize / 2;
    var ty = node.cy - 28 - iconSize / 2;

    var lines = wrapText(node.desc, 22);
    var descStartY = node.cy + (lines.length === 1 ? 17 : lines.length === 2 ? 14 : 10);
    var desc = lines
      .map(function (line, i) {
        return (
          '<text class="dir-graph-card-desc" x="' + node.cx +
          '" y="' + (descStartY + i * 13) +
          '" text-anchor="middle">' + line + "</text>"
        );
      })
      .join("");

    var inner =
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
      desc;

    if (node.href) {
      return (
        '<a class="dir-graph-card-link" href="' + node.href + '" data-node-id="' + node.id + '">' +
        '<g class="dir-graph-card dir-graph-card--link">' + inner + "</g></a>"
      );
    }

    return (
      '<g class="dir-graph-card" data-node-id="' + node.id + '">' + inner + "</g>"
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

  function journeyTitleLabel(key, index) {
    var label = JOURNEYS[key].label;
    if (index > 0) {
      return label.charAt(0).toLowerCase() + label.slice(1);
    }
    return label;
  }

  function buildJourneyButtons() {
    var keys = Object.keys(JOURNEYS);
    var parts = [];

    keys.forEach(function (key, index) {
      if (index === keys.length - 1 && keys.length > 1) {
        parts.push('<span class="dir-graph-heading__sep" aria-hidden="true">, and </span>');
      } else if (index > 0) {
        parts.push('<span class="dir-graph-heading__sep" aria-hidden="true">, </span>');
      }

      parts.push(
        '<button type="button" class="dir-graph-journey-btn" role="tab" ' +
        'aria-selected="false" data-journey="' + key + '">' +
        journeyTitleLabel(key, index) +
        "</button>",
      );
    });

    return (
      '<header class="dir-graph-heading">' +
      '<div class="dir-graph-journeys" role="tablist" aria-label="Publish, discover, and verify">' +
      parts.join("") +
      "</div></header>"
    );
  }

  function applyJourneyHighlight(wrap, journeyId) {
    var journey = journeyId ? JOURNEYS[journeyId] : null;
    var journeyActive = !!journey;
    var edgeLookup = journey ? toLookup(journey.edges) : null;
    var nodeLookup = journey ? nodeLookupForJourney(journey) : null;

    clearJourneyVisuals(wrap);
    wrap.classList.toggle("dir-graph-wrap--journey-active", journeyActive);

    if (!journeyActive) {
      return;
    }

    wrap.querySelectorAll(".dir-graph-card[data-node-id], .dir-graph-card-link[data-node-id]").forEach(function (card) {
      var id = card.getAttribute("data-node-id");
      var inJourney = !!nodeLookup[id];
      card.classList.toggle("dir-graph-card--active", inJourney);
      card.classList.toggle("dir-graph-card--dimmed", !inJourney);
    });

    wrap.querySelectorAll(".dir-graph-branch").forEach(function (branch) {
      var id = branch.getAttribute("data-edge-id");
      if (edgeLookup[id]) {
        branch.classList.add("dir-graph-branch--active");
      }
    });
  }

  function updateJourneyButtonStates(wrap) {
    var pinned = wrap.getAttribute("data-pinned-journey") || "";
    var preview = wrap.getAttribute("data-preview-journey") || "";

    wrap.querySelectorAll(".dir-graph-journey-btn").forEach(function (btn) {
      var id = btn.getAttribute("data-journey");
      btn.classList.toggle("is-active", id === pinned);
      btn.classList.toggle("is-preview", id === preview && id !== pinned);
      btn.setAttribute("aria-selected", id === pinned ? "true" : "false");
    });
  }

  function setPinnedJourney(wrap, journeyId) {
    wrap.setAttribute("data-pinned-journey", journeyId || "");
    wrap.removeAttribute("data-preview-journey");
    applyJourneyHighlight(wrap, journeyId || null);
    updateJourneyButtonStates(wrap);
  }

  function previewJourney(wrap, journeyId) {
    if (!journeyId || !JOURNEYS[journeyId]) {
      return;
    }

    var pinned = wrap.getAttribute("data-pinned-journey") || "";
    if (journeyId === pinned) {
      wrap.removeAttribute("data-preview-journey");
    } else {
      wrap.setAttribute("data-preview-journey", journeyId);
    }

    applyJourneyHighlight(wrap, journeyId);
    updateJourneyButtonStates(wrap);
  }

  function clearJourneyPreview(wrap) {
    if (!wrap.getAttribute("data-preview-journey")) {
      return;
    }

    wrap.removeAttribute("data-preview-journey");
    var pinned = wrap.getAttribute("data-pinned-journey") || "";
    applyJourneyHighlight(wrap, pinned || null);
    updateJourneyButtonStates(wrap);
  }

  function handleJourneyClick(wrap, event) {
    var btn = event.target.closest(".dir-graph-journey-btn");
    if (!btn || !wrap.contains(btn)) {
      return;
    }

    var journey = btn.getAttribute("data-journey");
    var pinned = wrap.getAttribute("data-pinned-journey") || "";
    setPinnedJourney(wrap, pinned === journey ? null : journey);
  }

  function handleJourneyPointerOver(wrap, event) {
    var btn = event.target.closest(".dir-graph-journey-btn");
    if (!btn || !wrap.contains(btn)) {
      return;
    }

    previewJourney(wrap, btn.getAttribute("data-journey"));
  }

  function handleJourneyPointerOut(wrap, event) {
    var btn = event.target.closest(".dir-graph-journey-btn");
    if (!btn || !wrap.contains(btn)) {
      return;
    }

    var related = event.relatedTarget;
    if (related && related.closest && related.closest(".dir-graph-journey-btn")) {
      return;
    }

    clearJourneyPreview(wrap);
  }

  function renderGraph(wrap) {
    var pinnedJourney = wrap.getAttribute("data-pinned-journey") || "";

    wrap.innerHTML =
      buildJourneyButtons() + '<div class="dir-graph-stage">' + buildSvg() + "</div>";

    if (pinnedJourney && JOURNEYS[pinnedJourney]) {
      setPinnedJourney(wrap, pinnedJourney);
    }
  }

  function initGraph(wrap) {
    if (wrap.dataset.dirGraphInit) {
      return;
    }

    wrap.dataset.dirGraphInit = "1";
    wrap.addEventListener("click", function (event) {
      handleJourneyClick(wrap, event);
    });
    wrap.addEventListener("mouseover", function (event) {
      handleJourneyPointerOver(wrap, event);
    });
    wrap.addEventListener("mouseout", function (event) {
      handleJourneyPointerOut(wrap, event);
    });
    renderGraph(wrap);
  }

  document
    .querySelectorAll(".dir-graph-wrap[data-dir-graph]")
    .forEach(initGraph);
});
