/* Copyright AGNTCY Contributors (https://github.com/agntcy) */
/* SPDX-License-Identifier: Apache-2.0 */

/* Hero tagline: typewriter cycle between framework / protocol / registry. */
document$.subscribe(function () {
  var reduceMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  var pauseMs = 1200;
  var typeMs = 75;
  var deleteMs = 45;
  var staticIntervalMs = 2800;
  var easterEggVisit = 7;
  var easterEggTypo = "franewo";
  var easterEggPrefix = "fra";
  var easterEggTypoPauseMs = 350;

  document.querySelectorAll(".dir-hero__flip[data-words]").forEach(function (el) {
    if (el.dataset.flipInit === "true") {
      return;
    }
    el.dataset.flipInit = "true";

    var words = el.getAttribute("data-words").split(",").map(function (word) {
      return word.trim();
    });
    if (words.length < 2) {
      return;
    }

    var frameworkIndex = words.indexOf("framework");
    var wordIndex = 0;
    var charIndex = words[0].length;
    var deleting = false;
    var paused = true;
    var frameworkVisitCount = 1;
    var easterEggDone = false;
    /* null | "typo" | "rewind" | "finish" */
    var easterEggPhase = null;

    el.textContent = words[0];

    if (reduceMotion) {
      window.setInterval(function () {
        wordIndex = (wordIndex + 1) % words.length;
        el.textContent = words[wordIndex];
      }, staticIntervalMs);
      return;
    }

    function beginEasterEgg() {
      easterEggPhase = "typo";
      charIndex = 0;
      el.textContent = "";
    }

    function shouldTriggerEasterEgg() {
      return (
        !easterEggDone &&
        frameworkIndex !== -1 &&
        wordIndex === frameworkIndex &&
        frameworkVisitCount === easterEggVisit - 1
      );
    }

    function onFrameworkComplete() {
      frameworkVisitCount += 1;
      if (easterEggPhase === "finish") {
        easterEggPhase = null;
        easterEggDone = true;
      }
    }

    function step() {
      var word = words[wordIndex];

      if (paused) {
        paused = false;
        if (easterEggPhase === "rewind") {
          window.setTimeout(step, 0);
          return;
        }
        deleting = true;
        window.setTimeout(step, pauseMs);
        return;
      }

      if (deleting) {
        if (charIndex > 0) {
          charIndex -= 1;
          el.textContent = word.slice(0, charIndex);
          window.setTimeout(step, deleteMs);
          return;
        }

        wordIndex = (wordIndex + 1) % words.length;
        deleting = false;
        if (shouldTriggerEasterEgg()) {
          beginEasterEgg();
        }
        window.setTimeout(step, typeMs);
        return;
      }

      if (easterEggPhase === "typo") {
        if (charIndex < easterEggTypo.length) {
          charIndex += 1;
          el.textContent = easterEggTypo.slice(0, charIndex);
          window.setTimeout(step, typeMs);
          return;
        }

        paused = true;
        easterEggPhase = "rewind";
        window.setTimeout(step, easterEggTypoPauseMs);
        return;
      }

      if (easterEggPhase === "rewind") {
        if (charIndex > easterEggPrefix.length) {
          charIndex -= 1;
          el.textContent = easterEggTypo.slice(0, charIndex);
          window.setTimeout(step, deleteMs);
          return;
        }

        charIndex = easterEggPrefix.length;
        el.textContent = easterEggPrefix;
        easterEggPhase = "finish";
        window.setTimeout(step, typeMs);
        return;
      }

      word = words[wordIndex];
      if (charIndex < word.length) {
        charIndex += 1;
        el.textContent = word.slice(0, charIndex);
        window.setTimeout(step, typeMs);
        return;
      }

      if (wordIndex === frameworkIndex) {
        onFrameworkComplete();
      }

      paused = true;
      window.setTimeout(step, pauseMs);
    }

    window.setTimeout(step, pauseMs);
  });
});
