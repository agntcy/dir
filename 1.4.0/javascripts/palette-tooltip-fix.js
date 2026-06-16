/* Copyright AGNTCY Contributors (https://github.com/agntcy) */
/* SPDX-License-Identifier: Apache-2.0 */

/* Palette toggle: dismiss stuck Material tooltip2 popups on touch (no mouseleave). */
document$.subscribe(function () {
  var palette = document.querySelector('[data-md-component="palette"]');
  if (!palette || palette.dataset.paletteTooltipFix === "true") {
    return;
  }
  palette.dataset.paletteTooltipFix = "true";

  function dismissPaletteTooltips() {
    palette.querySelectorAll("label").forEach(function (label) {
      var tipId = label.getAttribute("aria-describedby");
      if (tipId) {
        var tip = document.getElementById(tipId);
        if (tip) {
          tip.classList.remove("md-tooltip2--active");
          tip.remove();
        }
        label.removeAttribute("aria-describedby");
      }
      label.blur();
    });
  }

  function scheduleDismiss() {
    dismissPaletteTooltips();
    window.requestAnimationFrame(dismissPaletteTooltips);
    window.setTimeout(dismissPaletteTooltips, 300);
  }

  palette.addEventListener("change", scheduleDismiss, true);
  palette.addEventListener("click", scheduleDismiss, true);

  if (
    window.matchMedia("(hover: none)").matches ||
    window.matchMedia("(pointer: coarse)").matches
  ) {
    document.addEventListener(
      "touchstart",
      function (event) {
        if (!palette.contains(event.target)) {
          scheduleDismiss();
        }
      },
      { passive: true },
    );
  }
});
