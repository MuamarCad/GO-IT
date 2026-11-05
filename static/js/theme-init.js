/* Подключается в <head> без defer: применяет тему до первой отрисовки, чтобы не было вспышки. */
(function () {
  var root = document.documentElement;
  root.classList.add("js");
  try {
    var theme = localStorage.getItem("goit-theme");
    if (theme === "light" || theme === "dark") root.setAttribute("data-theme", theme);
    if (localStorage.getItem("goit-contrast") === "high") root.setAttribute("data-contrast", "high");
  } catch (e) { /* localStorage недоступен — остаются системные настройки */ }
})();
