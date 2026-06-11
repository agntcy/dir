/* Copyright AGNTCY Contributors (https://github.com/agntcy) */
/* SPDX-License-Identifier: Apache-2.0 */

/* Home-page dirctl terminal: scripted demos + dirctl try-it shell. */
(function () {
  var CURSOR = '<span class="dirctl-terminal-cursor">|</span>';
  var TYPE_MS = 32;
  function escapeHtml(text) {
    return text
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;");
  }

  function prefersReducedMotion() {
    return (
      window.matchMedia &&
      window.matchMedia("(prefers-reduced-motion: reduce)").matches
    );
  }

  function getData() {
    return window.DirctlDemoData || {};
  }

  function createTerminal(section) {
    if (section.getAttribute("data-dirctl-bound") === "1") {
      return null;
    }

    var terminal = section.querySelector(".dirctl-terminal");
    var output = section.querySelector(".dirctl-terminal-output");
    var inputForm = section.querySelector(".dirctl-terminal-input");
    var input = section.querySelector(".dirctl-terminal-command");
    var tryBtn = section.querySelector('[data-mode-switch="try"]');
    var reopenBtn = section.querySelector(".dirctl-terminal-reopen");
    var titleEl = section.querySelector(".dirctl-terminal-title");
    var promptEl = section.querySelector(".dirctl-terminal-prompt");
    var introEls = section.querySelectorAll(".dirctl-terminal-intro[data-intro-level]");

    if (!terminal || !output) {
      return null;
    }

    section.setAttribute("data-dirctl-bound", "1");

    var state = {
      mode: "demo",
      demoLevel: "agent",
      daemonRunning: false,
      lastCid: getData().demoCid || "",
      published: false,
      tryHistory: [],
      demoTimer: null,
      demoObserver: null,
      demoRunning: false,
    };

    var demoLineMeta = {
      command: { prefix: "$ ", className: "dirctl-terminal-cmd" },
      user: { prefix: "> ", className: "dirctl-terminal-user" },
      agent: { prefix: "● ", className: "dirctl-terminal-agent" },
      tool: { prefix: "→ ", className: "dirctl-terminal-tool" },
    };

    function getActiveScript() {
      var data = getData();
      if (state.demoLevel === "agent") {
        return data.agentDemoScript || [];
      }
      return data.demoScript || [];
    }

    function updateActionChrome() {
      var activeLevel = state.mode === "try" ? "try" : state.demoLevel;
      introEls.forEach(function (el) {
        el.hidden = el.getAttribute("data-intro-level") !== activeLevel;
      });
      section.querySelectorAll("[data-demo-level]").forEach(function (btn) {
        btn.classList.toggle(
          "is-active",
          state.mode === "demo" &&
            btn.getAttribute("data-demo-level") === state.demoLevel
        );
      });
      if (tryBtn) {
        tryBtn.classList.toggle("is-active", state.mode === "try");
      }
    }

    function updateDemoChrome() {
      var data = getData();
      var titles = data.demoTitles || {};
      if (titleEl) {
        titleEl.textContent = titles[state.demoLevel] || titles.cli || "user@dir:~";
      }
      updateActionChrome();
    }

    function formatDemoLine(step) {
      var meta = demoLineMeta[step.type];
      if (!meta) {
        return null;
      }
      return (
        '<span class="' + meta.className + '">' + meta.prefix + escapeHtml(step.text) + "</span>"
      );
    }

    function clearDemoTimer() {
      if (state.demoTimer) {
        clearTimeout(state.demoTimer);
        state.demoTimer = null;
      }
    }

    function updateTryChrome() {
      var data = getData();
      var titles = data.demoTitles || {};
      if (titleEl) {
        titleEl.textContent = titles.cli || "user@dir:~";
      }
      if (promptEl) {
        promptEl.textContent = "user@dir:~$";
      }
      if (input) {
        input.setAttribute("aria-label", "Enter a dirctl command");
      }
    }

    function setMode(mode) {
      clearDemoTimer();
      state.mode = mode;
      terminal.setAttribute("data-mode", mode);

      if (mode === "try") {
        state.demoRunning = false;
        if (state.demoObserver) {
          state.demoObserver.disconnect();
          state.demoObserver = null;
        }
        output.innerHTML = "";
        inputForm.hidden = false;
        updateTryChrome();
        updateActionChrome();
        input.focus();
        appendTryLine("Type a command or enter help for suggestions.", "muted");
      } else {
        inputForm.hidden = true;
        output.innerHTML = "";
        updateDemoChrome();
        startDemoObserver();
      }
    }

    function setDemoLevel(level) {
      if (state.mode === "try") {
        state.demoLevel = level;
        setMode("demo");
        return;
      }
      if (state.demoLevel === level) {
        return;
      }
      state.demoLevel = level;
      updateDemoChrome();
      resetDemo();
    }

    function appendTryLine(text, kind) {
      var className = "dirctl-terminal-out";
      var prefix = "";
      if (kind === "muted") {
        className = "dirctl-terminal-muted";
      } else if (kind === "err") {
        className = "dirctl-terminal-err";
      } else if (kind === "command") {
        className = "dirctl-terminal-cmd";
        prefix = "$ ";
      }
      output.innerHTML +=
        '<span class="' + className + '">' + prefix + escapeHtml(text) + "</span>\n";
      output.scrollTop = output.scrollHeight;
    }

    function tokenize(line) {
      var tokens = [];
      var current = "";
      var inQuote = false;
      var quote = "";

      for (var i = 0; i < line.length; i++) {
        var ch = line[i];
        if (inQuote) {
          if (ch === quote) {
            inQuote = false;
            tokens.push(current);
            current = "";
          } else {
            current += ch;
          }
          continue;
        }
        if (ch === '"' || ch === "'") {
          inQuote = true;
          quote = ch;
          continue;
        }
        if (/\s/.test(ch)) {
          if (current) {
            tokens.push(current);
            current = "";
          }
          continue;
        }
        current += ch;
      }
      if (current) {
        tokens.push(current);
      }
      return tokens;
    }

    function requireDaemon() {
      if (!state.daemonRunning) {
        appendTryLine("error: Directory daemon is not running. Try: dirctl daemon start", "err");
        return false;
      }
      return true;
    }

    function handleCliTryInput(line) {
      var trimmed = line.trim();
      var lower = trimmed.toLowerCase();
      var data = getData();

      appendTryLine(trimmed, "command");
      state.tryHistory.push(trimmed);

      if (lower === "help") {
        appendTryLine(data.helpText || "Enter a dirctl command.");
        return;
      }
      if (lower === "clear") {
        output.innerHTML = "";
        return;
      }

      var tokens = tokenize(trimmed);
      if (tokens[0] !== "dirctl") {
        appendTryLine("command not found: " + trimmed, "err");
        return;
      }

      if (tokens.length === 1 || (tokens.length === 2 && tokens[1] === "--help")) {
        appendTryLine(data.dirctlHelp || "dirctl help");
        return;
      }

      var sub = tokens[1];

      if (sub === "daemon" && tokens[2] === "start") {
        state.daemonRunning = true;
        appendTryLine("Directory daemon listening on localhost:8888");
        return;
      }

      if (sub === "push") {
        if (!requireDaemon()) {
          return;
        }
        state.lastCid = data.demoCid || state.lastCid;
        var rawOutput = tokens.indexOf("--output") !== -1 && tokens[tokens.indexOf("--output") + 1] === "raw";
        appendTryLine(rawOutput ? state.lastCid : "Stored record: " + state.lastCid);
        return;
      }

      if (sub === "routing") {
        if (tokens[2] === "--help") {
          appendTryLine(data.routingHelp || "dirctl routing help");
          return;
        }
        if (!requireDaemon()) {
          return;
        }
        if (tokens[2] === "publish") {
          var publishCid = tokens[3];
          if (!publishCid) {
            appendTryLine("error: CID required. Usage: dirctl routing publish <cid>", "err");
            return;
          }
          if (publishCid !== state.lastCid && publishCid !== data.demoCid) {
            appendTryLine("error: record not found in local store: " + publishCid, "err");
            return;
          }
          state.published = true;
          appendTryLine("Published record to routing network.");
          return;
        }
        if (tokens[2] === "list") {
          if (!state.published) {
            appendTryLine("(no published records)");
            return;
          }
          appendTryLine(data.routingList || state.lastCid);
          return;
        }
        if (tokens[2] === "search") {
          var skillIdx = tokens.indexOf("--skill");
          if (skillIdx === -1) {
            appendTryLine("error: --skill is required", "err");
            return;
          }
          appendTryLine(data.routingSearch || "No records found.");
          return;
        }
        appendTryLine("dirctl: unknown routing command. Try: dirctl routing --help", "err");
        return;
      }

      if (sub === "pull") {
        if (!requireDaemon()) {
          return;
        }
        var pullCid = tokens[2];
        if (!pullCid) {
          appendTryLine("error: CID required. Usage: dirctl pull <cid>", "err");
          return;
        }
        if (pullCid !== state.lastCid && pullCid !== data.demoCid) {
          appendTryLine("error: record not found: " + pullCid, "err");
          return;
        }
        appendTryLine(data.pullRecord || "{}");
        return;
      }

      appendTryLine("dirctl: unknown command \"" + sub + "\". Try: dirctl --help", "err");
    }

    function handleTryInput(line) {
      if (!line.trim()) {
        return;
      }
      handleCliTryInput(line);
    }

    function renderDemoBlock(doneLines, partial) {
      var html = doneLines.join("\n");
      if (partial) {
        html += (doneLines.length ? "\n" : "") + partial + CURSOR;
      } else if (state.demoRunning) {
        html += (doneLines.length ? "\n" : "") + CURSOR;
      }
      output.innerHTML = html;
      output.scrollTop = output.scrollHeight;
    }

    function runDemoScript() {
      if (state.mode !== "demo" || terminal.classList.contains("dirctl-terminal-closed")) {
        return;
      }

      var script = getActiveScript();
      var doneLines = [];
      var stepIndex = 0;
      state.demoRunning = true;

      function finishStep() {
        stepIndex++;
        runStep();
      }

      function runStep() {
        if (state.mode !== "demo") {
          state.demoRunning = false;
          return;
        }

        if (stepIndex >= script.length) {
          state.demoTimer = setTimeout(function () {
            doneLines = [];
            stepIndex = 0;
            runStep();
          }, 0);
          return;
        }

        var step = script[stepIndex];

        if (step.type === "pause") {
          state.demoTimer = setTimeout(finishStep, step.ms || 1500);
          return;
        }

        if (step.type === "output") {
          doneLines.push(
            '<span class="dirctl-terminal-out">' + escapeHtml(step.text) + "</span>"
          );
          renderDemoBlock(doneLines, null);
          state.demoTimer = setTimeout(finishStep, prefersReducedMotion() ? 0 : 400);
          return;
        }

        if (demoLineMeta[step.type]) {
          var lineMeta = demoLineMeta[step.type];
          if (prefersReducedMotion()) {
            doneLines.push(formatDemoLine(step));
            renderDemoBlock(doneLines, null);
            finishStep();
            return;
          }

          var prefixHtml = '<span class="' + lineMeta.className + '">' + lineMeta.prefix + "</span>";
          var text = step.text;
          var charIndex = 0;

          function typeChar() {
            if (state.mode !== "demo") {
              return;
            }
            var partial =
              prefixHtml +
              '<span class="' + lineMeta.className + '">' + escapeHtml(text.slice(0, charIndex)) + "</span>";
            renderDemoBlock(doneLines, partial);
            if (charIndex <= text.length) {
              charIndex++;
              state.demoTimer = setTimeout(typeChar, TYPE_MS);
            } else {
              doneLines.push(formatDemoLine(step));
              finishStep();
            }
          }

          typeChar();
        }
      }

      if (prefersReducedMotion()) {
        script.forEach(function (step) {
          if (demoLineMeta[step.type]) {
            doneLines.push(formatDemoLine(step));
          } else if (step.type === "output") {
            doneLines.push(
              '<span class="dirctl-terminal-out">' + escapeHtml(step.text) + "</span>"
            );
          }
        });
        renderDemoBlock(doneLines, null);
        state.demoRunning = false;
        return;
      }

      runStep();
    }

    function resetDemo() {
      clearDemoTimer();
      output.innerHTML = "";
      runDemoScript();
    }

    function startDemoObserver() {
      if (state.demoObserver) {
        state.demoObserver.disconnect();
      }
      if (!("IntersectionObserver" in window)) {
        resetDemo();
        return;
      }
      state.demoObserver = new IntersectionObserver(
        function (entries) {
          entries.forEach(function (entry) {
            if (entry.isIntersecting && state.mode === "demo") {
              resetDemo();
            }
          });
        },
        { threshold: 0.45 }
      );
      state.demoObserver.observe(output);
    }

    if (reopenBtn) {
      reopenBtn.addEventListener("click", function () {
        terminal.classList.remove("dirctl-terminal-closed", "dirctl-terminal-min");
        reopenBtn.classList.remove("show");
        reopenBtn.hidden = true;
        if (state.mode === "demo") {
          resetDemo();
        } else {
          input.focus();
        }
      });
    }

    if (tryBtn) {
      tryBtn.addEventListener("click", function () {
        state.demoLevel = "cli";
        setMode("try");
      });
    }

    section.querySelectorAll("[data-demo-level]").forEach(function (btn) {
      btn.addEventListener("click", function () {
        setDemoLevel(btn.getAttribute("data-demo-level") || "cli");
      });
    });

    if (inputForm && input) {
      inputForm.addEventListener("submit", function (event) {
        event.preventDefault();
        var value = input.value;
        input.value = "";
        handleTryInput(value);
      });
    }

    updateDemoChrome();
    setMode("demo");

    return {
      destroy: function () {
        clearDemoTimer();
        if (state.demoObserver) {
          state.demoObserver.disconnect();
        }
        section.removeAttribute("data-dirctl-bound");
      },
    };
  }

  function initTerminals() {
    document.querySelectorAll(".dirctl-terminal-section").forEach(function (section) {
      createTerminal(section);
    });
  }

  if (typeof document$ !== "undefined") {
    document$.subscribe(initTerminals);
  } else {
    document.addEventListener("DOMContentLoaded", initTerminals);
  }
})();
