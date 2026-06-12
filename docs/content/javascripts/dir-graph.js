/* Copyright AGNTCY Contributors (https://github.com/agntcy) */
/* SPDX-License-Identifier: Apache-2.0 */

/* Interactive lifecycle graph: animated spokes + hover detail panel. */
document$.subscribe(function () {
  var BLUE = "#4d8fd4";
  var AMBER = "#f0a830";

  var ICON_PATHS = {
    import:
      '<path d="M12 17V3"/><path d="m6 11 6 6 6-6"/><path d="M19 21H5"/>',
    discover:
      '<circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/>',
    build:
      '<path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z"/>',
    verify: '<path d="M20 6 9 17l-5-5"/>',
    sync:
      '<path d="m2 9 3-3 3 3"/><path d="M13 18H7a2 2 0 0 1-2-2V6"/><path d="m22 15-3 3-3-3"/><path d="M11 6h6a2 2 0 0 1 2 2v10"/>',
    export:
      '<path d="m18 9-6-6-6 6"/><path d="M12 3v14"/><path d="M5 21h14"/>',
    announce:
      '<path d="m3 11 19-9-9 19-2-8-8-2z"/>',
    store:
      '<rect width="20" height="5" x="2" y="3" rx="1"/><path d="M4 8v11a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8"/><path d="M10 12h4"/>',
  };

  var NODES = [
    {
      id: "import",
      label: "Import",
      group: "Acquire",
      desc: "Register agent definitions from external registries and platforms",
    },
    {
      id: "discover",
      label: "Discover",
      group: "Acquire",
      desc: "Search and filter agents by capability, skill, or attribute",
    },
    {
      id: "build",
      label: "Build",
      group: "Build",
      desc: "Assemble new agents and configure multi-step pipelines",
    },
    {
      id: "verify",
      label: "Verify",
      group: "Verify & Sync",
      desc: "Validate schema, identity, and access policies before activation",
    },
    {
      id: "sync",
      label: "Sync",
      group: "Verify & Sync",
      desc: "Propagate state changes consistently across all environments",
    },
    {
      id: "export",
      label: "Export",
      group: "Publish",
      desc: "Bundle and distribute agent packages to target destinations",
    },
    {
      id: "announce",
      label: "Announce",
      group: "Publish",
      desc: "Broadcast agent availability to all subscribed downstream services",
    },
    {
      id: "store",
      label: "Store",
      group: "Publish",
      desc: "Persist and version agent records to durable long-term storage",
    },
  ];

  var CX = 400;
  var CY = 340;
  var ORBIT = 215;
  var NODE_R = 36;

  function polarToXY(angleDeg, radius) {
    var rad = ((angleDeg - 90) * Math.PI) / 180;
    return { x: CX + radius * Math.cos(rad), y: CY + radius * Math.sin(rad) };
  }

  function accentColor(i) {
    return i % 2 === 0 ? BLUE : AMBER;
  }

  function labelAnchor(angle) {
    if (angle < 20 || angle > 340) {
      return "middle";
    }
    if (angle > 160 && angle < 200) {
      return "middle";
    }
    if (angle >= 20 && angle <= 160) {
      return "start";
    }
    return "end";
  }

  function labelOffsetX(angle) {
    if (angle >= 20 && angle <= 160) {
      return 4;
    }
    if (angle >= 200 && angle <= 340) {
      return -4;
    }
    return 0;
  }

  function buildDotGrid() {
    var dots = [];
    for (var x = 20; x <= 780; x += 40) {
      for (var y = 20; y <= 660; y += 40) {
        dots.push(
          '<circle cx="' + x + '" cy="' + y + '" r="1" class="dir-graph-dot"/>',
        );
      }
    }
    return dots.join("");
  }

  function buildParticles(positions, animate) {
    if (!animate) {
      return "";
    }
    return positions
      .map(function (n, i) {
        var accent = accentColor(i);
        var delay = (i * 0.38).toFixed(2);
        return (
          '<circle r="2.5" fill="' +
          accent +
          '" class="dir-graph-particle">' +
          '<animateMotion dur="2.4s" repeatCount="indefinite" begin="' +
          delay +
          's" path="M' +
          CX +
          "," +
          CY +
          " L" +
          n.x.toFixed(2) +
          "," +
          n.y.toFixed(2) +
          '"/>' +
          '<animate attributeName="opacity" values="0;0.9;0.9;0" dur="2.4s" repeatCount="indefinite" begin="' +
          delay +
          's"/>' +
          "</circle>"
        );
      })
      .join("");
  }

  function buildSvg(animate) {
    var positions = NODES.map(function (n, i) {
      var pt = polarToXY((i * 360) / NODES.length, ORBIT);
      return {
        id: n.id,
        label: n.label,
        x: pt.x,
        y: pt.y,
        angle: (i * 360) / NODES.length,
        index: i,
      };
    });

    var lines = positions
      .map(function (n, i) {
        var next = positions[(i + 1) % positions.length];
        return (
          '<g class="dir-graph-spoke" data-node="' +
          n.id +
          '" data-accent="' +
          accentColor(i) +
          '">' +
          '<line class="dir-graph-spoke-line" x1="' +
          CX +
          '" y1="' +
          CY +
          '" x2="' +
          n.x.toFixed(2) +
          '" y2="' +
          n.y.toFixed(2) +
          '"/>' +
          '<line class="dir-graph-seg" x1="' +
          n.x.toFixed(2) +
          '" y1="' +
          n.y.toFixed(2) +
          '" x2="' +
          next.x.toFixed(2) +
          '" y2="' +
          next.y.toFixed(2) +
          '"/>' +
          "</g>"
        );
      })
      .join("");

    var nodes = positions
      .map(function (n) {
        var labelDist = ORBIT + NODE_R + 32;
        var lp = polarToXY(n.angle, labelDist);
        var anchor = labelAnchor(n.angle);
        var extraX = labelOffsetX(n.angle);
        var iconSize = 18;
        var scale = iconSize / 24;
        var tx = n.x - iconSize / 2;
        var ty = n.y + 2 - iconSize / 2;
        var accent = accentColor(n.index);

        return (
          '<g class="dir-graph-node" data-node="' +
          n.id +
          '" data-accent="' +
          accent +
          '" tabindex="0" role="button" aria-label="' +
          NODES[n.index].label +
          ": " +
          NODES[n.index].desc +
          '">' +
          '<circle class="dir-graph-node-glow" cx="' +
          n.x.toFixed(2) +
          '" cy="' +
          n.y.toFixed(2) +
          '" r="52" fill="url(#dir-graph-node-glow-' +
          (n.index % 2 === 0 ? "blue" : "amber") +
          ')"/>' +
          '<circle class="dir-graph-node-ring" cx="' +
          n.x.toFixed(2) +
          '" cy="' +
          n.y.toFixed(2) +
          '" r="' +
          (NODE_R + 5) +
          '"/>' +
          '<circle class="dir-graph-node-fill" cx="' +
          n.x.toFixed(2) +
          '" cy="' +
          n.y.toFixed(2) +
          '" r="' +
          NODE_R +
          '"/>' +
          '<circle class="dir-graph-node-stroke" cx="' +
          n.x.toFixed(2) +
          '" cy="' +
          n.y.toFixed(2) +
          '" r="' +
          NODE_R +
          '"/>' +
          '<g class="dir-graph-node-icon" transform="translate(' +
          tx.toFixed(2) +
          "," +
          ty.toFixed(2) +
          ") scale(" +
          scale.toFixed(4) +
          ')" fill="none" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">' +
          ICON_PATHS[n.id] +
          "</g>" +
          '<text class="dir-graph-node-label" x="' +
          (lp.x + extraX).toFixed(2) +
          '" y="' +
          (lp.y + 4).toFixed(2) +
          '" text-anchor="' +
          anchor +
          '">' +
          n.label +
          "</text>" +
          "</g>"
        );
      })
      .join("");

    return (
      '<svg class="dir-graph-root" viewBox="0 0 800 680" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">' +
      "<defs>" +
      '<radialGradient id="dir-graph-hub-glow" cx="50%" cy="50%" r="50%">' +
      '<stop offset="0%" stop-color="' +
      BLUE +
      '" stop-opacity="0.18"/>' +
      '<stop offset="100%" stop-color="' +
      BLUE +
      '" stop-opacity="0"/>' +
      "</radialGradient>" +
      '<radialGradient id="dir-graph-node-glow-blue" cx="50%" cy="50%" r="50%">' +
      '<stop offset="0%" stop-color="' +
      BLUE +
      '" stop-opacity="0.18"/>' +
      '<stop offset="100%" stop-color="' +
      BLUE +
      '" stop-opacity="0"/>' +
      "</radialGradient>" +
      '<radialGradient id="dir-graph-node-glow-amber" cx="50%" cy="50%" r="50%">' +
      '<stop offset="0%" stop-color="' +
      AMBER +
      '" stop-opacity="0.18"/>' +
      '<stop offset="100%" stop-color="' +
      AMBER +
      '" stop-opacity="0"/>' +
      "</radialGradient>" +
      "</defs>" +
      buildDotGrid() +
      '<circle cx="' +
      CX +
      '" cy="' +
      CY +
      '" r="310" fill="url(#dir-graph-hub-glow)"/>' +
      '<circle class="dir-graph-orbit" cx="' +
      CX +
      '" cy="' +
      CY +
      '" r="' +
      ORBIT +
      '"/>' +
      lines +
      buildParticles(positions, animate) +
      '<circle cx="' +
      CX +
      '" cy="' +
      CY +
      '" r="60" fill="url(#dir-graph-hub-glow)"/>' +
      '<circle class="dir-graph-hub-fill" cx="' +
      CX +
      '" cy="' +
      CY +
      '" r="50"/>' +
      '<circle class="dir-graph-hub-ring" cx="' +
      CX +
      '" cy="' +
      CY +
      '" r="54"/>' +
      '<text class="dir-graph-hub-text" text-anchor="middle">' +
      '<tspan x="' +
      CX +
      '" y="' +
      (CY - 8) +
      '">AGENT</tspan>' +
      '<tspan x="' +
      CX +
      '" dy="11">DIRECTORY</tspan>' +
      '<tspan x="' +
      CX +
      '" dy="11">SERVICE</tspan>' +
      "</text>" +
      nodes +
      "</svg>"
    );
  }

  function renderPanel(panel, nodeId) {
    if (!nodeId) {
      panel.hidden = true;
      panel.innerHTML = "";
      return;
    }

    var node = NODES.find(function (n) {
      return n.id === nodeId;
    });
    if (!node) {
      panel.hidden = true;
      return;
    }

    var index = NODES.indexOf(node);
    var accent = accentColor(index);
    var icon = ICON_PATHS[node.id];

    panel.hidden = false;
    panel.innerHTML =
      '<div class="dir-graph-panel__badge" style="color:' +
      accent +
      ";border-color:" +
      accent +
      "55;background:" +
      accent +
      '10">' +
      node.group +
      "</div>" +
      '<div class="dir-graph-panel__header">' +
      '<span class="dir-graph-panel__icon" style="color:' +
      accent +
      ";border-color:" +
      accent +
      "44;background:" +
      accent +
      '15">' +
      '<svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">' +
      icon +
      "</svg>" +
      "</span>" +
      '<span class="dir-graph-panel__title" style="color:' +
      accent +
      '">' +
      node.label +
      "</span>" +
      "</div>" +
      '<div class="dir-graph-panel__divider"></div>' +
      '<p class="dir-graph-panel__desc">' +
      node.desc +
      "</p>";
  }

  function setActive(wrap, nodeId) {
    var activeId = nodeId || null;
    wrap.querySelectorAll(".dir-graph-node").forEach(function (el) {
      el.classList.toggle("is-active", el.getAttribute("data-node") === activeId);
    });
    wrap.querySelectorAll(".dir-graph-spoke").forEach(function (el) {
      el.classList.toggle("is-active", el.getAttribute("data-node") === activeId);
    });
    renderPanel(wrap.querySelector(".dir-graph-panel"), activeId);
  }

  document.querySelectorAll(".dir-graph-wrap[data-dir-graph]").forEach(function (wrap) {
    if (wrap.dataset.graphInit === "true") {
      return;
    }
    wrap.dataset.graphInit = "true";

    var reduceMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    var pinned = null;

    wrap.innerHTML =
      '<div class="dir-graph-layout">' +
      '<div class="dir-graph-stage">' +
      buildSvg(!reduceMotion) +
      "</div>" +
      '<div class="dir-graph-panel" hidden></div>' +
      "</div>" +
      '<p class="dir-graph-hint">Hover or tap nodes to explore the lifecycle</p>';

    var panel = wrap.querySelector(".dir-graph-panel");

    function activate(nodeId) {
      if (pinned) {
        pinned = pinned === nodeId ? null : nodeId;
      } else {
        pinned = nodeId;
      }
      setActive(wrap, pinned);
    }

    wrap.querySelectorAll(".dir-graph-node").forEach(function (nodeEl) {
      var nodeId = nodeEl.getAttribute("data-node");

      nodeEl.addEventListener("mouseenter", function () {
        if (!pinned) {
          setActive(wrap, nodeId);
        }
      });

      nodeEl.addEventListener("mouseleave", function () {
        if (!pinned) {
          setActive(wrap, null);
        }
      });

      nodeEl.addEventListener("focus", function () {
        setActive(wrap, nodeId);
      });

      nodeEl.addEventListener("blur", function () {
        if (!pinned) {
          setActive(wrap, null);
        }
      });

      nodeEl.addEventListener("click", function () {
        activate(nodeId);
      });

      nodeEl.addEventListener("keydown", function (event) {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          activate(nodeId);
        }
        if (event.key === "Escape") {
          pinned = null;
          setActive(wrap, null);
          nodeEl.blur();
        }
      });
    });

    wrap.addEventListener("mouseleave", function () {
      if (!pinned) {
        setActive(wrap, null);
      }
    });
  });
});
