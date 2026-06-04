/* Grid card enhancements: featured highlight and single-link stretch targets. */
document$.subscribe(function () {
  document.querySelectorAll(".md-typeset .grid.cards > ul > li").forEach(function (li) {
    var links = li.querySelectorAll("a[href]");
    li.classList.toggle("card-single-link", links.length === 1);

    var title = li.querySelector("strong");
    if (title && title.textContent.trim() === "Join the Federation Testbed") {
      li.classList.add("card-featured");
    }
  });
});
