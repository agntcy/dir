/* Copyright AGNTCY Contributors (https://github.com/agntcy) */
/* SPDX-License-Identifier: Apache-2.0 */

/* Hero tagline: horizontal flip between framework / protocol / registry. */
document$.subscribe(function () {
  var reduceMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  var intervalMs = 2800;

  document.querySelectorAll(".dir-hero__flip[data-words]").forEach(function (el) {
    if (el.dataset.flipInit === "true") {
      return;
    }
    el.dataset.flipInit = "true";

    var words = el.getAttribute("data-words").split(",");
    var index = 0;
    var flipping = false;

    if (words.length < 2) {
      return;
    }

    function advanceWord() {
      if (flipping) {
        return;
      }

      var nextIndex = (index + 1) % words.length;

      if (reduceMotion) {
        index = nextIndex;
        el.textContent = words[index];
        return;
      }

      flipping = true;
      el.classList.add("is-flipping-out");

      el.addEventListener(
        "animationend",
        function onFlipOut(event) {
          if (event.animationName !== "dir-hero-flip-out") {
            return;
          }
          el.removeEventListener("animationend", onFlipOut);
          el.classList.remove("is-flipping-out");

          index = nextIndex;
          el.textContent = words[index];
          el.classList.add("is-flipping-in");

          el.addEventListener(
            "animationend",
            function onFlipIn(eventIn) {
              if (eventIn.animationName !== "dir-hero-flip-in") {
                return;
              }
              el.removeEventListener("animationend", onFlipIn);
              el.classList.remove("is-flipping-in");
              flipping = false;
            },
            { once: true }
          );
        },
        { once: true }
      );
    }

    window.setInterval(advanceWord, intervalMs);
  });
});
