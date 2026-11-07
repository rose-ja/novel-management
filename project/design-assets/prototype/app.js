/* =========================================================
   小说设定管理系统 · 高保真原型交互
   仅用于设计基线演示：无后端、数据为静态示例。
   ========================================================= */
(function () {
  'use strict';

  const ROUTES = ['search', 'character', 'faction', 'ranking', 'system', 'import', 'trash'];
  const views = Array.from(document.querySelectorAll('.view'));
  const navItems = Array.from(document.querySelectorAll('[data-nav]'));
  const main = document.getElementById('main');

  const toastHost = document.getElementById('toast-host');
  const confirmDialog = document.getElementById('confirm-dialog');
  const confirmTitle = document.getElementById('confirm-dialog-title');
  const confirmDesc = document.getElementById('confirm-dialog-desc');
  const confirmActionBtn = document.getElementById('confirm-dialog-action');
  let pendingConfirm = null;

  /* ---------------------------------------------------------
     Toast
     --------------------------------------------------------- */
  const TOAST_ICONS = {
    success: 'i-check',
    error: 'i-alert',
    info: 'i-info'
  };

  function showToast(message, options) {
    const opts = options || {};
    const type = opts.type || 'success';

    while (toastHost.children.length >= 3) {
      toastHost.firstElementChild.remove();
    }

    const toast = document.createElement('div');
    toast.className = 'toast' + (type === 'error' ? ' toast--error' : '');

    const iconWrap = document.createElement('span');
    iconWrap.className = 'toast__icon';
    iconWrap.innerHTML = '<svg class="icon icon--sm" aria-hidden="true"><use href="#' +
      (TOAST_ICONS[type] || TOAST_ICONS.info) + '"></use></svg>';

    const text = document.createElement('span');
    text.className = 'toast__message';
    text.textContent = message;

    toast.appendChild(iconWrap);
    toast.appendChild(text);

    if (opts.action) {
      const actionBtn = document.createElement('button');
      actionBtn.className = 'toast__action';
      actionBtn.type = 'button';
      actionBtn.textContent = opts.action.label;
      actionBtn.addEventListener('click', function () {
        dismissToast(toast);
        opts.action.run();
      });
      toast.appendChild(actionBtn);
    }

    toastHost.appendChild(toast);

    const lifetime = opts.action ? 6000 : 4000;
    setTimeout(function () { dismissToast(toast); }, lifetime);
  }

  function dismissToast(toast) {
    if (!toast || !toast.isConnected || toast.classList.contains('is-leaving')) return;
    toast.classList.add('is-leaving');
    setTimeout(function () { toast.remove(); }, 300);
  }

  /* ---------------------------------------------------------
     路由
     --------------------------------------------------------- */
  function rawHash() {
    return location.hash.replace('#', '');
  }

  function syncRoute() {
    const raw = rawHash();
    if (raw === 'main') return;

    let route = raw;
    if (ROUTES.indexOf(route) === -1) {
      if (raw) {
        showToast('「' + raw + '」页面不在本原型覆盖范围，已返回检索。', { type: 'info' });
      }
      route = 'search';
      history.replaceState(null, '', '#search');
    }

    views.forEach(function (view) {
      view.classList.toggle('active', view.dataset.view === route);
    });

    navItems.forEach(function (item) {
      const isActive = item.dataset.nav === route;
      item.classList.toggle('active', isActive);
      if (isActive) {
        item.setAttribute('aria-current', 'page');
      } else {
        item.removeAttribute('aria-current');
      }
    });

    closeWorkMenu();
    if (main) main.scrollTop = 0;
  }

  window.addEventListener('hashchange', syncRoute);

  /* ---------------------------------------------------------
     作品切换器
     --------------------------------------------------------- */
  const workSwitcher = document.getElementById('work-switcher');
  const workMenu = document.getElementById('work-menu');
  const currentWorkName = document.getElementById('current-work-name');

  function closeWorkMenu() {
    if (!workMenu || workMenu.hidden) return;
    workMenu.hidden = true;
    workSwitcher.setAttribute('aria-expanded', 'false');
  }

  function toggleWorkMenu() {
    const willOpen = workMenu.hidden;
    workMenu.hidden = !willOpen;
    workSwitcher.setAttribute('aria-expanded', String(willOpen));
    if (willOpen) {
      const first = workMenu.querySelector('.work-menu__item');
      if (first) first.focus();
    }
  }

  if (workSwitcher) {
    workSwitcher.addEventListener('click', function (event) {
      event.stopPropagation();
      toggleWorkMenu();
    });
  }

  document.addEventListener('click', function (event) {
    if (!workMenu || workMenu.hidden) return;
    if (!event.target.closest('.topbar__work')) closeWorkMenu();
  });

  document.addEventListener('keydown', function (event) {
    if (event.key === 'Escape' && workMenu && !workMenu.hidden) {
      closeWorkMenu();
      workSwitcher.focus();
    }
  });

  /* ---------------------------------------------------------
     检索
     --------------------------------------------------------- */
  const searchInput = document.getElementById('search-input');
  const filterPanel = document.getElementById('filter-panel');
  const searchModeButtons = Array.from(document.querySelectorAll('[data-search-mode]'));
  const filterSelects = Array.from(document.querySelectorAll('#filter-panel select'));

  searchModeButtons.forEach(function (btn) {
    btn.addEventListener('click', function () {
      searchModeButtons.forEach(function (other) {
        other.setAttribute('aria-pressed', String(other === btn));
      });
      const isConditional = btn.dataset.searchMode === 'conditional';
      if (filterPanel) filterPanel.hidden = !isConditional;
      if (searchInput) {
        searchInput.placeholder = isConditional
          ? '输入名称、别名、称号或关键词…'
          : '输入名称、别名、称号或关键词；输入即搜';
      }
    });
  });

  if (searchInput) {
    searchInput.addEventListener('keydown', function (event) {
      if (event.key !== 'Enter') return;
      event.preventDefault();
      const value = searchInput.value.trim();
      if (!value) {
        showToast('请输入检索关键词。', { type: 'error' });
        return;
      }
      showToast('已在当前作品内检索「' + value + '」：12 条结果。', { type: 'info' });
    });
  }

  /* ---------------------------------------------------------
     人物详情：内联编辑 / 标签输入 / 校对
     --------------------------------------------------------- */
  const characterViewMode = document.getElementById('character-view-mode');
  const characterEditMode = document.getElementById('character-edit-mode');
  const characterEditForm = document.getElementById('character-edit-form');
  const editNameInput = document.getElementById('edit-name');

  function setCharacterEditing(editing) {
    if (!characterViewMode || !characterEditMode) return;
    characterViewMode.hidden = editing;
    characterEditMode.hidden = !editing;
    if (editing && editNameInput) {
      editNameInput.focus();
      editNameInput.select();
    }
  }

  if (characterEditForm) {
    characterEditForm.addEventListener('submit', function (event) {
      event.preventDefault();
      const name = editNameInput.value.trim();
      const existingError = characterEditForm.querySelector('.form-error--name');
      if (existingError) existingError.remove();
      editNameInput.removeAttribute('aria-invalid');

      if (!name) {
        editNameInput.setAttribute('aria-invalid', 'true');
        const error = document.createElement('span');
        error.className = 'form-error form-error--name';
        error.textContent = '主名称不能为空。';
        editNameInput.insertAdjacentElement('afterend', error);
        editNameInput.focus();
        showToast('主名称不能为空，未保存。', { type: 'error' });
        return;
      }

      const title = document.getElementById('character-title');
      if (title) title.textContent = name;
      const crumb = document.querySelector('#view-character .breadcrumb [aria-current="page"]');
      if (crumb) crumb.textContent = name;

      setCharacterEditing(false);
      showToast('人物「' + name + '」已保存，关联页面与检索索引已同步。');
    });
  }

  function addChip(input) {
    if (!input) return;
    const value = input.value.trim();
    if (!value) return;

    const wrap = input.closest('.tag-input');
    if (!wrap) return;

    if (Array.from(wrap.querySelectorAll('.chip')).some(function (chip) {
      return chip.firstChild && chip.firstChild.textContent.trim() === value;
    })) {
      showToast('「' + value + '」已存在。', { type: 'error' });
      input.value = '';
      return;
    }

    const chip = document.createElement('span');
    chip.className = 'chip';
    chip.appendChild(document.createTextNode(value));

    const removeBtn = document.createElement('button');
    removeBtn.className = 'chip__remove';
    removeBtn.type = 'button';
    removeBtn.setAttribute('aria-label', '移除 ' + value);
    removeBtn.textContent = '×';
    chip.appendChild(removeBtn);

    wrap.insertBefore(chip, input);
    input.value = '';
    input.focus();
  }

  ['edit-alias', 'edit-title'].forEach(function (id) {
    const input = document.getElementById(id);
    if (!input) return;
    input.addEventListener('keydown', function (event) {
      if (event.key === 'Enter') {
        event.preventDefault();
        addChip(input);
      }
      if (event.key === 'Backspace' && !input.value) {
        const wrap = input.closest('.tag-input');
        const chips = wrap ? wrap.querySelectorAll('.chip') : [];
        if (chips.length) chips[chips.length - 1].remove();
      }
    });
  });

  /* ---------------------------------------------------------
     势力：成员范围
     --------------------------------------------------------- */
  const memberScopeButtons = Array.from(document.querySelectorAll('[data-member-scope]'));
  const factionMemberBody = document.getElementById('faction-member-body');

  function setMemberScope(scope) {
    memberScopeButtons.forEach(function (btn) {
      btn.setAttribute('aria-pressed', String(btn.dataset.memberScope === scope));
    });
    if (!factionMemberBody) return;
    Array.from(factionMemberBody.querySelectorAll('tr[data-scope="all"]')).forEach(function (row) {
      row.hidden = scope !== 'all';
    });
  }

  memberScopeButtons.forEach(function (btn) {
    btn.addEventListener('click', function () {
      setMemberScope(btn.dataset.memberScope);
      showToast(btn.dataset.memberScope === 'all'
        ? '已显示含下级成员：共 34 人。'
        : '仅显示直属成员：共 12 人。', { type: 'info' });
    });
  });

  /* ---------------------------------------------------------
     榜单
     --------------------------------------------------------- */
  const seatBody = document.getElementById('seat-body');
  const seatSearch = document.getElementById('seat-search');
  const jumpSeatInput = document.getElementById('jump-seat');
  const rankingView = document.getElementById('view-ranking');
  let seatPage = 1;
  let seatQuery = '';

  const occupiedSeats = {
    1: { name: '玄冥', faction: '归墟遗族', realm: '神境', note: '旧约见证' },
    2: { name: '沈照夜', faction: '归墟遗族', realm: '圣者境 · 大成', note: '归墟之主' },
    3: { name: '司空摘天', faction: '山海妖族', realm: '帝境 · 五重天', note: '帝境之首' },
    4: { name: '青玄', faction: '万族', realm: '圣者境 · 大成', note: '—' },
    5: { name: '姜黎', faction: '青丘分支', realm: '圣者境 · 大成', note: '青丘之首' },
    6: { name: '陆青崖', faction: '山海妖族', realm: '圣者境 · 大成', note: '—' },
    8: { name: '白无咎', faction: '白狐分支', realm: '圣者境 · 初阶', note: '掌族中典籍' },
    9: { name: '顾长庚', faction: '玄狐分支', realm: '圣者境 · 初阶', note: '驻归墟边境' },
    11: { name: '云栖', faction: '山海妖族', realm: '圣者境 · 初阶', note: '—' },
    13: { name: '谢孤鸿', faction: '山海妖族', realm: '凡境 · 九层', note: '—' },
    14: { name: '丹阳', faction: '天枢院', realm: '圣者境 · 初阶', note: '—' },
    17: { name: '赤霄', faction: '万族', realm: '圣者境 · 大成', note: '—' },
    19: { name: '玄同', faction: '归墟遗族', realm: '圣者境 · 初阶', note: '—' },
    22: { name: '洛书', faction: '天枢院', realm: '凡境 · 九层', note: '—' },
    26: { name: '沉舟', faction: '山海妖族', realm: '凡境 · 九层', note: '—' },
    28: { name: '祁连', faction: '万族', realm: '圣者境 · 初阶', note: '—' },
    31: { name: '闻人', faction: '青丘分支', realm: '凡境 · 九层', note: '—' },
    33: { name: '苏合', faction: '白狐分支', realm: '凡境 · 九层', note: '—' },
    38: { name: '陆离', faction: '天枢院', realm: '圣者境 · 初阶', note: '—' },
    42: { name: '白泽', faction: '山海妖族', realm: '圣者境 · 大成', note: '—' },
    45: { name: '重明', faction: '万族', realm: '圣者境 · 初阶', note: '—' },
    47: { name: '凤栖', faction: '青丘分支', realm: '凡境 · 九层', note: '—' },
    49: { name: '偃师', faction: '玄狐分支', realm: '凡境 · 九层', note: '—' }
  };

  function avatarText(name) {
    return name ? name.trim().charAt(0) : '·';
  }

  function seatRowHTML(seat) {
    const person = occupiedSeats[seat];
    if (!person) {
      return '<tr class="seat-row--empty" data-seat="' + seat + '">' +
        '<td><span class="seat-number">' + seat + '</span></td>' +
        '<td><span class="seat-empty__label">空席</span></td>' +
        '<td><span class="text-secondary">—</span></td>' +
        '<td><span class="text-secondary">—</span></td>' +
        '<td><div class="seat-actions">' +
          '<button class="btn btn--ghost btn--sm" type="button" data-action="assign-seat" data-seat="' + seat + '">指派</button>' +
        '</div></td>' +
      '</tr>';
    }

    const note = person.note && person.note !== '—'
      ? person.note
      : '<span class="text-secondary">—</span>';

    return '<tr data-seat="' + seat + '">' +
      '<td><span class="seat-number' + (seat <= 5 ? ' seat-number--top' : '') + '">' + seat + '</span></td>' +
      '<td><span class="member-cell">' +
        '<span class="avatar" aria-hidden="true">' + avatarText(person.name) + '</span>' +
        '<a class="member-cell__name" href="#character">' + person.name + '</a>' +
      '</span></td>' +
      '<td>' + person.faction + ' · ' + person.realm + '</td>' +
      '<td>' + note + '</td>' +
      '<td><div class="seat-actions">' +
        '<button class="icon-btn icon-btn--sm" type="button" aria-label="第 ' + seat + ' 席操作" data-action="seat-menu" data-seat="' + seat + '">' +
          '<svg class="icon icon--sm" aria-hidden="true"><use href="#i-more"></use></svg>' +
        '</button>' +
      '</div></td>' +
    '</tr>';
  }

  function updateRankingStats() {
    if (!rankingView) return;
    const occupied = Object.keys(occupiedSeats).length;
    const values = rankingView.querySelectorAll('.ranking-stat__value');
    if (values.length >= 3) {
      values[0].textContent = '100';
      values[1].textContent = String(occupied);
      values[2].textContent = String(100 - occupied);
    }
  }

  function updateRankingPager() {
    if (!rankingView) return;
    const label = seatPage === 1 ? '第 1–50 席 / 共 100 席' : '第 51–100 席 / 共 100 席';
    const indicator = rankingView.querySelector('.page-indicator');
    if (indicator) indicator.textContent = label;

    const pagination = rankingView.querySelector('.pagination');
    if (pagination) {
      const total = pagination.querySelector('.pagination__total');
      if (total) total.textContent = label;

      const numeric = Array.from(pagination.querySelectorAll('.page-btn')).filter(function (btn) {
        return /^\d+$/.test(btn.textContent.trim());
      });
      numeric.forEach(function (btn) {
        const isActive = Number(btn.textContent.trim()) === seatPage;
        btn.classList.toggle('page-btn--active', isActive);
        if (isActive) {
          btn.setAttribute('aria-current', 'page');
        } else {
          btn.removeAttribute('aria-current');
        }
      });

      pagination.querySelectorAll('.page-btn').forEach(function (btn) {
        if (btn.getAttribute('aria-label') === '上一页') btn.disabled = seatPage === 1;
        if (btn.getAttribute('aria-label') === '下一页') btn.disabled = seatPage === 2;
      });
    }
  }

  function applySeatFilter() {
    if (!seatBody) return;
    const query = seatQuery.trim().toLowerCase();
    Array.from(seatBody.querySelectorAll('tr')).forEach(function (row) {
      row.hidden = Boolean(query) && row.textContent.toLowerCase().indexOf(query) === -1;
    });
  }

  function renderSeats(page) {
    if (!seatBody) return;
    seatPage = page;
    const start = page === 1 ? 1 : 51;
    const end = page === 1 ? 50 : 100;
    let html = '';
    for (let seat = start; seat <= end; seat += 1) {
      html += seatRowHTML(seat);
    }
    seatBody.innerHTML = html;
    updateRankingStats();
    updateRankingPager();
    applySeatFilter();
  }

  function flashRow(row) {
    if (!row) return;
    row.style.boxShadow = 'inset 0 0 0 2px var(--color-brand-500)';
    setTimeout(function () { row.style.boxShadow = ''; }, 1200);
  }

  function jumpToSeat() {
    const value = Number(jumpSeatInput.value);
    if (!value || value < 1 || value > 100) {
      showToast('请输入 1–100 之间的席位号。', { type: 'error' });
      jumpSeatInput.focus();
      return;
    }
    const targetPage = value <= 50 ? 1 : 2;
    if (targetPage !== seatPage) renderSeats(targetPage);
    const row = seatBody.querySelector('tr[data-seat="' + value + '"]');
    if (row) {
      row.scrollIntoView({ block: 'center' });
      flashRow(row);
    }
    showToast('已定位到第 ' + value + ' 席。', { type: 'info' });
  }

  if (seatSearch) {
    seatSearch.addEventListener('input', function () {
      seatQuery = seatSearch.value;
      applySeatFilter();
    });
  }

  if (jumpSeatInput) {
    jumpSeatInput.addEventListener('keydown', function (event) {
      if (event.key === 'Enter') {
        event.preventDefault();
        jumpToSeat();
      }
    });
  }

  /* ---------------------------------------------------------
     指派席位 Dialog
     --------------------------------------------------------- */
  const assignDialog = document.getElementById('assign-dialog');
  const assignTitle = document.getElementById('assign-dialog-title');
  const assignConfirm = document.getElementById('assign-confirm');
  const assignSearch = assignDialog ? assignDialog.querySelector('.dialog__search input') : null;
  let assignSeat = null;
  let selectedCandidate = null;

  function closeDialog(dialog) {
    if (dialog && dialog.open) dialog.close();
  }

  function clearCandidateSelection() {
    selectedCandidate = null;
    if (!assignDialog) return;
    assignDialog.querySelectorAll('.candidate-row').forEach(function (row) {
      row.classList.remove('is-selected');
      row.setAttribute('aria-selected', 'false');
    });
    if (assignConfirm) assignConfirm.disabled = true;
  }

  function openAssignDialog(seat) {
    if (!assignDialog) return;
    assignSeat = Number(seat);
    if (assignTitle) assignTitle.textContent = '指派到第 ' + assignSeat + ' 席';
    if (assignConfirm) assignConfirm.textContent = '指派到第 ' + assignSeat + ' 席';
    if (assignSearch) assignSearch.value = '';
    assignDialog.querySelectorAll('.candidate-row').forEach(function (row) { row.hidden = false; });
    clearCandidateSelection();
    assignDialog.showModal();
  }

  if (assignSearch) {
    assignSearch.addEventListener('input', function () {
      const query = assignSearch.value.trim().toLowerCase();
      assignDialog.querySelectorAll('.candidate-row').forEach(function (row) {
        row.hidden = Boolean(query) && row.textContent.toLowerCase().indexOf(query) === -1;
      });
    });
  }

  if (assignConfirm) {
    assignConfirm.addEventListener('click', function () {
      if (!selectedCandidate || !assignSeat) return;
      const row = selectedCandidate;
      const name = row.dataset.candidate;
      const meta = row.querySelector('.candidate-row__meta');
      const metaParts = meta ? meta.textContent.split(' / ') : [];
      occupiedSeats[assignSeat] = {
        name: name,
        faction: metaParts[0] || '—',
        realm: metaParts[1] || '—',
        note: '手工指派'
      };
      row.remove();
      renderSeats(seatPage);
      closeDialog(assignDialog);
      showToast('已将「' + name + '」指派到第 ' + assignSeat + ' 席。');
      assignSeat = null;
      clearCandidateSelection();
    });
  }

  /* ---------------------------------------------------------
     体系配置
     --------------------------------------------------------- */
  const categoryDetails = {
    境界: {
      desc: '内置体系类别，名称固定，不可停用或删除。人物可引用 0..1 个境界节点；未启用的类别不会出现在人物表单与详情页。',
      meta: ['普通', '24', '86 人', '两级']
    },
    星命: {
      desc: '自定义体系类别，用于星命、命格一类的设定。人物可引用 0..1 个节点；节点最多两级，同级可排序。',
      meta: ['普通', '12', '8 人', '两级']
    },
    圣体: {
      desc: '自定义体系类别，用于血脉、体质一类的设定。人物可引用 0..1 个节点；删除节点前必须无引用。',
      meta: ['普通', '9', '6 人', '两级']
    },
    纪元: {
      desc: '自定义体系类别，用于时代、纪元划分。名称自由填写；节点最多两级，可被大事记等实体引用。',
      meta: ['普通', '6', '0 人', '两级']
    }
  };

  function selectCategory(btn) {
    const name = btn.dataset.category;
    if (btn.classList.contains('category-item--disabled')) {
      showToast('「' + name + '」已停用，启用前需先检查引用。', { type: 'info' });
      return;
    }

    document.querySelectorAll('.category-item').forEach(function (item) {
      item.classList.toggle('active', item === btn);
      if (item === btn) {
        item.setAttribute('aria-current', 'true');
      } else {
        item.removeAttribute('aria-current');
      }
    });

    const detail = categoryDetails[name];
    if (!detail) return;

    const title = document.querySelector('.category-title');
    if (title) title.textContent = name;

    const desc = document.querySelector('.category-desc');
    if (desc) desc.textContent = detail.desc;

    const titleRow = document.querySelector('.category-title-row');
    if (titleRow) {
      titleRow.querySelectorAll('.badge').forEach(function (badge) { badge.remove(); });
      if (name === '境界') {
        const builtIn = document.createElement('span');
        builtIn.className = 'badge badge--neutral';
        builtIn.textContent = '内置';
        const required = document.createElement('span');
        required.className = 'badge badge--brand';
        required.textContent = '必开';
        titleRow.appendChild(builtIn);
        titleRow.appendChild(required);
      }
    }

    const metaValues = document.querySelectorAll('.category-meta__value');
    detail.meta.forEach(function (value, index) {
      if (metaValues[index]) metaValues[index].textContent = value;
    });

    showToast('已切换到「' + name + '」类别。节点树当前仅展示境界示例。', { type: 'info' });
  }

  function toggleNode(btn) {
    const expanded = btn.getAttribute('aria-expanded') === 'true';
    const nextState = !expanded;
    const row = btn.closest('.tree-row');
    const nameEl = row ? row.querySelector('.tree-name') : null;
    const name = nameEl ? nameEl.textContent : '';

    btn.setAttribute('aria-expanded', String(nextState));
    btn.setAttribute('aria-label', (nextState ? '收起' : '展开') + name + '下级');

    const use = btn.querySelector('use');
    if (use) {
      use.setAttribute('href', nextState ? '#i-chevron-down' : '#i-chevron-right');
    }

    if (!row) return;
    let sibling = row.nextElementSibling;
    while (sibling && sibling.classList.contains('tree-row--child')) {
      sibling.hidden = !nextState;
      sibling = sibling.nextElementSibling;
    }
  }

  /* ---------------------------------------------------------
     导入向导
     --------------------------------------------------------- */
  const importSteps = Array.from(document.querySelectorAll('.stepper .step'));
  const importPanel1 = document.getElementById('import-step-panel-1');
  const importPanel2 = document.getElementById('import-step-panel-2');
  const importAside1 = document.getElementById('import-aside-actions-1');
  const importAside2 = document.getElementById('import-aside-actions-2');

  function setImportStep(step) {
    importSteps.forEach(function (btn, index) {
      const number = index + 1;
      btn.classList.remove('is-done', 'is-active', 'is-error', 'is-disabled');
      btn.disabled = false;
      btn.removeAttribute('aria-current');

      const indexEl = btn.querySelector('.step__index');
      if (number < step) {
        btn.classList.add('is-done');
        if (indexEl) {
          indexEl.innerHTML = '<svg class="icon icon--sm" aria-hidden="true"><use href="#i-check"></use></svg>';
        }
      } else if (number === step) {
        btn.classList.add(step === 2 ? 'is-error' : 'is-active');
        btn.setAttribute('aria-current', 'step');
        if (indexEl) {
          indexEl.innerHTML = step === 2
            ? '<svg class="icon icon--sm" aria-hidden="true"><use href="#i-alert"></use></svg>'
            : String(number);
        }
      } else {
        btn.classList.add('is-disabled');
        btn.disabled = true;
        if (indexEl) indexEl.textContent = String(number);
      }
    });

    if (importPanel1) importPanel1.hidden = step !== 1;
    if (importPanel2) importPanel2.hidden = step !== 2;
    if (importAside1) importAside1.hidden = step !== 1;
    if (importAside2) importAside2.hidden = step !== 2;
  }

  /* ---------------------------------------------------------
     回收站
     --------------------------------------------------------- */
  const trashBody = document.getElementById('trash-body');
  const trashTableWrap = document.getElementById('trash-table-wrap');
  const trashEmpty = document.getElementById('trash-empty');
  const trashSearch = document.getElementById('trash-search');
  const trashType = document.getElementById('trash-type');
  const trashWork = document.getElementById('trash-work');
  const trashPagination = document.getElementById('trash-pagination');
  const trashView = document.getElementById('view-trash');

  function trashRows() {
    return trashBody ? Array.from(trashBody.querySelectorAll('tr')) : [];
  }

  function setTrashEmptyState(kind) {
    if (!trashEmpty) return;
    const titleEl = trashEmpty.querySelector('.empty-state__title');
    const textEl = trashEmpty.querySelector('.empty-state__text');
    if (kind === 'filtered') {
      if (titleEl) titleEl.textContent = '没有符合筛选的回收站项';
      if (textEl) textEl.textContent = '调整搜索词、类型或原作品筛选后再试。';
    } else {
      if (titleEl) titleEl.textContent = '回收站为空';
      if (textEl) textEl.textContent = '删除的人物、势力、地点、榜单、体系节点与作品会先进入这里；可在快照保留期内恢复。';
    }
  }

  function refreshTrash() {
    const rows = trashRows();
    const query = trashSearch ? trashSearch.value.trim().toLowerCase() : '';
    const type = trashType ? trashType.value : '全部类型';
    const work = trashWork ? trashWork.value : '全部作品';

    let visible = 0;
    rows.forEach(function (row) {
      const cells = row.cells;
      const name = (row.dataset.name || '').toLowerCase();
      const typeText = cells[0] ? cells[0].textContent.trim() : '';
      const workText = cells[2] ? cells[2].textContent.trim() : '';
      const nameText = cells[1] ? cells[1].textContent.trim() : '';

      const matchQuery = !query || name.indexOf(query) !== -1 || nameText.toLowerCase().indexOf(query) !== -1;
      const matchType = type === '全部类型' || typeText === type;
      const matchWork = work === '全部作品' || workText === work ||
        (workText === '—' && nameText.indexOf(work) !== -1);

      const matches = matchQuery && matchType && matchWork;
      row.hidden = !matches;
      if (matches) visible += 1;
    });

    const total = rows.length;
    const subtitle = trashView ? trashView.querySelector('.page-subtitle') : null;
    if (subtitle) {
      subtitle.textContent = '跨作品查看已删除项 · 共 ' + total + ' 项 · 恢复后可继续编辑';
    }

    if (trashPagination) {
      const totalEl = trashPagination.querySelector('.pagination__total');
      if (totalEl) totalEl.textContent = '每页 20 项 · 共 ' + total + ' 项';
    }

    const emptyBtn = trashView ? trashView.querySelector('[data-action="empty-trash"]') : null;
    if (emptyBtn) {
      emptyBtn.disabled = total === 0;
      emptyBtn.title = total === 0 ? '回收站已为空' : '';
    }

    if (total === 0) {
      if (trashTableWrap) trashTableWrap.hidden = true;
      if (trashEmpty) trashEmpty.hidden = false;
      setTrashEmptyState('empty');
    } else if (visible === 0) {
      if (trashTableWrap) trashTableWrap.hidden = true;
      if (trashEmpty) trashEmpty.hidden = false;
      setTrashEmptyState('filtered');
    } else {
      if (trashTableWrap) trashTableWrap.hidden = false;
      if (trashEmpty) trashEmpty.hidden = true;
    }
  }

  function removeTrashRow(row, message) {
    if (!row) return;
    row.remove();
    refreshTrash();
    showToast(message);
  }

  /* ---------------------------------------------------------
     确认对话框
     --------------------------------------------------------- */
  function openConfirm(options) {
    if (!confirmDialog) return;
    confirmTitle.textContent = options.title;
    confirmDesc.textContent = options.desc;
    confirmActionBtn.textContent = options.actionLabel || '确认';
    confirmActionBtn.className = options.tone === 'primary' ? 'btn btn--primary' : 'btn btn--danger-solid';
    pendingConfirm = options.onConfirm || null;
    confirmDialog.showModal();
    const cancelBtn = confirmDialog.querySelector('[data-close-dialog]');
    if (cancelBtn) cancelBtn.focus();
  }

  if (confirmActionBtn) {
    confirmActionBtn.addEventListener('click', function () {
      const action = pendingConfirm;
      pendingConfirm = null;
      if (confirmDialog.open) confirmDialog.close();
      if (action) action();
    });
  }

  if (confirmDialog) {
    confirmDialog.addEventListener('close', function () { pendingConfirm = null; });
  }

  [confirmDialog, assignDialog].forEach(function (dialog) {
    if (!dialog) return;
    dialog.addEventListener('click', function (event) {
      if (event.target === dialog) dialog.close();
    });
  });

  /* ---------------------------------------------------------
     事件分发
     --------------------------------------------------------- */
  document.addEventListener('click', function (event) {
    const uncovered = event.target.closest('[data-uncovered]');
    if (uncovered) {
      event.preventDefault();
      showToast(uncovered.dataset.uncovered || '该页面未在本原型中覆盖。', { type: 'info' });
      return;
    }

    const workItem = event.target.closest('.work-menu__item[data-work]');
    if (workItem) {
      const work = workItem.dataset.work;
      if (currentWorkName) currentWorkName.textContent = work;
      document.querySelectorAll('.work-menu__item[data-work]').forEach(function (item) {
        const selected = item === workItem;
        item.classList.toggle('is-selected', selected);
        item.setAttribute('aria-checked', String(selected));
      });
      closeWorkMenu();
      showToast('已切换作品至《' + work + '》。');
      return;
    }

    const candidate = event.target.closest('.candidate-row');
    if (candidate) {
      clearCandidateSelection();
      selectedCandidate = candidate;
      candidate.classList.add('is-selected');
      candidate.setAttribute('aria-selected', 'true');
      assignConfirm.disabled = false;
      return;
    }

    const category = event.target.closest('.category-item');
    if (category) {
      selectCategory(category);
      return;
    }

    const importStep = event.target.closest('[data-import-step]');
    if (importStep) {
      const step = Number(importStep.dataset.importStep);
      setImportStep(step);
      if (step === 1) showToast('已返回上传步骤；修改文件后重新校验。', { type: 'info' });
      return;
    }

    const actionEl = event.target.closest('[data-action]');
    if (!actionEl) return;

    const action = actionEl.dataset.action;

    switch (action) {
      case 'clear-search':
        if (searchInput) {
          searchInput.value = '';
          searchInput.focus();
        }
        showToast('已清空检索关键词。', { type: 'info' });
        break;

      case 'reset-filters':
        filterSelects.forEach(function (select) { select.selectedIndex = 0; });
        showToast('已重置检索条件。', { type: 'info' });
        break;

      case 'run-filter':
        showToast('已按当前条件检索：12 条结果。', { type: 'info' });
        break;

      case 'edit-character':
        setCharacterEditing(true);
        break;

      case 'cancel-edit-character':
        setCharacterEditing(false);
        showToast('已取消编辑，未保存修改。', { type: 'info' });
        break;

      case 'mark-proofread':
        document.querySelectorAll('#view-character .badge--warning').forEach(function (badge) {
          badge.classList.remove('badge--warning');
          badge.classList.add('badge--brand');
          badge.innerHTML = '<svg class="icon icon--sm" aria-hidden="true"><use href="#i-check"></use></svg>已校对';
        });
        showToast('已标记为「已校对」。');
        break;

      case 'upload-image':
        showToast('原型演示：选择本地图片后上传（JPG / PNG / WebP，≤10MB）。', { type: 'info' });
        break;

      case 'remove-image':
        showToast('原型演示：已移除人物主图。', { type: 'info' });
        break;

      case 'add-chronicle':
        showToast('原型演示：新增大事记表单见 PG-03 / PG-05。', { type: 'info' });
        break;

      case 'edit-chronicle':
        showToast('原型演示：内联编辑大事记。', { type: 'info' });
        break;

      case 'delete-chronicle':
        openConfirm({
          title: '删除这条大事记？',
          desc: '删除后进入回收站，可在 8 秒内撤销。',
          actionLabel: '删除',
          onConfirm: function () { showToast('大事记已删除，可在回收站恢复。'); }
        });
        break;

      case 'edit-faction':
        showToast('势力编辑为页面内联表单，交互规范见 PG-05；原型仅展示阅览态。', { type: 'info' });
        break;

      case 'more-actions':
        showToast('更多操作：移入回收站、复制链接、查看变更记录（原型未展开）。', { type: 'info' });
        break;

      case 'assign-seat':
        openAssignDialog(actionEl.dataset.seat);
        break;

      case 'seat-menu':
        showToast('第 ' + actionEl.dataset.seat + ' 席操作：移动、交换、移出榜单（原型未展开）。', { type: 'info' });
        break;

      case 'jump-seat':
        jumpToSeat();
        break;

      case 'expand-ranking':
        showToast('扩容席位：当前 100 席，可扩容至 200 席；缩容前需检查引用。', { type: 'info' });
        break;

      case 'add-category':
        showToast('新增体系类别（当前 5 / 10）；原型未覆盖表单。', { type: 'info' });
        break;

      case 'add-node':
        showToast('新增主层级节点；同一类别最多两级。', { type: 'info' });
        break;

      case 'add-child-node':
        showToast('新增子层级节点；节点最多两级。', { type: 'info' });
        break;

      case 'toggle-node':
        toggleNode(actionEl);
        break;

      case 'move-node-up':
        showToast('已上移节点（原型演示，不写入数据）。', { type: 'info' });
        break;

      case 'move-node-down':
        showToast('已下移节点（原型演示，不写入数据）。', { type: 'info' });
        break;

      case 'node-menu':
        showToast('节点操作：重命名、停用、删除；删除前检查人物引用。', { type: 'info' });
        break;

      case 'choose-files':
        showToast('原型演示：已选择「山海志异设定.md」与「images.zip」。', { type: 'info' });
        break;

      case 'remove-file': {
        const row = actionEl.closest('.file-row');
        if (row) row.remove();
        showToast('已移除文件。', { type: 'info' });
        break;
      }

      case 'download-template':
        showToast('原型演示：下载固定模板 v1.0。', { type: 'info' });
        break;

      case 'download-sample':
        showToast('原型演示：下载示例文件（可被解析）。', { type: 'info' });
        break;

      case 'cancel-import':
        showToast('已取消导入，未写入任何数据。', { type: 'info' });
        break;

      case 'start-validation': {
        const files = importPanel1 ? importPanel1.querySelectorAll('.file-row') : [];
        if (!files.length) {
          showToast('请先选择固定模板 Markdown 与图片 zip。', { type: 'error' });
          break;
        }
        setImportStep(2);
        showToast('校验完成：发现 2 个错误，整批暂不写入。', { type: 'error' });
        break;
      }

      case 'reupload':
        setImportStep(1);
        showToast('请重新选择文件后再次校验。', { type: 'info' });
        break;

      case 'restore-item':
        removeTrashRow(actionEl.closest('tr[data-name]'), '已恢复「' + actionEl.dataset.name + '」到原位置。');
        break;

      case 'delete-item':
        openConfirm({
          title: '永久删除「' + actionEl.dataset.name + '」？',
          desc: '此操作不可撤销。删除前会自动创建快照，快照失败时不会执行删除。',
          actionLabel: '永久删除',
          onConfirm: function () {
            removeTrashRow(actionEl.closest('tr[data-name]'), '已永久删除「' + actionEl.dataset.name + '」。');
          }
        });
        break;

      case 'empty-trash': {
        const total = trashRows().length;
        if (!total) {
          showToast('回收站已为空。', { type: 'info' });
          break;
        }
        openConfirm({
          title: '清空回收站？',
          desc: '将永久删除全部 ' + total + ' 项，包括整本作品与整组类别。此操作不可撤销；删除前会自动创建快照。',
          actionLabel: '清空回收站',
          onConfirm: function () {
            trashRows().forEach(function (row) { row.remove(); });
            refreshTrash();
            showToast('回收站已清空。');
          }
        });
        break;
      }

      case 'reset-trash-filters':
        if (trashSearch) trashSearch.value = '';
        if (trashType) trashType.selectedIndex = 0;
        if (trashWork) trashWork.selectedIndex = 0;
        refreshTrash();
        showToast('已重置回收站筛选。', { type: 'info' });
        break;

      case 'new-work':
        closeWorkMenu();
        showToast('新建作品表单见 PG-00 / PG-13 规格；原型未覆盖。', { type: 'info' });
        break;

      case 'edit-work':
        closeWorkMenu();
        showToast('编辑当前作品：名称、简介与封面（原型未覆盖）。', { type: 'info' });
        break;

      case 'delete-work':
        closeWorkMenu();
        openConfirm({
          title: '删除作品《' + (currentWorkName ? currentWorkName.textContent : '当前作品') + '》？',
          desc: '整本作品及其人物、势力、地点、榜单与体系将进入回收站，可在 8 秒内撤销。删除前自动创建快照。',
          actionLabel: '删除作品',
          onConfirm: function () { showToast('作品已移入回收站，可在 8 秒内撤销。'); }
        });
        break;

      default:
        break;
    }
  });

  document.addEventListener('click', function (event) {
    const closeBtn = event.target.closest('[data-close-dialog]');
    if (closeBtn) {
      const dialog = closeBtn.closest('dialog');
      if (dialog) dialog.close();
      return;
    }

    const removeChip = event.target.closest('.chip__remove');
    if (removeChip) {
      const chip = removeChip.closest('.chip');
      if (chip) chip.remove();
    }
  });

  /* 回收站筛选 */
  if (trashSearch) trashSearch.addEventListener('input', refreshTrash);
  if (trashType) trashType.addEventListener('change', refreshTrash);
  if (trashWork) trashWork.addEventListener('change', refreshTrash);

  /* 榜单分页按钮（数字与箭头） */
  if (rankingView) {
    rankingView.querySelector('.pagination').addEventListener('click', function (event) {
      const btn = event.target.closest('.page-btn');
      if (!btn || btn.disabled) return;
      if (btn.dataset.action === 'ranking-page-2') {
        renderSeats(2);
        return;
      }
      const numeric = btn.textContent.trim();
      if (numeric === '1') {
        renderSeats(1);
        return;
      }
      if (btn.getAttribute('aria-label') === '上一页') {
        renderSeats(1);
        return;
      }
      if (btn.getAttribute('aria-label') === '下一页') {
        renderSeats(2);
      }
    });
  }

  /* ---------------------------------------------------------
     初始化
     --------------------------------------------------------- */
  renderSeats(1);
  setImportStep(2);
  refreshTrash();
  syncRoute();
})();
