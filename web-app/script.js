// ACT — логика сайта записи в секции.
//
// Сайт — точка входа сама по себе: на него ведут прямые ссылки вида
//   index.html?type=swimming&city=vladimir&age=8
// такую ссылку чат-бот формирует по результату теста и присылает пользователю
// (type — рекомендованный вид, см. список в NAMES ниже). Если ссылка открыта
// без параметров, просто показываются все секции без готовой рекомендации —
// отдельного "демо-экрана" в сайте нет.
//
// После отправки формы сайт по-настоящему отправляет письмо с заявкой на почту
// через EmailJS (без бэкенда). Чтобы письма реально приходили, один раз
// Данные секций фронт берёт из mock-api (см. функцию loadData ниже),
// а не из локального массива — единый источник с бэкендом и ботом.
// настройте EmailJS — см. README.md → "Настоящие письма на почту".
 
(function () {
  "use strict";
  var reduceMotion = window.matchMedia && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
 
  // ---------- Настройка EmailJS (заполните после регистрации на emailjs.com) ----------
  var EMAILJS_CONFIG = {
    serviceId: "YOUR_SERVICE_ID",   // EmailJS → Email Services
    templateId: "YOUR_TEMPLATE_ID", // EmailJS → Email Templates
    publicKey: "YOUR_PUBLIC_KEY",   // EmailJS → Account → General → Public Key
    adminEmail: "your@email.com"    // куда должны падать заявки — подставьте в шаблон как {{to_email}}
  };
 
  function isConfigured() {
    return EMAILJS_CONFIG.serviceId.indexOf("YOUR_") !== 0 &&
      EMAILJS_CONFIG.templateId.indexOf("YOUR_") !== 0 &&
      EMAILJS_CONFIG.publicKey.indexOf("YOUR_") !== 0;
  }
  if (typeof window.emailjs !== "undefined" && isConfigured()) {
    try { window.emailjs.init({ publicKey: EMAILJS_CONFIG.publicKey }); } catch (e) { /* библиотека не загрузилась — не критично */ }
  }
 
  // ---------- Иконки (inline SVG, без внешних библиотек) ----------
  var ICONS = {
    football: '<svg viewBox="0 0 24 24" fill="none"><circle cx="12" cy="12" r="9" stroke="currentColor" stroke-width="1.8"/><path d="M12 7l3.5 2.5-1.3 4.1H9.8L8.5 9.5 12 7z" stroke="currentColor" stroke-width="1.6" stroke-linejoin="round"/><path d="M12 3v4M3.5 9l3.7 1.2M20.5 9l-3.7 1.2M6 20l2-5.3M18 20l-2-5.3" stroke="currentColor" stroke-width="1.4" stroke-linecap="round"/></svg>',
    judo: '<svg viewBox="0 0 24 24" fill="none"><circle cx="12" cy="6" r="2.2" stroke="currentColor" stroke-width="1.7"/><path d="M12 8.5v5M8 20l4-6.5L16 20M8.5 12h7" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"/></svg>',
    swimming: '<svg viewBox="0 0 24 24" fill="none"><path d="M3 15c1.5 1.4 3 1.4 4.5 0s3-1.4 4.5 0 3 1.4 4.5 0 3-1.4 4.5 0M3 19c1.5 1.4 3 1.4 4.5 0s3-1.4 4.5 0 3 1.4 4.5 0 3-1.4 4.5 0" stroke="currentColor" stroke-width="1.7" stroke-linecap="round"/><circle cx="16" cy="6" r="1.8" stroke="currentColor" stroke-width="1.6"/><path d="M6 11l5-2 2 3 4-1" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/></svg>',
    art: '<svg viewBox="0 0 24 24" fill="none"><path d="M12 3C6.9 3 3 6.6 3 11c0 3 2 4.5 4 4.5.9 0 1.3-.5 1.3-1.2 0-.6-.4-1-.4-1.8 0-1.8 1.6-3 3.6-3 3.4 0 5.5 2 5.5 4.8 0 3.2-2.2 5.2-6 5.7" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"/><circle cx="8" cy="9.5" r=".9" fill="currentColor"/><circle cx="12.5" cy="7.5" r=".9" fill="currentColor"/><circle cx="16" cy="10" r=".9" fill="currentColor"/></svg>',
    theatre: '<svg viewBox="0 0 24 24" fill="none"><path d="M4 8c0-2.8 3.6-5 8-5s8 2.2 8 5c0 2-1.6 2.6-2.4 4.2-.6 1.2-.2 2.8-1.6 3.6-1.2.7-2.8-.2-4-.2s-2.8.9-4 .2c-1.4-.8-1-2.4-1.6-3.6C5.6 10.6 4 10 4 8z" stroke="currentColor" stroke-width="1.7" stroke-linejoin="round"/><circle cx="9" cy="9" r=".9" fill="currentColor"/><circle cx="15" cy="9" r=".9" fill="currentColor"/></svg>',
    pin: '<svg viewBox="0 0 24 24" fill="none"><path d="M12 21s7-6.3 7-11.5C19 5.4 15.9 3 12 3S5 5.4 5 9.5C5 14.7 12 21 12 21z" stroke="currentColor" stroke-width="1.7" stroke-linejoin="round"/><circle cx="12" cy="9.5" r="2.2" stroke="currentColor" stroke-width="1.6"/></svg>',
    clock: '<svg viewBox="0 0 24 24" fill="none"><circle cx="12" cy="12" r="8.5" stroke="currentColor" stroke-width="1.7"/><path d="M12 7.5V12l3 2" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"/></svg>',
    user: '<svg viewBox="0 0 24 24" fill="none"><circle cx="12" cy="8" r="3.4" stroke="currentColor" stroke-width="1.7"/><path d="M5 20c1.2-3.6 4-5.4 7-5.4s5.8 1.8 7 5.4" stroke="currentColor" stroke-width="1.7" stroke-linecap="round"/></svg>',
    tag: '<svg viewBox="0 0 24 24" fill="none"><path d="M12 3l8 8-9 9-8-8V4h8z" stroke="currentColor" stroke-width="1.7" stroke-linejoin="round"/><circle cx="8.3" cy="7.7" r="1.1" fill="currentColor"/></svg>'
  };
 
  // ---------- Конфигурация API ----------
  // Внутри Docker фронт всегда ходит в относительный /api/ — nginx проксирует на mock-api.
  // При локальной отладке через Live Server (порт 5500/5501) — напрямую на localhost:3001.
  var API_BASE = (function () {
    var p = location.port;
    if (location.protocol === "file:" || p === "5500" || p === "5501") {
      return "http://localhost:3001";
    }
    return "/api";
  })();

  // ---------- Данные (загружаются из mock-api) ----------
  var DATA = { sections: [], districts: [], filterTypes: [], loaded: false };

  function loadData() {
    return Promise.all([
      fetch(API_BASE + "/sections").then(function (r) {
        if (!r.ok) throw new Error("sections: " + r.status);
        return r.json();
      }),
      fetch(API_BASE + "/districts").then(function (r) {
        if (!r.ok) throw new Error("districts: " + r.status);
        return r.json();
      }),
      fetch(API_BASE + "/filterTypes").then(function (r) {
        if (!r.ok) throw new Error("filterTypes: " + r.status);
        return r.json();
      })
    ]).then(function (res) {
      DATA.sections = res[0];
      DATA.districts = res[1];
      DATA.filterTypes = res[2].map(function (x) { return [x.value, x.label]; });
      DATA.loaded = true;
    });
  }
 
  function fmtPrice(p) { return p === 0 ? "Бесплатно" : p.toLocaleString("ru-RU") + " ₽/мес"; }
 
  // ---------- Контекст (из ссылки, которой поделился бот) ----------
  var qs = new URLSearchParams(location.search);
  var ctx = {
    recoType: qs.get("type") || null,
    city: qs.get("city") || "vladimir",
    age: qs.get("age") || null,
    filterTypes: [],
    filterDistricts: [],
    currentSection: null,
    waitlist: false
  };
 
  var stageEl = {};
  document.querySelectorAll(".stage").forEach(function (s) { stageEl[s.dataset.stage] = s; });
  var history_ = [];
  var GROUP_OF_STAGE = { recoList: 1, detail: 2, form: 2, processing: 3, confirm: 3 };
 
  function setRail(stage) {
    var g = GROUP_OF_STAGE[stage] || 1;
    document.querySelectorAll(".rail-step").forEach(function (el) {
      var n = parseInt(el.dataset.group, 10);
      el.classList.remove("done", "current");
      if (n < g) el.classList.add("done");
      else if (n === g) el.classList.add("current");
    });
  }
 
  function goTo(stage, opts) {
    opts = opts || {};
    if (!opts.silent) {
      var cur = document.querySelector(".stage.active");
      if (cur) history_.push(cur.dataset.stage);
    }
    Object.keys(stageEl).forEach(function (k) { stageEl[k].classList.toggle("active", k === stage); });
    setRail(stage);
    window.scrollTo(0, 0);
  }
  function goBack() {
    var prev = history_.pop();
    if (prev) goTo(prev, { silent: true });
  }
  document.querySelectorAll("[data-back]").forEach(function (b) { b.addEventListener("click", goBack); });
 
  // ---------- RECO + LIST (точка входа) ----------
  function renderRecoHeader() {
    var t = ctx.recoType;
    if (t) {
      document.getElementById("recoEyebrow").textContent = "Рекомендация из чат-бота MAX";
      document.getElementById("recoTitle").textContent = NAMES[t] || "Секция";
      var ageTxt = ctx.age ? ", возраст " + ctx.age + " лет" : "";
      document.getElementById("recoWhy").innerHTML =
        "<b>По итогам теста в чат-боте</b> ребёнку подходит: «" + (NAMES[t] || "Секция") + "»" + ageTxt +
        ". Ниже — секции этого профиля рядом с вами, отсортированные по наличию мест. Можно посмотреть и другие варианты через фильтр.";
    } else {
      document.getElementById("recoEyebrow").textContent = "Секции рядом";
      document.getElementById("recoTitle").textContent = "Все секции";
      document.getElementById("recoWhy").innerHTML = "Выберите вид занятий и район — ниже появятся подходящие секции.";
    }
  }
 
  // Один фильтр вместо двух рядов чипов: кнопка "Фильтр" открывает панель с чекбоксами.
  // Пока ничего не отмечено в группе — показаны все варианты по ней; отметили одно
  // или несколько значений — список сужается только до них. Есть кнопка сброса.

  function toggleInArray(arr, value, checked) {
    var idx = arr.indexOf(value);
    if (checked && idx === -1) arr.push(value);
    if (!checked && idx !== -1) arr.splice(idx, 1);
  }
 
  function updateFilterBadge() {
    var active = ctx.filterTypes.length + ctx.filterDistricts.length;
    document.getElementById("filterBadge").hidden = active === 0;
  }
 
  function renderCheckGroup(hostId, options, selectedArr) {
    var host = document.getElementById(hostId);
    host.innerHTML = options.map(function (opt) {
      var value = Array.isArray(opt) ? opt[0] : opt;
      var label = Array.isArray(opt) ? opt[1] : opt;
      var checked = selectedArr.indexOf(value) !== -1;
      return '<label class="check-item"><input type="checkbox" value="' + value + '"' + (checked ? " checked" : "") + ">" + label + "</label>";
    }).join("");
    host.querySelectorAll("input").forEach(function (cb) {
      cb.addEventListener("change", function () {
        toggleInArray(selectedArr, cb.value, cb.checked);
        updateFilterBadge();
        renderList();
      });
    });
  }
 
  function renderFilters() {
    renderCheckGroup("checksType", DATA.filterTypes, ctx.filterTypes);
    renderCheckGroup("checksDistrict", DATA.districts, ctx.filterDistricts);
    updateFilterBadge();

    document.getElementById("btnResetFilter").addEventListener("click", function () {
      ctx.filterTypes = [];
      ctx.filterDistricts = [];
      renderCheckGroup("checksType", DATA.filterTypes, ctx.filterTypes);
      renderCheckGroup("checksDistrict", DATA.districts, ctx.filterDistricts);
      updateFilterBadge();
      renderList();
    });
 
    var toggle = document.getElementById("filterToggle");
    var panel = document.getElementById("filterPanel");
    toggle.addEventListener("click", function (e) {
      e.stopPropagation();
      var willOpen = panel.hidden;
      panel.hidden = !willOpen;
      toggle.setAttribute("aria-expanded", String(willOpen));
    });
    document.addEventListener("click", function (e) {
      if (!panel.hidden && !panel.contains(e.target) && !toggle.contains(e.target)) {
        panel.hidden = true;
        toggle.setAttribute("aria-expanded", "false");
      }
    });
  }
 
  function renderList() {
    renderRecoHeader();
      var list = DATA.sections.filter(function (s) { 
      if (ctx.filterTypes.length && ctx.filterTypes.indexOf(s.type) === -1) return false;
      if (ctx.filterDistricts.length && ctx.filterDistricts.indexOf(s.district) === -1) return false;
      return true;
    });
    list.sort(function (a, b) {
      var ar = ctx.recoType && a.sport === ctx.recoType ? 0 : 1;
      var br = ctx.recoType && b.sport === ctx.recoType ? 0 : 1;
      if (ar !== br) return ar - br;
      return b.spots - a.spots;
    });
    var host = document.getElementById("sectionList");
    if (list.length === 0) {
      host.innerHTML = '<div class="empty">В этом районе пока нет подходящих секций. Попробуйте другой фильтр.</div>';
      return;
    }
    host.innerHTML = list.map(function (s) {
      var spotsBadge = s.spots === 0
        ? '<span class="badge full">Мест нет</span>'
        : '<span class="badge spots">' + s.spots + " " + (s.spots === 1 ? "место" : "места") + "</span>";
      var recoBadge = ctx.recoType && s.sport === ctx.recoType ? '<span class="badge reco">Рекомендовано</span>' : "";
      return '<button class="scard" data-id="' + s.id + '">' +
        '<span class="scard-icon ' + s.type + '">' + ICONS[s.sport] + "</span>" +
        '<span class="scard-body">' +
        '<span class="scard-name">' + s.name + "</span>" +
        '<div class="scard-meta">' + recoBadge + spotsBadge + "</div>" +
        '<div class="scard-meta">' + s.district + " район · от " + s.minAge + " лет</div>" +
        '<div class="scard-price">' + fmtPrice(s.price) + "</div>" +
        "</span>" +
        "</button>";
    }).join("");
    host.querySelectorAll(".scard").forEach(function (c) {
      c.addEventListener("click", function () { openDetail(parseInt(c.dataset.id, 10)); });
    });
  }
 
  // ---------- DETAIL ----------
  function openDetail(id) {
    var s = DATA.sections.filter(function (x) { return x.id === id; })[0];
    ctx.currentSection = s;
    var full = s.spots === 0;
    document.getElementById("detailHost").innerHTML =
      '<div class="detail-media ' + s.type + '">' + ICONS[s.sport] + "</div>" +
      "<h2>" + s.name + "</h2>" +
      '<div class="detail-rows">' +
      '<div class="drow">' + ICONS.pin + '<div><div class="drow-label">Адрес</div><div class="drow-val">' + s.address + ", " + s.district + " район, Владимир</div></div></div>" +
      '<div class="drow">' + ICONS.clock + '<div><div class="drow-label">Расписание</div><div class="drow-val">' + s.schedule + "</div></div></div>" +
      '<div class="drow">' + ICONS.user + '<div><div class="drow-label">Педагог / тренер</div><div class="drow-val">' + s.coach + "</div></div></div>" +
      '<div class="drow">' + ICONS.tag + '<div><div class="drow-label">Возраст</div><div class="drow-val">от ' + s.minAge + " лет</div></div></div>" +
      "</div>" +
      '<div class="price-row"><span class="amt">' + fmtPrice(s.price) + "</span>" +
      (full ? '<span class="badge full">Мест нет</span>' : '<span class="badge spots">' + s.spots + " своб. мест</span>") +
      "</div>" +
      '<div class="sticky-cta"><button class="btn btn-primary btn-block" id="btnGoForm">' +
      (full ? "Записаться в лист ожидания" : "Записать ребёнка") + "</button></div>";
    document.getElementById("btnGoForm").addEventListener("click", function () {
      ctx.waitlist = full;
      openForm();
    });
    goTo("detail");
  }
 
  // ---------- FORM ----------
  function openForm() {
    var s = ctx.currentSection;
    document.getElementById("formSub").textContent = (ctx.waitlist ? "Лист ожидания · " : "") + s.name;
    if (ctx.age) document.getElementById("fAge").value = String(ctx.age);
    goTo("form");
  }

  function validateForm() {
    var ok = document.getElementById("fParent").value.trim().length > 1 &&
      document.getElementById("fPhone").value.trim().length > 5 &&
      document.getElementById("fChild").value.trim().length > 1 &&
      document.getElementById("fConsent").checked;
    document.getElementById("btnSubmitForm").disabled = !ok;
  }
  ["fParent", "fPhone", "fChild", "fConsent"].forEach(function (id) {
    document.getElementById(id).addEventListener("input", validateForm);
    document.getElementById(id).addEventListener("change", validateForm);
  });
  document.getElementById("btnSubmitForm").addEventListener("click", startProcessing);
 
  // ---------- PROCESSING ----------
  // Шаг с пометкой integration — место будущего вызова API Госуслуг (пока имитация).
  // Последний реальный шаг — sendBookingEmail() — по-настоящему отправляет письмо через EmailJS.
  var PROC_STEPS = [
    { t: "Проверяем свободные места", note: null },
    { t: "Резервируем место за ребёнком", note: null },
    { t: "Оформляем запись через Госуслуги", note: "integration" }
  ];

  function startProcessing() {
    document.getElementById("procTitle").textContent = ctx.waitlist ? "Добавляем в лист ожидания…" : "Отправляем заявку…";
    var host = document.getElementById("procList");
    host.innerHTML = PROC_STEPS.map(function (p) {
      var noteHtml = p.note === "integration"
        ? '<div class="integration-tag"><svg viewBox="0 0 24 24" fill="none"><path d="M12 3v3M12 18v3M4.2 4.2l2.1 2.1M17.7 17.7l2.1 2.1M3 12h3M18 12h3M4.2 19.8l2.1-2.1M17.7 6.3l2.1-2.1" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"/></svg>Демо: точка интеграции с API Госуслуг</div>'
        : "";
      return '<div class="proc-item"><span class="proc-mark"><span class="spinner"></span><svg class="check" viewBox="0 0 24 24" fill="none"><path d="M5 13l4 4L19 7" stroke="#fff" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"/></svg></span><span><div class="proc-title">' + p.t + "</div>" + noteHtml + "</span></div>";
    }).join("");
    goTo("processing");
    runSteps(0);
  }

  function runSteps(i) {
    var items = document.querySelectorAll("#procList .proc-item");
    if (i >= items.length) {
      sendBookingEmail().then(function (result) {
        setTimeout(function () { showConfirm(result); }, reduceMotion ? 50 : 300);
      });
      return;
    }
    items.forEach(function (el, idx) { el.classList.toggle("active", idx === i); });
    var dur = reduceMotion ? 60 : 700 + i * 150;
    setTimeout(function () {
      items[i].classList.remove("active");
      items[i].classList.add("done");
      // TODO(реальная интеграция): когда появится доступ к API Госуслуг, настоящий
      // вызов записи должен произойти здесь (шаг i === 2), перед переходом дальше.
      runSteps(i + 1);
    }, dur);
  }
 
  // ---------- Настоящее письмо через EmailJS ----------
  function sendBookingEmail() {
    var s = ctx.currentSection;
    var params = {
      to_email: EMAILJS_CONFIG.adminEmail,
      section_name: s.name,
      section_address: s.address + ", " + s.district + " район, Владимир",
      section_schedule: s.schedule,
      section_price: fmtPrice(s.price),
      status: ctx.waitlist ? "Лист ожидания" : "Подтверждено",
      parent_name: document.getElementById("fParent").value.trim(),
      parent_phone: document.getElementById("fPhone").value.trim(),
      parent_email: document.getElementById("fEmail").value.trim() || "не указан",
      child_name: document.getElementById("fChild").value.trim(),
      child_age: document.getElementById("fAge").value,
      submitted_at: new Date().toLocaleString("ru-RU")
    };
 
    if (typeof window.emailjs === "undefined" || !isConfigured()) {
      return Promise.resolve({ ok: false, reason: "not_configured" });
    }
    return window.emailjs.send(EMAILJS_CONFIG.serviceId, EMAILJS_CONFIG.templateId, params)
      .then(function () { return { ok: true }; })
      .catch(function (err) {
        console.error("EmailJS: не удалось отправить письмо", err);
        return { ok: false, reason: "send_failed", error: err };
      });
  }
 
  // ---------- CONFIRM ----------
  function showConfirm(emailResult) {
    var s = ctx.currentSection;
    var wl = ctx.waitlist;
    document.getElementById("confirmTitle").textContent = wl ? "Вы в листе ожидания" : "Заявка принята!";
    document.getElementById("confirmLede").textContent = wl
      ? "Как только освободится место, вам придёт уведомление."
      : "Место зарезервировано за вашим ребёнком.";
    var child = document.getElementById("fChild").value.trim() || "Ребёнок";
    document.getElementById("confirmSummary").innerHTML =
      '<div class="csn">' + s.name + "</div>" +
      '<div class="crow"><span>Ребёнок</span><span>' + child + ", " + document.getElementById("fAge").value + " лет</span></div>" +
      '<div class="crow"><span>Расписание</span><span>' + s.schedule + "</span></div>" +
      '<div class="crow"><span>Адрес</span><span>' + s.address + "</span></div>" +
      '<div class="crow"><span>Стоимость</span><span>' + fmtPrice(s.price) + "</span></div>" +
      '<div class="crow"><span>Статус</span><span>' + (wl ? "Лист ожидания" : "Подтверждено") + "</span></div>";
 
    var noteEl = document.getElementById("confirmNote");
    var noteText = document.getElementById("confirmNoteText");
    var ok = emailResult && emailResult.ok;
    noteEl.classList.toggle("note-warn", !ok);
    if (ok) {
      noteText.textContent = "Письмо с заявкой отправлено на почту организатора.";
    } else if (emailResult && emailResult.reason === "not_configured") {
      noteText.textContent = "Письмо не отправлено: подключите EmailJS в script.js (см. README.md), чтобы заявки реально приходили на почту.";
    } else {
      noteText.textContent = "Не удалось отправить письмо — проверьте настройки EmailJS и подключение к интернету.";
    }
    goTo("confirm");
  }
 
  document.getElementById("btnRestart").addEventListener("click", function () {
    ctx.waitlist = false;
    ["fParent", "fPhone", "fEmail", "fChild"].forEach(function (id) { document.getElementById(id).value = ""; });
    document.getElementById("fConsent").checked = false;
    document.getElementById("fAge").selectedIndex = 0;
    document.getElementById("btnSubmitForm").disabled = true;
    history_ = [];
    goTo("recoList", { silent: true });
  });
 
  // ---------- BOOT ----------
  var bootHost = document.getElementById("sectionList");
  bootHost.innerHTML = '<div class="empty">Загружаем секции…</div>';

  loadData()
    .then(function () {
      renderFilters();
      renderList();
      goTo("recoList", { silent: true });
    })
    .catch(function (err) {
      console.error("Не удалось загрузить данные из mock-api:", err);
      bootHost.innerHTML =
        '<div class="empty">Не удалось загрузить секции. ' +
        'Проверьте, что mock-api запущен. Адрес: ' + API_BASE + '</div>';
    });
})();
 