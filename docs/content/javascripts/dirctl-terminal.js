/* Copyright AGNTCY Contributors (https://github.com/agntcy) */
/* SPDX-License-Identifier: Apache-2.0 */

/* Home-page dirctl terminal: scripted demo + limited try-it shell. */
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
    var demoBtn = section.querySelector('[data-mode-switch="demo"]');
    var reopenBtn = section.querySelector(".dirctl-terminal-reopen");

    if (!terminal || !output) {
      return null;
    }

    section.setAttribute("data-dirctl-bound", "1");

    var state = {
      mode: "demo",
      daemonRunning: false,
      lastCid: getData().demoCid || "",
      published: false,
      tryHistory: [],
      demoTimer: null,
      demoObserver: null,
      demoRunning: false,
    };

    function clearDemoTimer() {
      if (state.demoTimer) {
        clearTimeout(state.demoTimer);
        state.demoTimer = null;
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
        tryBtn.hidden = true;
        demoBtn.hidden = false;
        input.focus();
        appendTryOutput("Type a command or enter help for suggestions.", "muted");
      } else {
        inputForm.hidden = true;
        tryBtn.hidden = false;
        demoBtn.hidden = true;
        output.innerHTML = "";
        startDemoObserver();
      }
    }

    function appendTryOutput(text, kind) {
      var className = "dirctl-terminal-out";
      if (kind === "muted") {
        className = "dirctl-terminal-muted";
      } else if (kind === "err") {
        className = "dirctl-terminal-err";
      }
      output.innerHTML += '<span class="' + className + '">' + escapeHtml(text) + "</span>\n";
      output.scrollTop = output.scrollHeight;
    }

    function appendTryCommand(line) {
      output.innerHTML +=
        '<span class="dirctl-terminal-cmd">$ ' + escapeHtml(line) + "</span>\n";
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
        appendTryOutput("error: Directory daemon is not running. Try: dirctl daemon start", "err");
        return false;
      }
      return true;
    }

    function handleTryCommand(line) {
      var trimmed = line.trim();
      if (!trimmed) {
        return;
      }

      appendTryCommand(trimmed);
      state.tryHistory.push(trimmed);

      var lower = trimmed.toLowerCase();
      var data = getData();

      if (lower === "help") {
        appendTryOutput(data.helpText || "Enter a dirctl command.");
        return;
      }
      if (lower === "clear") {
        output.innerHTML = "";
        return;
      }

      var tokens = tokenize(trimmed);
      if (tokens[0] !== "dirctl") {
        appendTryOutput("command not found: " + trimmed, "err");
        return;
      }

      if (tokens.length === 1 || (tokens.length === 2 && tokens[1] === "--help")) {
        appendTryOutput(data.dirctlHelp || "dirctl help");
        return;
      }

      var sub = tokens[1];

      if (sub === "daemon" && tokens[2] === "start") {
        state.daemonRunning = true;
        appendTryOutput("Directory daemon listening on localhost:8888");
        return;
      }

      if (sub === "push") {
        if (!requireDaemon()) {
          return;
        }
        state.lastCid = data.demoCid || state.lastCid;
        var rawOutput = tokens.indexOf("--output") !== -1 && tokens[tokens.indexOf("--output") + 1] === "raw";
        appendTryOutput(rawOutput ? state.lastCid : "Stored record: " + state.lastCid);
        return;
      }

      if (sub === "routing") {
        if (tokens[2] === "--help") {
          appendTryOutput(data.routingHelp || "dirctl routing help");
          return;
        }
        if (!requireDaemon()) {
          return;
        }
        if (tokens[2] === "publish") {
          var publishCid = tokens[3];
          if (!publishCid) {
            appendTryOutput("error: CID required. Usage: dirctl routing publish <cid>", "err");
            return;
          }
          if (publishCid !== state.lastCid && publishCid !== data.demoCid) {
            appendTryOutput("error: record not found in local store: " + publishCid, "err");
            return;
          }
          state.published = true;
          appendTryOutput("Published record to routing network.");
          return;
        }
        if (tokens[2] === "list") {
          if (!state.published) {
            appendTryOutput("(no published records)");
            return;
          }
          appendTryOutput(data.routingList || state.lastCid);
          return;
        }
        if (tokens[2] === "search") {
          var skillIdx = tokens.indexOf("--skill");
          if (skillIdx === -1) {
            appendTryOutput("error: --skill is required", "err");
            return;
          }
          appendTryOutput(data.routingSearch || "No records found.");
          return;
        }
        appendTryOutput("dirctl: unknown routing command. Try: dirctl routing --help", "err");
        return;
      }

      if (sub === "pull") {
        if (!requireDaemon()) {
          return;
        }
        var pullCid = tokens[2];
        if (!pullCid) {
          appendTryOutput("error: CID required. Usage: dirctl pull <cid>", "err");
          return;
        }
        if (pullCid !== state.lastCid && pullCid !== data.demoCid) {
          appendTryOutput("error: record not found: " + pullCid, "err");
          return;
        }
        appendTryOutput(data.pullRecord || "{}");
        return;
      }

      appendTryOutput("dirctl: unknown command \"" + sub + "\". Try: dirctl --help", "err");
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

      var script = getData().demoScript || [];
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

        if (step.type === "command") {
          if (prefersReducedMotion()) {
            doneLines.push(
              '<span class="dirctl-terminal-cmd">$ ' + escapeHtml(step.text) + "</span>"
            );
            renderDemoBlock(doneLines, null);
            finishStep();
            return;
          }

          var prefix = '<span class="dirctl-terminal-cmd">$ </span>';
          var text = step.text;
          var charIndex = 0;

          function typeChar() {
            if (state.mode !== "demo") {
              return;
            }
            var partial =
              prefix + '<span class="dirctl-terminal-cmd">' + escapeHtml(text.slice(0, charIndex)) + "</span>";
            renderDemoBlock(doneLines, partial);
            if (charIndex <= text.length) {
              charIndex++;
              state.demoTimer = setTimeout(typeChar, TYPE_MS);
            } else {
              doneLines.push(
                '<span class="dirctl-terminal-cmd">$ ' + escapeHtml(text) + "</span>"
              );
              finishStep();
            }
          }

          typeChar();
        }
      }

      if (prefersReducedMotion()) {
        script.forEach(function (step) {
          if (step.type === "command") {
            doneLines.push(
              '<span class="dirctl-terminal-cmd">$ ' + escapeHtml(step.text) + "</span>"
            );
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

    terminal.querySelectorAll("[data-term]").forEach(function (btn) {
      btn.addEventListener("click", function () {
        var action = btn.getAttribute("data-term");
        if (action === "min") {
          terminal.classList.toggle("dirctl-terminal-min");
        } else if (action === "close") {
          clearDemoTimer();
          terminal.classList.add("dirctl-terminal-closed");
          terminal.classList.remove("dirctl-terminal-min");
          if (reopenBtn) {
            reopenBtn.classList.add("show");
            reopenBtn.hidden = false;
          }
        }
      });
    });

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
        setMode("try");
      });
    }

    if (demoBtn) {
      demoBtn.addEventListener("click", function () {
        setMode("demo");
      });
    }

    if (inputForm && input) {
      inputForm.addEventListener("submit", function (event) {
        event.preventDefault();
        var value = input.value;
        input.value = "";
        handleTryCommand(value);
      });
    }

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
