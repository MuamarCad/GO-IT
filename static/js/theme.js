/* GO-IT: тема, контраст, меню, защита от двойной отправки, чек-лист roadmap. Без зависимостей. */
(function () {
  var root = document.documentElement;
  var store = {
    get: function (k) { try { return localStorage.getItem(k); } catch (e) { return null; } },
    set: function (k, v) { try { localStorage.setItem(k, v); } catch (e) {} }
  };

  function effectiveTheme() {
    var t = root.getAttribute("data-theme");
    if (t) return t;
    return window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
  }

  function syncButtons() {
    var dark = effectiveTheme() === "dark";
    var high = root.getAttribute("data-contrast") === "high";
    document.querySelectorAll("[data-theme-toggle]").forEach(function (b) {
      b.setAttribute("aria-pressed", String(dark));
      b.setAttribute("aria-label", dark ? "Тёмная тема включена. Переключить на светлую" : "Светлая тема включена. Переключить на тёмную");
    });
    document.querySelectorAll("[data-contrast-toggle]").forEach(function (b) {
      b.setAttribute("aria-pressed", String(high));
      b.setAttribute("aria-label", high ? "Высокая контрастность включена. Выключить" : "Включить высокую контрастность");
    });
  }

  document.querySelectorAll("[data-theme-toggle]").forEach(function (btn) {
    btn.addEventListener("click", function () {
      var next = effectiveTheme() === "dark" ? "light" : "dark";
      root.setAttribute("data-theme", next);
      store.set("goit-theme", next);
      syncButtons();
    });
  });

  document.querySelectorAll("[data-contrast-toggle]").forEach(function (btn) {
    btn.addEventListener("click", function () {
      if (root.getAttribute("data-contrast") === "high") {
        root.removeAttribute("data-contrast");
        store.set("goit-contrast", "normal");
      } else {
        root.setAttribute("data-contrast", "high");
        store.set("goit-contrast", "high");
      }
      syncButtons();
    });
  });
  syncButtons();

  /* Меню-бургер на узких экранах */
  var navToggle = document.querySelector("[data-nav-toggle]");
  var nav = document.getElementById("site-nav");
  if (navToggle && nav) {
    function setNav(open) {
      nav.classList.toggle("is-open", open);
      navToggle.setAttribute("aria-expanded", String(open));
      navToggle.setAttribute("aria-label", open ? "Закрыть меню" : "Открыть меню");
    }
    navToggle.addEventListener("click", function () { setNav(!nav.classList.contains("is-open")); });
    document.addEventListener("keydown", function (e) {
      if (e.key === "Escape" && nav.classList.contains("is-open")) { setNav(false); navToggle.focus(); }
    });
  }

  /* Защита от двойной отправки: форма с data-once блокирует кнопку после первой отправки */
  document.querySelectorAll("form[data-once]").forEach(function (form) {
    form.addEventListener("submit", function () {
      var btn = form.querySelector("[type=submit]");
      if (!btn || btn.disabled) return;
      btn.setAttribute("data-label", btn.textContent);
      btn.textContent = btn.getAttribute("data-busy") || "Отправляем…";
      btn.disabled = true;
      var status = form.querySelector("[data-status]");
      if (status) status.textContent = "Отправляем…";
    });
  });
  window.addEventListener("pageshow", function (e) {
    if (!e.persisted) return;
    document.querySelectorAll("form[data-once] [type=submit]").forEach(function (btn) {
      btn.disabled = false;
      if (btn.getAttribute("data-label")) btn.textContent = btn.getAttribute("data-label");
    });
  });

  /* Оглавление roadmap: якоря генерируем сами, потому что автоматические id для кириллицы нечитаемы */
  var article = document.querySelector("[data-toc]");
  if (article) {
    var h2s = article.querySelectorAll("h2");
    if (h2s.length >= 3) {
      var details = document.createElement("details");
      details.className = "toc";
      details.open = window.matchMedia("(min-width: 861px)").matches;
      var summary = document.createElement("summary");
      summary.textContent = "Содержание";
      var ol = document.createElement("ol");
      h2s.forEach(function (h, i) {
        h.id = "section-" + (i + 1);
        var li = document.createElement("li");
        var a = document.createElement("a");
        a.href = "#" + h.id;
        a.textContent = h.textContent;
        li.appendChild(a);
        ol.appendChild(li);
      });
      details.appendChild(summary);
      details.appendChild(ol);
      var h1 = article.querySelector("h1");
      if (h1) h1.insertAdjacentElement("afterend", details); else article.prepend(details);
    }
  }

  /* Боковая панель темы на схеме roadmap. Без JS узлы ведут по якорям к «Справочнику тем». */
  var rm = document.querySelector(".rm");
  var panel = document.getElementById("rm-panel");
  if (rm && panel && typeof panel.showModal === "function") {
    rm.classList.add("rm-enhanced");
    var pKind = panel.querySelector("[data-panel-kind]");
    var pStage = panel.querySelector("[data-panel-stage]");
    var pTitle = panel.querySelector("#rm-panel-title");
    var pBody = panel.querySelector("[data-panel-body]");

    var openTopic = function (id, updateHash) {
      var src = document.getElementById(id);
      if (!src || !src.classList.contains("rm-topic")) return false;
      pTitle.textContent = src.querySelector("h3").textContent;
      pStage.textContent = src.getAttribute("data-stage");
      pKind.textContent = src.getAttribute("data-kind");
      pBody.textContent = "";
      Array.prototype.forEach.call(src.children, function (child) {
        if (child.tagName !== "H3") pBody.appendChild(child.cloneNode(true));
      });
      panel.scrollTop = 0;
      if (!panel.open) panel.showModal();
      if (updateHash) history.replaceState(null, "", "#" + id);
      return true;
    };

    rm.addEventListener("click", function (e) {
      var node = e.target.closest("[data-topic]");
      if (!node) return;
      e.preventDefault();
      openTopic(node.getAttribute("data-topic"), true);
    });
    panel.addEventListener("click", function (e) {
      /* клик по затемнению (сам <dialog>) или по кнопке закрытия */
      if (e.target === panel || e.target.closest("[data-panel-close]")) panel.close();
    });
    panel.addEventListener("close", function () {
      if (location.hash.indexOf("#t-") === 0) history.replaceState(null, "", location.pathname + location.search);
    });
    window.addEventListener("hashchange", function () {
      if (location.hash.indexOf("#t-") === 0) openTopic(location.hash.slice(1), false);
      else if (panel.open) panel.close();
    });
    if (location.hash.indexOf("#t-") === 0) openTopic(location.hash.slice(1), false);
  }

  /* Чек-лист «Критерии прогресса» в roadmap: отметки хранятся только в браузере */
  var box = document.querySelector("[data-roadmap]");
  if (box) {
    var slug = box.getAttribute("data-roadmap");
    var key = "goit-progress-" + slug;
    var saved = {};
    try { saved = JSON.parse(store.get(key) || "{}") || {}; } catch (e) { saved = {}; }
    var heads = box.querySelectorAll("h2");
    heads.forEach(function (h) {
      if (h.textContent.trim().toLowerCase() !== "критерии прогресса") return;
      var list = h.nextElementSibling;
      if (!list || list.tagName !== "UL") return;
      var items = list.querySelectorAll("li");
      var counter = document.createElement("p");
      counter.className = "hint";
      counter.setAttribute("role", "status");
      function update() {
        var done = list.querySelectorAll("input:checked").length;
        counter.textContent = "Отмечено: " + done + " из " + items.length + ". Отметки хранятся только в этом браузере.";
      }
      list.style.listStyle = "none";
      list.style.paddingLeft = "0";
      items.forEach(function (li, i) {
        var id = "progress-" + i;
        var label = document.createElement("label");
        label.style.cssText = "display:flex;gap:12px;align-items:flex-start;font-weight:400;cursor:pointer";
        var cb = document.createElement("input");
        cb.type = "checkbox";
        cb.id = id;
        cb.style.cssText = "margin-top:6px;width:18px;height:18px;flex:none";
        cb.checked = !!saved[i];
        cb.addEventListener("change", function () {
          saved[i] = cb.checked;
          store.set(key, JSON.stringify(saved));
          update();
        });
        var text = document.createElement("span");
        while (li.firstChild) text.appendChild(li.firstChild);
        label.appendChild(cb);
        label.appendChild(text);
        li.appendChild(label);
      });
      list.insertAdjacentElement("afterend", counter);
      update();
    });
  }

  /* HTMX: шаги кейса меняются без перезагрузки страницы. Без JS форма работает как обычная. */
  document.body.addEventListener("htmx:beforeSwap", function (e) {
    if (e.detail.xhr && e.detail.xhr.status === 422) { e.detail.shouldSwap = true; e.detail.isError = false; }
  });
  document.body.addEventListener("htmx:afterSettle", function () {
    var target = document.querySelector("#main [data-focus]");
    if (target) { target.setAttribute("tabindex", "-1"); target.focus({ preventScroll: true }); }
  });
})();
