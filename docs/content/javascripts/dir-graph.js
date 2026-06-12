/* Copyright AGNTCY Contributors (https://github.com/agntcy) */
/* SPDX-License-Identifier: Apache-2.0 */

/* Operations radial view: static mind-map SVG with flow particles. */
document$.subscribe(function () {
  var BLUE = "#4d8fd4";
  var AMBER = "#f0a830";
  var TEAL = "#2dd4bf";
  var PURPLE = "#a78bfa";

  var ICON_PATHS = {
    import:
      '<path d="M12 17V3"/><path d="m6 11 6 6 6-6"/><path d="M19 21H5"/>',
    acquire:
      '<circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/>',
    build:
      '<path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"/>',
    verify: '<path d="M20 6 9 17l-5-5"/>',
    sync:
      '<path d="m2 9 3-3 3 3"/><path d="M13 18H7a2 2 0 0 1-2-2V6"/><path d="m22 15-3 3-3-3"/><path d="M11 6h6a2 2 0 0 1 2 2v10"/>',
    export:
      '<path d="m18 9-6-6-6 6"/><path d="M12 3v14"/><path d="M5 21h14"/>',
    publish:
      '<path d="m3 11 19-9-9 19-2-8-8-2z"/>',
    store:
      '<rect width="20" height="5" x="2" y="3" rx="1"/><path d="M4 8v11a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8"/><path d="M10 12h4"/>',
  };

  var NODES = [
    {
      id: "import",
      label: "Import",
      color: BLUE,
      flow: "outbound",
      cx: 118,
      cy: 240,
      cardDesc: "Pull from connectors",
    },
    {
      id: "acquire",
      label: "Acquire",
      color: BLUE,
      flow: "outbound",
      cx: 118,
      cy: 452,
      cardDesc: "Scan catalogs",
    },
    {
      id: "build",
      label: "Build",
      color: AMBER,
      flow: "outbound",
      cx: 600,
      cy: 76,
      cardDesc: "Merge & enrich",
    },
    {
      id: "verify",
      label: "Verify",
      color: TEAL,
      flow: "pingpong",
      cx: 1062,
      cy: 240,
      cardDesc: "Validate & audit",
    },
    {
      id: "sync",
      label: "Sync",
      color: TEAL,
      flow: "pingpong",
      cx: 1062,
      cy: 452,
      cardDesc: "Propagate deltas",
    },
    {
      id: "export",
      label: "Export",
      color: PURPLE,
      flow: "inbound",
      cx: 278,
      cy: 614,
      cardDesc: "Serialize to targets",
    },
    {
      id: "publish",
      label: "Publish",
      color: PURPLE,
      flow: "inbound",
      cx: 600,
      cy: 648,
      cardDesc: "Broadcast via event bus",
    },
    {
      id: "store",
      label: "Store",
      color: PURPLE,
      flow: "inbound",
      cx: 922,
      cy: 614,
      cardDesc: "Commit to registry",
    },
  ];

  var GROUP_LABELS = [
    { label: "Acquire", x: 308, y: 338, color: BLUE },
    { label: "Build", x: 600, y: 196, color: AMBER },
    { label: "Verify & Sync", x: 882, y: 338, color: TEAL },
    { label: "Publish", x: 600, y: 526, color: PURPLE },
  ];

  var HCX = 600;
  var HCY = 345;
  var HW = 230;
  var HH = 72;
  var CW = 158;
  var CH = 122;
  var FLOW_DUR = 2.6;
  var FLOW_DUR_JITTER = 0.5;
  /* Ping-pong runs hub → card → hub in one cycle; 2× duration matches leg speed. */
  var FLOW_DUR_PINGPONG_MULT = 2;

  function randomFlowDur(flow) {
    var base =
      flow === "pingpong" ? FLOW_DUR * FLOW_DUR_PINGPONG_MULT : FLOW_DUR;
    return (
      base +
      (Math.random() * FLOW_DUR_JITTER * 2 - FLOW_DUR_JITTER)
    ).toFixed(2);
  }

  function branchPath(fcx, fcy) {
    var cpx = HCX + (fcx - HCX) * 0.45;
    var cpy = HCY + (fcy - HCY) * 0.45;
    return (
      "M" +
      HCX +
      "," +
      HCY +
      " Q" +
      cpx +
      "," +
      cpy +
      " " +
      fcx +
      "," +
      fcy
    );
  }

  function inboundPath(fcx, fcy) {
    var cpx = HCX + (fcx - HCX) * 0.45;
    var cpy = HCY + (fcy - HCY) * 0.45;
    return (
      "M" +
      fcx +
      "," +
      fcy +
      " Q" +
      cpx +
      "," +
      cpy +
      " " +
      HCX +
      "," +
      HCY
    );
  }

  function buildDotGrid() {
    var dots = [];
    for (var x = 20; x <= 1180; x += 40) {
      for (var y = 20; y <= 700; y += 40) {
        dots.push(
          '<circle cx="' + x + '" cy="' + y + '" r="1" class="dir-graph-dot"/>',
        );
      }
    }
    return dots.join("");
  }

  function buildParticle(node, delay) {
    if (node.flow === "none") {
      return "";
    }

    var dur = randomFlowDur(node.flow) + "s";
    var motion = "";

    if (node.flow === "inbound") {
      motion =
        '<animateMotion dur="' +
        dur +
        '" repeatCount="1" begin="' +
        delay +
        's" path="' +
        inboundPath(node.cx, node.cy) +
        '"/>';
    } else if (node.flow === "pingpong") {
      motion =
        '<animateMotion dur="' +
        dur +
        '" repeatCount="1" begin="' +
        delay +
        's" path="' +
        branchPath(node.cx, node.cy) +
        '" keyPoints="0;1;0" keyTimes="0;0.5;1" calcMode="linear"/>';
    } else {
      motion =
        '<animateMotion dur="' +
        dur +
        '" repeatCount="1" begin="' +
        delay +
        's" path="' +
        branchPath(node.cx, node.cy) +
        '"/>';
    }

    return (
      '<g class="dir-graph-particle" data-flow="' +
      node.flow +
      '">' +
      motion +
      '<animate attributeName="opacity" values="0;0.9;0.9;0" dur="' +
      dur +
      '" repeatCount="1" begin="' +
      delay +
      's"/>' +
      '<circle r="2.5" fill="' +
      node.color +
      '" filter="url(#dir-graph-particle-glow)"/>' +
      "</g>"
    );
  }

  function bindParticleCycleRandomization(wrap) {
    wrap.querySelectorAll(".dir-graph-particle").forEach(function (particle) {
      var flow = particle.getAttribute("data-flow");
      var motion = particle.querySelector("animateMotion");
      var opacity = particle.querySelector('animate[attributeName="opacity"]');
      if (!motion || !opacity || !flow || flow === "none") {
        return;
      }

      motion.addEventListener("endEvent", function () {
        var dur = randomFlowDur(flow) + "s";
        motion.setAttribute("dur", dur);
        opacity.setAttribute("dur", dur);
        motion.beginElement();
        opacity.beginElement();
      });
    });
  }

  function buildCard(node) {
    var halfW = CW / 2;
    var halfH = CH / 2;
    var iconSize = 22;
    var scale = iconSize / 24;
    var tx = node.cx - iconSize / 2;
    var ty = node.cy - 30 - iconSize / 2;

    return (
      '<g class="dir-graph-card">' +
      '<rect class="dir-graph-card-fill" x="' +
      (node.cx - halfW) +
      '" y="' +
      (node.cy - halfH) +
      '" width="' +
      CW +
      '" height="' +
      CH +
      '" rx="10"/>' +
      '<rect class="dir-graph-card-tint" x="' +
      (node.cx - halfW) +
      '" y="' +
      (node.cy - halfH) +
      '" width="' +
      CW +
      '" height="' +
      CH +
      '" rx="10" fill="' +
      node.color +
      '"/>' +
      '<rect class="dir-graph-card-border" x="' +
      (node.cx - halfW) +
      '" y="' +
      (node.cy - halfH) +
      '" width="' +
      CW +
      '" height="' +
      CH +
      '" rx="10" stroke="' +
      node.color +
      '"/>' +
      '<g class="dir-graph-card-icon" transform="translate(' +
      tx +
      "," +
      ty +
      ") scale(" +
      scale.toFixed(4) +
      ')" fill="none" stroke="' +
      node.color +
      '" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">' +
      ICON_PATHS[node.id] +
      "</g>" +
      '<text class="dir-graph-card-label" x="' +
      node.cx +
      '" y="' +
      (node.cy + 16) +
      '" text-anchor="middle">' +
      node.label +
      "</text>" +
      '<text class="dir-graph-card-desc" x="' +
      node.cx +
      '" y="' +
      (node.cy + 35) +
      '" text-anchor="middle">' +
      node.cardDesc +
      "</text>" +
      "</g>"
    );
  }

  function buildSvg() {
    var branches = NODES.map(function (node, i) {
      return (
        '<g class="dir-graph-branch">' +
        '<path class="dir-graph-branch-line" d="' +
        branchPath(node.cx, node.cy) +
        '" stroke="' +
        node.color +
        '"/>' +
        buildParticle(node, (i * 0.38).toFixed(2)) +
        "</g>"
      );
    }).join("");

    var groupLabels = GROUP_LABELS.map(function (group) {
      return (
        '<text class="dir-graph-group-label" x="' +
        group.x +
        '" y="' +
        group.y +
        '" text-anchor="middle" fill="' +
        group.color +
        '">' +
        group.label.toUpperCase() +
        "</text>"
      );
    }).join("");

    var cards = NODES.map(buildCard).join("");

    return (
      '<svg class="dir-graph-root" viewBox="0 0 1200 720" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">' +
      "<defs>" +
      '<radialGradient id="dir-graph-wash" cx="50%" cy="50%" r="55%">' +
      '<stop offset="0%" stop-color="' +
      BLUE +
      '" stop-opacity="0.08"/>' +
      '<stop offset="100%" stop-color="' +
      BLUE +
      '" stop-opacity="0"/>' +
      "</radialGradient>" +
      '<radialGradient id="dir-graph-hub-glow" cx="50%" cy="50%" r="50%">' +
      '<stop offset="0%" stop-color="' +
      BLUE +
      '" stop-opacity="0.22"/>' +
      '<stop offset="100%" stop-color="' +
      BLUE +
      '" stop-opacity="0"/>' +
      "</radialGradient>" +
      '<filter id="dir-graph-particle-glow" x="-100%" y="-100%" width="300%" height="300%">' +
      '<feGaussianBlur in="SourceGraphic" stdDeviation="1.6" result="blur"/>' +
      "<feMerge>" +
      '<feMergeNode in="blur"/>' +
      '<feMergeNode in="SourceGraphic"/>' +
      "</feMerge>" +
      "</filter>" +
      "</defs>" +
      buildDotGrid() +
      '<rect width="1200" height="720" fill="url(#dir-graph-wash)"/>' +
      branches +
      groupLabels +
      '<ellipse cx="' +
      HCX +
      '" cy="' +
      HCY +
      '" rx="' +
      HW * 0.6 +
      '" ry="' +
      HH * 1.6 +
      '" fill="url(#dir-graph-hub-glow)"/>' +
      '<rect class="dir-graph-hub-ring" x="' +
      (HCX - HW / 2 - 7) +
      '" y="' +
      (HCY - HH / 2 - 7) +
      '" width="' +
      (HW + 14) +
      '" height="' +
      (HH + 14) +
      '" rx="' +
      (HH / 2 + 7) +
      '"/>' +
      '<rect class="dir-graph-hub-fill" x="' +
      (HCX - HW / 2) +
      '" y="' +
      (HCY - HH / 2) +
      '" width="' +
      HW +
      '" height="' +
      HH +
      '" rx="' +
      HH / 2 +
      '"/>' +
      '<text class="dir-graph-hub-title" x="' +
      HCX +
      '" y="' +
      (HCY - 4) +
      '" text-anchor="middle">Agent Directory Service</text>' +
      '<text class="dir-graph-hub-sub" x="' +
      HCX +
      '" y="' +
      (HCY + 17) +
      '" text-anchor="middle">8 OPERATIONS</text>' +
      cards +
      "</svg>"
    );
  }

  document.querySelectorAll(".dir-graph-wrap[data-dir-graph]").forEach(function (wrap) {
    if (wrap.dataset.graphInit === "true") {
      return;
    }
    wrap.dataset.graphInit = "true";
    wrap.innerHTML = buildSvg();
    bindParticleCycleRandomization(wrap);
  });
});
