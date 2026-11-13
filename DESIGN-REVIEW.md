# GO-IT: Финальный дизайн-ревью и рекомендации

**Дата:** 10 ноября 2026
**Статус:** Представлено на реализацию
**Целевая аудитория:** Подростки 14–17 лет
**Технологии:** SSR Go + `html/template`, HTMX, CSS Custom Properties (3 темы)

---

## Содержание

1. [Резюме и общие оценки](#1-резюме-и-общие-оценки)
2. [Consistency Audit](#2-consistency-audit)
3. [Brand Visual Alignment](#3-brand-visual-alignment)
4. [Flow Analysis](#4-flow-analysis)
5. [Information Hierarchy](#5-information-hierarchy)
6. [Оценка каждого экрана (1–5)](#6-оценка-каждого-экрана-15)
7. [Рекомендации по приоритетам](#7-рекомендации-по-приоритетам)
8. [Конкретные примеры улучшений](#8-конкретные-примеры-улучшений)
9. [Accessibility Audit](#9-accessibility-audit)
10. [Дорожная карта улучшений](#10-дорожная-карта-улучшений)

---

## 1. Резюме и общие оценки

### Общее впечатление

GO-IT — **крепкий MVP** с продуманной базой: системные токены, три темы, семантическая HTML-разметка, доступная навигация. Однако платформа ощущается «студенческой», а не «продуктовой». Разрозненные отступы, эмодзи в навигации, отсутствие единой системы间距 и слабая визуальная иерархия снижают технологичный образ.

| Критерий | Оценка | Комментарий |
|----------|--------|-------------|
| Типографика | ⭐⭐⭐⭐☆ | Системные тоны `.h1`–`.h3` определены, но применяются непоследовательно |
| Цвета | ⭐⭐⭐⭐⭐ | Палитра `hero-bg` → `cyan` → `cta-bg` работает; 3 темы корректны |
| Компоненты | ⭐⭐⭐☆☆ | Карточки, кнопки, формы есть, но используются фрагментарно |
| Отступы/Сетка | ⭐⭐☆☆☆ | Разброс 8px–80px без системы; inline-стили на каждой второй странице |
| Мобильная адаптация | ⭐⭐⭐⭐☆ | Breakpoints 600/860/980px покрывают базовые сценарии |
| Доступность | ⭐⭐⭐⭐☆ | Skip-link, ARIA, focus-visible — хорошо. Эмодзи в навигации — минус |
| Визуальная «ценность» | ⭐⭐⭐☆☆ | Концепт сильный, но исполнение выглядит «первым маком» |

---

## 2. Consistency Audit

### 2.1 Типографика

**Определено в CSS:**
```css
.h1 { font-size: clamp(2rem, 5.2vw, 3.4rem); }
.h2 { font-size: clamp(1.5rem, 3.4vw, 2.2rem); }
.h3 { font-size: 1.2rem; }
```

**Проблемы:**

| Шаблон | Проблема | Текущее значение | Должно быть |
|--------|----------|------------------|-------------|
| `home.html` | `.hero h1` — свой `clamp(2rem, 5.2vw, 3.4rem)` — совпадает, но без класса `.h1` | `h1` (raw) | `class="h1"` |
| `page-head` | `.page-head .h1` переопределяет на `clamp(1.9rem, 4.4vw, 2.8rem)` — конфликтует с глобальной `.h1` | 2 значения h1 | Единая шкала |
| `case_question.html` | `h1` кейса — свой `clamp(1.6rem, 4vw, 2.3rem)` — третий размер | 3 значения h1 | `class="h2"` |
| `case_result.html` | `h1` скрыт `.visually-hidden`, но `p.score strong` — 4rem — нет токена | 4rem inline | Токен `.score-num` |
| `lk_student.html` | `.lk-heading` — 1.5rem, но `.lk-section h3` — 1.1rem — иерархия сломана | 2 значения внутри LK | `.lk-heading` → 1.8rem, `h3` → 1.2rem |
| `mentor.html` | `h1` — `font-size: 1.8rem` inline — 6-й размер | 1.8rem inline | `class="h2"` |
| `roadmap.html` | `h1` через `.prose` — `clamp(1.8rem, 4.4vw, 2.6rem)` — 4-й размер | `clamp(1.8rem, 4.4vw, 2.6rem)` | `class="h2"` |

**Рекомендация:** Единая типографская шкала с токенами.

### 2.2 Система spacing

**Текущие значения (без системы):**

```
8px, 10px, 12px, 14px, 16px, 18px, 20px, 22px, 24px, 26px,
28px, 32px, 36px, 38px, 40px, 44px, 46px, 48px, 56px, 60px,
72px, 80px, 88px, 96px
```

40+ уникальных значений padding/margin. Это не система — это хаос.

**Примеры конфликтов:**

| Место | Текущий padding | Проблема |
|-------|-----------------|----------|
| `.section` | `80px 0` | Не совпадает с `.hero` (80px 0 88px) |
| `.page-head` | `56px 0 8px` | Начинается раньше, чем `.section` — нет визуального зазора |
| `.lk-main` | `32px 40px` | Боковые отступы (40px) ≠ обёртка (32px) |
| `home.html` секция "help" | inline `padding-bottom: 80px` на карточках | Inline-стиль вместо класса |
| `direction.html` | `margin-top: 20px` на h1 inline | Inline |
| `roadmap.html` | inline `padding: 24px 0 80px` | Inline |

**Рекомендация:** Ввести 5 уровней вертикального spacing через CSS Custom Properties.

### 2.3 Система карточек

**Определено:** `.card` — 28px padding, 20px border-radius, 1px border, box-shadow.

**Несоответствия:**
- `.lk-section` — 28px padding, `var(--radius)` — ОК
- `.lk-item` — 16px padding — отличается от `.card`
- `application_form.html` → `.card` внутри `.split` — 28px padding, но без внутренних отступов для заголовков
- `.auth-form-card` — 28px padding — совпадает
- `preview` (на лендинге) — 26px padding — не совпадает с 28px

### 2.4 Кнопки

**Определено:** `.btn` (48px min-height), `.btn--cta`, `.btn--primary`, `.btn--outline`, `.btn--ghost`, `.btn--sm` (38px), `.btn--vk`, `.btn--ya`.

**Проблемы:**
- В LK используются `.btn--primary btn--sm` — ОК
- `.btn-row` — 14px gap — корректно
- В `cases.html` карточках — `btn--cta` без `.btn-row` обёртки — OK но неконсистентно
- **Нет** `.btn--danger` — кнопки «Отклонить» используют `btn--outline` вместо визуального различия
- В `application_form.html` — кнопка submit — `btn--primary` — OK
- `btn[disabled]` и `aria-disabled` — стилизованы, но `aria-disabled` не работает с HTMX (нужен `disabled`)

### 2.5 Grid system

**Определено:**
- `.grid-2` → `repeat(2, minmax(0, 1fr))`
- `.grid-3` → `repeat(3, minmax(0, 1fr))`
- `.steps-row` → `repeat(4, minmax(0, 1fr))`
- `.num-grid` → `repeat(3, minmax(0, 1fr))`

**Несоответствия:**
- `lk_student.html`, `lk_mentor.html`, `lk_partner.html` — нет `wrap`/`lk-main` обёртки для контента. Статистика `.lk-stats` использует inline grid без break-points
- `case_result.html` — `.result-list` — custom grid, нет класса
- `split` layout (mentors, volunteering) — `1.2fr` / `1fr` — нестандартное соотношение

---

## 3. Brand Visual Alignment

### 3.1 Что работает

| Элемент | Оценка | Комментарий |
|---------|--------|-------------|
| Шрифты | ⭐⭐⭐⭐⭐ | Unbounded (заголовки) + Onest (текст) — отличный выбор для IT-платформы |
| Цветовая палитра | ⭐⭐⭐⭐⭐ | Navy + Cyan + Blue + Orange CTA — «технологичный» контраст |
| Три темы | ⭐⭐⭐⭐⭐ | Светлая, тёмная, high-contrast — FOUC-protected |
| Превью кейса | ⭐⭐⭐⭐☆ | На главном экране — сильный визуальный элемент |
| Прогресс-бары | ⭐⭐⭐⭐☆ | Работают во всех контекстах |
| Логотип | ⭐⭐⭐⭐⭐ | SVG, чёткий, технологичный |

### 3.2 Что снижает «технологичность»

| Элемент | Проблема | Влияние |
|---------|----------|---------|
| **Эмодзи в LK навигации** | `🏠`, `📋`, `🗺️`, `⭐`, `💚`, `⚙️`, `🚪`, `📅`, `👥`, `✏️`, `📩`, `📊`, `📌` | Размывает профессиональный вид, выглядит «детским» |
| **Страница логина** | Три блока кнопок (соцсети + быстрый вход) — перегруз | 7 кнопок на одной форме — когнитивная перегрузка |
| **Карточки без изображений** | Все карточки — только иконка-буква | Нет визуальной дифференциации направлений |
| **Page-head на list-страницах** | Простой h1 + lead — нет визуального якоря | Каждая list-страница выглядит одинаково |
| **Footer** | «Студенты группы ИПС611-3» | Позиционирует как студенческий проект, а не продукт |
| **Простые карточки направлений** | `ic` → буква → текст — мало информации | Нет мини-превью профессии |

### 3.3 Рекомендации по технологичному виду

1. **Заменить эмодзи на SVG-иконки** в стиле логотипа (stroke-based, 20×20, `var(--cyan)` акцент)
2. **Убрать блок «быстрый вход»** со страницы логина — оставить соцсети + email/пароль
3. **Добавить cover-график** для карточек направлений (абстрактные паттерны, не фото)
4. **Стандартизировать page-head** с eyebrow → h1 → lead → визуальный якорь (линия, декор, или иконка)
5. **Footer** — заменить на стандартный: ссылки на направления, о проекте, контакты, копирайт

---

## 4. Flow Analysis

### 4.1 Student Flow: Лендинг → Результат кейса

```
/ (лендинг)
  → Клик «Пройти первый кейс» или «Выбрать направление»
  → /cases (каталог кейсов)
    → Клик «Начать» на карточке
    → /cases/{slug} (вопрос 1)
      → Выбор варианта → POST
      → /cases/{slug} (вопрос 2)
        → ... (n вопросов)
      → /cases/{slug}/result (результат)
        → Клик «Перейти к roadmap»
        → /roadmaps/{slug} (roadmap)
          → Клик «Найти наставника»
          → /mentors (каталог менторов)
            → Клик на ментора
            → /mentors/{slug} (профиль)
              → Клик «Отправить заявку»
              → Форма → POST → /mentors/{slug}?sent=1
                → → → Конец потока
```

**Потери на каждом шаге:**

| Шаг | Потенциальная потеря | Причина |
|-----|---------------------|---------|
| Лендинг → Каталог кейсов | ~30% | Две CTA-кнопки делят внимание; нет персонализации |
| Каталог → Кейс | ~20% | Карточки однотипны; нет объяснения «почему это про вас» |
| Вопрос → Вопрос | ~15% | Нет сохранения прогресса при закрытии вкладки |
| Результат → Roadmap | ~25% | CTA «Перейти к roadmap» — один из трёх; нет подсвеченного выбора |
| Roadmap → Ментор | ~40% | CTA «Найти наставника» — среди других; нет персонализации |
| Каталог менторов → Профиль | ~35% | Фильтры работают, но карточки менторов одинаковые |
| Профиль → Заявка → Конец | ~50% | После заявки — нет объяснения «что дальше», тупик |

**Dead-ends:**
1. После заявки на ментора — возврат на профиль ментора с текстом «Заявка сохранена» — пользователь не знает, где увидеть статус
2. После результата кейса — нет кнопки «Пройти ещё один кейс»
3. На странице ментора нет кнопки «Назад к списку»

### 4.2 Mentor Flow: Вход → Обработка заявки

```
/auth (логин)
  → /lk/mentor (дашборд)
    → Клик «Заявки»
    → /lk/mentor/applications
      → Клик «Принять»/«Отклонить»
        → 302 на /lk/mentor (с флеш-сообщением)
          → → → Конец
```

**Потери:**
- Нет разделения заявок на «Новые», «В обработке», «Завершённые» — одна длинная страница
- Нет контекста заявки: студент не видит историю прохождения кейсов
- Принятие/отклонение — сразу редирект на дашборд — нет подтверждения

### 4.3 Volunteer Flow

```
/volunteering (каталог)
  → /volunteering/{slug} (задача)
    → Форма отклика → POST
    → → Конец (redirect на страницу задачи)
```

**Потери:**
- После отклика — нет объяснения «что будет дальше»
- Нет страницы «Мои отклики» для студента
- Нет статуса заявки для волонтёра

### 4.4 Recommendations for flow improvements

1. **Персонализированный лендинг:** после просмотра направления — показывать релевантный кейс
2. **Сохранение прогресса:** при закрытии вкладки — сохранять текущий вопрос в localStorage
3. **Пост-кейс:** персонализированная рекомендация roadmap + конкретный ментор по теме
4. **Пост-заявка:** страница «Статус заявки» с таймлайном
5. **Навигация назад:** на каждой detail-странице — кнопка «← Назад к списку»

---

## 5. Information Hierarchy

### 5.1 Визуальные акценты

| Элемент | Текущий акцент | Оценка |
|---------|----------------|--------|
| H1 | `.h1` — 2–3.4rem, Unbounded 700 | ✅ Хорошо |
| Eyebrow | `.eyebrow` — 0.82rem, uppercase, --primary | ✅ Хорошо |
| Lead | `.lead` — 1.05rem, --muted | ✅ Хорошо |
| CTA | `.btn--cta` — orange, 48px | ⚠️ Иногда теряется среди других кнопок |
| Score result | `4rem` — большой, но без цвета | ⚠️ Только цвет из --primary |
| Badge | `--ok`, `--error`, `--primary` | ✅ Хорошо |

### 5.2 CTA видимость

**Сильные места:**
- Главная CTA «Пройти первый кейс» — orange, крупная, первая в `.btn-row`
- Hero-секция — CTA на navy фоне — высокий контраст

**Слабые места:**
- В `case_result.html` — 3 кнопки на одном уровне — `btn--cta` выделена, но `btn--outline` кнопки визуально спорят
- В LK-дашбордах — кнопки в `.lk-item` маленькие (`btn--sm`) — OK, но нет визуального приоритета между «Продолжить» и «Записаться»
- В `roadmap.html` — CTA «Проверить себя на кейсе» — `btn--cta` среди `btn--outline` — OK, но кнопка не на уровне внимания после длинного roadmap

### 5.3 Потерянная важная информация

| Информация | Где теряется | Рекомендация |
|------------|-------------|-------------|
| Статус заявки после отправки | Только alert в форме — легко пропустить | Выделить alert в отдельный блок с иконкой |
| Дисклеймер в результате кейса | `.hint` — приглушённый шрифт | Отдельный блок с `--ok-bg` |
| Новых заявок у ментора | Только badge в навигации — маленький | Добавить prominent alert на дашборде |
| Рекомендации после кейса | Среди 3 кнопок — CTA не выделяется | Выделить roadmap-кнопку как primary |

---

## 6. Оценка каждого экрана (1–5)

### Публичные страницы

| # | Экран | Маршрут | Оценка | Ключевые проблемы |
|---|-------|---------|--------|-------------------|
| 1 | Главная / Лендинг | `/` | ⭐⭐⭐⭐☆ | Две CTA делят внимание; карточки направлений без cover-изображений |
| 2 | О проекте | `/about` | ⭐⭐⭐⭐☆ | Хороший контент; нет визуального акцента на CTA в конце |
| 3 | Каталог направлений | `/directions` | ⭐⭐⭐☆☆ | Карточки одинаковые; нет визуального якоря; eyebrow → h1 → lead — OK но平淡 |
| 4 | Страница направления | `/directions/{slug}` | ⭐⭐⭐☆☆ | `page-narrow` layout — OK; нет превью кейса; inline margin-top:20px |
| 5 | Каталог кейсов | `/cases` | ⭐⭐⭐☆☆ | Карточки кейсов — нет визуальной разницы между направлениями; placeholder при пустом списке |
| 6 | Вопрос кейса | `/cases/{slug}` | ⭐⭐⭐⭐☆ | Прогресс-бар, radio-карточки — хорошо. Нет подсказки «можно пропустить» |
| 7 | Результат кейса | `/cases/{slug}/result` | ⭐⭐⭐⭐☆ | Score — крупный. 3 CTA — один теряется. Дисклеймер — в hint |
| 8 | Каталог менторов | `/mentors` | ⭐⭐⭐☆☆ | Карточки одинаковые; нет аватаров/фото; фильтры работают но flat |
| 9 | Профиль ментора | `/mentors/{slug}` | ⭐⭐⭐☆☆ | Split layout — OK; форма в карточке — OK; нет кнопки «Назад» |
| 10 | Каталог волонтёрства | `/volunteering` | ⭐⭐⭐☆☆ | Аналогично менторам —平淡 карточки |
| 11 | Задача волонтёрства | `/volunteering/{slug}` | ⭐⭐⭐☆☆ | Аналогично ментору — нет визуальной дифференциации |
| 12 | Каталог roadmaps | `/roadmaps` | ⭐⭐⭐⭐☆ | Хорошо структурирован; `grid-2` — OK; можно добавить cover-изображения |
| 13 | Roadmap детальная | `/roadmaps/{slug}` | ⭐⭐⭐⭐☆ | Диаграмма — сильная сторона. CTA в конце — OK. Long topic list — утомляет |
| 14 | Блог (список) | `/blog` | ⭐⭐⭐☆☆ | Карточки без изображений; дата — tag вверху — OK но平淡 |
| 15 | Статья | `/blog/{slug}` | ⭐⭐⭐⭐☆ | Проза-layout — хороший. CTA внизу — OK. Cover-изображение отсутвует |
| 16 | Страница ошибки | `/error/{code}` | ⭐⭐⭐☆☆ | `page-narrow` — OK; `h1` + lead — OK; одна CTA — минимальная |
| 17 | 404 Not Found | `/` (fallback) | ⭐⭐⭐⭐☆ | Две CTA — OK; `page-narrow` — OK; дружелюбный текст |

### Авторизация

| # | Экран | Маршрут | Оценка | Ключевые проблемы |
|---|-------|---------|--------|-------------------|
| 18 | Логин / Auth | `/auth` | ⭐⭐☆☆☆ | **7 кнопок** на одной форме; 3 роли; соцсети + быстрый вход — перегруз; «Узнайте о проекте» — слабая ссылка |
| 19 | ЛК Ученика | `/lk/student` | ⭐⭐⭐☆☆ | Эмодзи навигация; placeholder-данные; нет пустых состояний |
| 20 | ЛК Ментора | `/lk/mentor` | ⭐⭐⭐☆☆ | Эмодзи навигация; вкладки без JS не работают; badges — OK |
| 21 | ЛК Партнёра | `/lk/partner` | ⭐⭐⭐☆☆ | Эмодзи навигация; inline table в HTML — плохая практика; вкладки без JS |

---

## 7. Рекомендации по приоритетам

### 🔴 Критично (исправить сейчас)

| # | Задача | Страницы | Влияние |
|---|--------|----------|---------|
| 1 | **Заменить эмодзи на SVG-иконки** в LK-навигации | `lk_student`, `lk_mentor`, `lk_partner` | Профессиональный вид, доступность |
| 2 | **Ввести систему spacing tokens** | Все страницы | Консистентность, поддержка |
| 3 | **Переработать страницу логина** — убрать «быстрый вход» | `login.html` | Когнитивная разгрузка, фокус на одной action |
| 4 | **Добавить .btn--danger** для кнопок «Отклонить» | `lk_mentor`, `lk_partner` | Визуальное различие действий |
| 5 | **Стандартизировать заголовки** — убрать inline font-size | `direction`, `mentor`, `roadmap`, `case_result` | Типографская консистентность |

### 🟡 Важно (следующая итерация)

| # | Задача | Страницы | Влияние |
|---|--------|----------|---------|
| 6 | Добавить cover-изображения для карточек направлений | `home`, `directions` | Визуальная дифференциация |
| 7 | Добавить кнопку «Назад к списку» на detail-страницах | `direction`, `mentor`, `volunteer` | Навигационная ясность |
| 8 | Персонализировать результат кейса — показать roadmap как primary CTA | `case_result` | Удержание пользователя |
| 9 | Добавить пустые состояния (empty state) для LK-разделов | `lk_student`, `lk_mentor`, `lk_partner` | UX при отсутствии данных |
| 10 | Выделить дисклеймер результата кейса в отдельный блок | `case_result` | Информационная ясность |

### 🟢 Желательно (будущее)

| # | Задача | Страницы | Влияние |
|---|--------|----------|---------|
| 11 | Реализовать систему `.grid-4` и `.gap-*` утилит | Все страницы | Гибкость сетки |
| 12 | Добавить cover-image для статей блога | `blog`, `article` | Визуальная привлекательность |
| 13 | Footer — убрать «студенты группы ИПС611-3» | `footer.html` | Продуктовый вид |
| 14 | Добавить page-header декор на list-страницы | `directions`, `mentors`, `volunteering` | Визуальная иерархия |
| 15 | Страница «Мои заявки» для студента | Новый маршрут | Полный цикл обратной связи |

---

## 8. Конкретные примеры улучшений

### 8.1 Система spacing tokens

**CSS:**
```css
:root {
  /* Добавить в существующие токены */
  --space-1: 4px;
  --space-2: 8px;
  --space-3: 12px;
  --space-4: 16px;
  --space-5: 24px;
  --space-6: 32px;
  --space-7: 48px;
  --space-8: 64px;
  --space-9: 80px;
}

/* Утилиты */
.gap-2 { gap: var(--space-2); }
.gap-3 { gap: var(--space-3); }
.gap-4 { gap: var(--space-4); }
.gap-5 { gap: var(--space-5); }
.gap-6 { gap: var(--space-6); }
.gap-7 { gap: var(--space-7); }
.gap-8 { gap: var(--space-8); }
.gap-9 { gap: var(--space-9); }

.py-6 { padding-top: var(--space-6); padding-bottom: var(--space-6); }
.py-7 { padding-top: var(--space-7); padding-bottom: var(--space-7); }
.py-8 { padding-top: var(--space-8); padding-bottom: var(--space-8); }

/* Обновлённые базовые классы */
.section { padding: var(--space-9) 0; }
.hero { padding: var(--space-9) 0 var(--space-8); }
.page-head { padding: var(--space-7) 0 var(--space-2); }
.page-narrow { padding: var(--space-7) 0 var(--space-9); }
.lk-main { padding: var(--space-6) var(--space-5); }
```

**Применение в templates:**
```html
<!-- Было -->
<div class="grid-2" style="padding-bottom:80px">

<!-- Стало -->
<div class="grid-2 py-7">
```

### 8.2 Замена эмодзи на SVG в LK-навигации

**Было (`lk_student.html`):**
```html
<li><a href="/lk/student" aria-current="page">
  <span class="lk-nav-icon" aria-hidden="true">🏠</span> Обзор
</a></li>
```

**Стало:**
```html
<li><a href="/lk/student" aria-current="page">
  <span class="lk-nav-icon" aria-hidden="true">
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="20" height="20">
      <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>
      <polyline points="9 22 9 12 15 12 15 22"/>
    </svg>
  </span> Обзор
</a></li>
```

### 8.3 Переработка страницы логина

**Было:** 7 кнопок (2 соцсети + 3 быстрый вход + форма email/пароль)

**Стало:** 3 блока, компактная раскладка

```html
<!-- Левая колонка — без изменений -->

<!-- Правая колонка — упрощённая -->
<div class="auth-form-card">
  <h2 class="auth-form-title">Вход в аккаунт</h2>
  <p class="auth-form-subtitle">Выберите способ входа</p>

  <!-- Табы ролей — без изменений -->

  <!-- Соцсети — без изменений -->

  <div class="auth-divider">или</div>

  <!-- Email/пароль — без изменений -->

  <!-- УБРАНО: «быстрый вход» -->
  <!-- УБРАНО: 3 кнопки roleAuth() -->

  <p class="auth-footer">Нет аккаунта? <a href="/about">Узнайте о проекте GO-IT</a></p>
</div>
```

### 8.4 .btn--danger для кнопок отклонения

**CSS (добавить в app.css):**
```css
.btn--danger {
  background: var(--error);
  color: var(--primary-ink);
}
.btn--danger:hover {
  filter: brightness(1.15);
}
[data-contrast="high"] .btn--danger {
  border-color: #000;
  background: var(--error-bg);
  color: var(--error);
}
```

**Применение (`lk_mentor.html`, `lk_partner.html`):**
```html
<!-- Было -->
<a class="btn btn--outline btn--sm" href="/lk/mentor/applications/{{.ID}}/decline">Отклонить</a>

<!-- Стало -->
<a class="btn btn--danger btn--sm" href="/lk/mentor/applications/{{.ID}}/decline">Отклонить</a>
```

### 8.5 Улучшение карточек направлений

**Было (`home.html`):**
```html
<article class="card">
  <div class="ic" aria-hidden="true">{{.Icon}}</div>
  <h3>{{.Title}}</h3>
  <p>{{.Summary}}</p>
  <a class="more" href="/roadmaps/{{.RoadmapSlug}}">Смотреть roadmap →</a>
</article>
```

**Стало:**
```html
<article class="card card--featured">
  <div class="card-icon">{{.Icon}}</div>
  <h3 class="h3">{{.Title}}</h3>
  <p>{{.Summary}}</p>
  <div class="card-actions">
    <a class="btn btn--sm btn--primary" href="/roadmaps/{{.RoadmapSlug}}">Roadmap</a>
    {{if .CaseSlug}}<a class="btn btn--sm" href="/cases/{{.CaseSlug}}">Кейс</a>{{end}}
  </div>
</article>
```

**CSS:**
```css
.card--featured {
  display: flex;
  flex-direction: column;
}
.card--featured .card-icon {
  width: 56px;
  height: 56px;
  font-size: 1.2rem;
  margin-bottom: var(--space-3);
}
.card--featured h3 {
  margin-bottom: var(--space-2);
}
.card--featured p {
  flex-grow: 1;
  margin-bottom: var(--space-4);
}
.card-actions {
  display: flex;
  gap: var(--space-2);
  margin-top: auto;
}
```

### 8.6 Персонализация результата кейса

**Было (`case_result.html`):**
```html
<div class="btn-row" style="margin-top:32px">
  <a class="btn btn--cta" href="/roadmaps/{{.Data.Case.RoadmapSlug}}">Перейти к roadmap</a>
  <a class="btn btn--outline" href="/cases/{{.Data.Case.Slug}}">Пройти заново</a>
  <a class="btn btn--outline" href="/mentors">Найти наставника</a>
</div>
```

**Стало:**
```html
<div style="margin-top:32px">
  <a class="btn btn--cta" href="/roadmaps/{{.Data.Case.RoadmapSlug}}" style="width:100%">
    🗺️ Перейти к вашей дорожной карте
  </a>
  <div class="btn-row" style="margin-top:14px">
    <a class="btn btn--outline" href="/cases/{{.Data.Case.Slug}}">Пройти заново</a>
    <a class="btn btn--outline" href="/mentors">Найти наставника</a>
  </div>
</div>
```

### 8.7 Стандартный breadcrumb-back паттерн

**Для `mentor.html` и `volunteer.html`:**
```html
<div class="wrap">
  <header class="page-head">
    <nav class="breadcrumbs" aria-label="Навигация">
      <a href="/mentors">Менторы</a>
      <span aria-hidden="true">/</span>
      <span aria-current="page">{{.Data.Mentor.DisplayName}}</span>
    </nav>
    <!-- Добавить кнопку "Назад" -->
    <a class="btn btn--ghost btn--sm" href="/mentors" style="margin-top:16px">
      ← Все наставники
    </a>
  </header>
```

### 8.8 Выделение дисклеймера в отдельный блок

**`case_result.html`:**
```html
<!-- Было -->
<p class="hint">{{.Data.Disclaimer}}</p>

<!-- Стало -->
<div class="disclaimer" role="note">
  <svg class="disclaimer-icon" aria-hidden="true" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
    <circle cx="12" cy="12" r="10"/>
    <line x1="12" y1="16" x2="12" y2="12"/>
    <line x1="12" y1="8" x2="12.01" y2="8"/>
  </svg>
  <p>{{.Data.Disclaimer}}</p>
</div>
```

**CSS:**
```css
.disclaimer {
  display: flex;
  gap: 10px;
  background: var(--ok-bg);
  border: 1px solid var(--ok);
  border-radius: var(--radius-sm);
  padding: 14px 16px;
  margin: 24px 0;
  font-size: 0.95rem;
  color: var(--ok);
}
.disclaimer-icon {
  flex: none;
  margin-top: 2px;
  opacity: 0.8;
}
```

---

## 9. Accessibility Audit

### 9.1 Что сделано правильно ✅

| Элемент | Статус |
|---------|--------|
| Skip-link «К содержанию» | ✅ Реализован в `base.html` |
| `lang="ru"` на `<html>` | ✅ |
| `meta name="viewport"` | ✅ |
| `aria-label` на навигациях | ✅ (`.site-header nav`, breadcrumbs, filters) |
| `aria-current="page"` | ✅ |
| `aria-pressed` для табов | ✅ (theme, contrast, role tabs) |
| `aria-describedby` для полей форм | ✅ (application_form) |
| `aria-invalid` при ошибках | ✅ |
| `aria-hidden="true"` для декоративных иконок | ✅ |
| Focus-visible outline | ✅ 3px solid |
| `role="alert"` для ошибок | ✅ |
| `role="progressbar"` | ✅ (cases, LK) |
| `prefers-reduced-motion` | ✅ |
| `data-theme` до рендера (FOUC) | ✅ `theme-init.js` в `<head>` |
| Semantically correct headings | ✅ h1 только один на страницу |

### 9.2 Проблемы ⚠️

| Проблема | Где | Severity |
|----------|-----|----------|
| **Эмодзи как навигация** | LK sidebar | Medium — screen reader читает эмодзи по-разному |
| **Inline table** в `lk_partner.html` | Статистика | Low — таблица не связана с данными, но структура OK |
| **`aria-disabled` vs HTMX** | Кнопки submit | Medium — HTMX не поддерживает `aria-disabled`, нужен `disabled` |
| **Roadmap dialog без JS** | `roadmap_diagram.html` | Medium — `<dialog>` работает только с JS |
| **Отсутствие `aria-label`** на `.lk-nav** | LK sidebar | Low — навигация не имеет role="navigation" |
| **Emoji в icon-btn** | Все LK | Medium — accessibility inconsistent |
| **Filters list** | `mentors.html` | Low — `<ul>` без `role="tablist"` для фильтров |
| **`html { font-size: 17px }`** | Базовый CSS | Low — 17px ≠ 16px стандарт, может ломать user overrides |

### 9.3 Рекомендации по доступности

1. **Заменить эмодзи на SVG** — это решит большинство проблем
2. **Добавить `role="navigation"`** к `.lk-nav`
3. **Проверить контраст** всех текстовых элементов:
   - `--muted` (#4b5563) на `--bg` (#ffffff) = **6.8:1** ✅
   - `--hero-muted` (#c9d6f2) на `--hero-bg` (#0a1240) = **6.2:1** ✅
   - `.tag` (primary) на `--surface` — **4.5:1** ✅
   - `--border` декоративный — 3:1 ✅
4. **17px base font-size** — оставить, но документировать как осознанное решение
5. **Добавить `aria-live="polite"`** для HTMX-подмен контента

---

## 10. Дорожная карта улучшений

### Фаза 1: Непосредственно (1–2 дня)

```
🔴 P0 — Критичные исправления:

[ ] 1.1  SVG-иконки для LK-навигации (13 иконок)
[ ] 1.2  Система spacing tokens + migration 5 основных страниц
[ ] 1.3  .btn--danger для кнопок «Отклонить»
[ ] 1.4  Убрать «быстрый вход» со страницы логина
[ ] 1.5  Стандартизация заголовков (убрать inline font-size)
```

### Фаза 2: Следующая итерация (3–5 дней)

```
🟡 P1 — Улучшение UX:

[ ] 2.1  Cover-изображения для карточек направлений
[ ] 2.2  Кнопка «Назад к списку» на detail-страницах
[ ] 2.3  Персонализация результата кейса (primary CTA)
[ ] 2.4  Empty states для LK-разделов
[ ] 2.5  Выделение дисклеймера в отдельный блок
[ ] 2.6  Добавление `role="navigation"` к `.lk-nav`
```

### Фаза 3: Будущее (1–2 недели)

```
🟢 P2 — polish и продвинутые улучшения:

[ ] 3.1  Cover-изображения для статей блога
[ ] 3.2  Обновлённый footer (убрать «студенты»)
[ ] 3.3  Page-header декор на list-страницах
[ ] 3.4  Страница «Мои заявки» для студента
[ ] 3.5  CSS .grid-4 и .gap-* утилиты
[ ] 3.6  `aria-live="polite"` для HTMX
[ ] 3.7  Персонализированный лендинг
```

---

## Приложение A: Сводная таблица страниц

| Страница | Маршрут | Роль | Оценка | Приоритет правок |
|----------|---------|------|--------|-----------------|
| Главная | `/` | Guest | ⭐⭐⭐⭐☆ | P1 — cover-images |
| О проекте | `/about` | Guest | ⭐⭐⭐⭐☆ | P1 — CTA accent |
| Направления | `/directions` | Guest | ⭐⭐⭐☆☆ | P1 — cover-images |
| Направление | `/directions/{slug}` | Guest | ⭐⭐⭐☆☆ | P0 — heading, P1 — back-btn |
| Кейсы | `/cases` | Guest | ⭐⭐⭐☆☆ | P1 — cards |
| Кейс вопрос | `/cases/{slug}` | Guest/Student | ⭐⭐⭐⭐☆ | P2 — save progress |
| Кейс результат | `/cases/{slug}/result` | Guest/Student | ⭐⭐⭐⭐☆ | P1 — personalize CTA, disclaimer |
| Менторы | `/mentors` | Guest | ⭐⭐⭐☆☆ | P1 — cover-images |
| Ментор | `/mentors/{slug}` | Guest/Student | ⭐⭐⭐☆☆ | P0 — back-btn, heading |
| Волонтёрство | `/volunteering` | Guest | ⭐⭐⭐☆☆ | P1 — cover-images |
| Задача | `/volunteering/{slug}` | Guest/Student | ⭐⭐⭐☆☆ | P0 — back-btn |
| Roadmaps | `/roadmaps` | Guest | ⭐⭐⭐⭐☆ | P1 — cover-images |
| Roadmap | `/roadmaps/{slug}` | Guest | ⭐⭐⭐⭐☆ | P2 — topic collapsible |
| Блог | `/blog` | Guest | ⭐⭐⭐☆☆ | P2 — cover-images |
| Статья | `/blog/{slug}` | Guest | ⭐⭐⭐⭐☆ | P2 — cover-image |
| Логин | `/auth` | Guest | ⭐⭐☆☆☆ | P0 — simplify |
| ЛК Ученик | `/lk/student` | Student | ⭐⭐⭐☆☆ | P0 — SVG icons |
| ЛК Ментор | `/lk/mentor` | Mentor | ⭐⭐⭐☆☆ | P0 — SVG icons, btn--danger |
| ЛК Партнёр | `/lk/partner` | Partner | ⭐⭐⭐☆☆ | P0 — SVG icons, btn--danger |
| Ошибка | `/error` | Guest | ⭐⭐⭐☆☆ | P2 — improved design |
| 404 | `/` (fallback) | Guest | ⭐⭐⭐⭐☆ | — |

---

## Приложение B: CSS-токены для добавления

```css
/* Добавить в :root (в начало файла, после существующих токенов) */

/* Spacing system */
--space-1: 4px;
--space-2: 8px;
--space-3: 12px;
--space-4: 16px;
--space-5: 24px;
--space-6: 32px;
--space-7: 48px;
--space-8: 64px;
--space-9: 80px;

/* New button variants */
--btn-danger-bg: var(--error);
--btn-danger-text: var(--primary-ink);

/* Typography tokens (use these instead of inline h1-h6 sizing) */
--text-h1: clamp(2rem, 5.2vw, 3.4rem);
--text-h2: clamp(1.5rem, 3.4vw, 2.2rem);
--text-h3: 1.2rem;
--text-h4: 1.1rem;

/* Component-specific tokens */
--card-padding: 28px;
--card-radius: var(--radius);
--lk-sidebar-width: 280px;
--content-max-width: 1200px;
```

---

*Документ подготовлен для команды разработки GO-IT. Все рекомендации учитывают SSR-архитектуру, `html/template` ограничения и систему CSS Custom Properties.*
