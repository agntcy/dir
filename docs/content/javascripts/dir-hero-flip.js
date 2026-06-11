/* Copyright AGNTCY Contributors (https://github.com/agntcy) */
/* SPDX-License-Identifier: Apache-2.0 */

/* Hero tagline: typewriter cycle between framework / protocol / registry. */
document$.subscribe(function () {
  var reduceMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  var pauseMs = 2000;
  var typeMs = 75;
  var deleteMs = 45;
  var staticIntervalMs = 2800;

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

    var wordIndex = 0;
    var charIndex = words[0].length;
    var deleting = false;
    var paused = true;

    el.textContent = words[0];

    if (reduceMotion) {
      window.setInterval(function () {
        wordIndex = (wordIndex + 1) % words.length;
        el.textContent = words[wordIndex];
      }, staticIntervalMs);
      return;
    }

    function step() {
      var word = words[wordIndex];

      if (paused) {
        paused = false;
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

      paused = true;
      window.setTimeout(step, pauseMs);
    }

    window.setTimeout(step, pauseMs);
  });
});
