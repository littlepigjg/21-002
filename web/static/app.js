"use strict";

function show(el) {
  el.hidden = false;
}

function hide(el) {
  el.hidden = true;
}

function renderJSON(el, data) {
  el.textContent = JSON.stringify(data, null, 2);
  show(el);
}

async function postJSON(url, body) {
  const resp = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return resp.json();
}

async function getJSON(url) {
  const resp = await fetch(url);
  return resp.json();
}

function escapeHTML(s) {
  return String(s)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
}

// 单篇提交
document.getElementById("single-form").addEventListener("submit", async (e) => {
  e.preventDefault();
  const resultEl = document.getElementById("single-result");
  hide(resultEl);
  const title = document.getElementById("single-title").value.trim();
  const content = document.getElementById("single-content").value.trim();
  if (!content) {
    alert("正文不能为空");
    return;
  }
  try {
    const data = await postJSON("/api/v1/articles", { title, content });
    renderJSON(resultEl, data);
  } catch (err) {
    resultEl.textContent = "请求失败：" + err.message;
    show(resultEl);
  }
});

// 批量提交
document.getElementById("batch-form").addEventListener("submit", async (e) => {
  e.preventDefault();
  const resultEl = document.getElementById("batch-result");
  hide(resultEl);
  const text = document.getElementById("batch-content").value.trim();
  if (!text) {
    alert("请至少输入一篇文章");
    return;
  }
  const articles = text.split("\n").filter((l) => l.trim()).map((line) => {
    const idx = line.indexOf("|");
    if (idx === -1) {
      return { title: "", content: line.trim() };
    }
    return { title: line.slice(0, idx).trim(), content: line.slice(idx + 1).trim() };
  });
  try {
    const data = await postJSON("/api/v1/batch", { articles });
    renderJSON(resultEl, data);
  } catch (err) {
    resultEl.textContent = "请求失败：" + err.message;
    show(resultEl);
  }
});

// 任务查询
document.getElementById("query-task").addEventListener("click", async () => {
  const resultEl = document.getElementById("task-result");
  hide(resultEl);
  const id = document.getElementById("task-id").value.trim();
  if (!id) {
    alert("请输入任务 ID");
    return;
  }
  try {
    const data = await getJSON("/api/v1/tasks/" + encodeURIComponent(id));
    renderJSON(resultEl, data);
  } catch (err) {
    resultEl.textContent = "请求失败：" + err.message;
    show(resultEl);
  }
});

// 历史记录
document.getElementById("load-history").addEventListener("click", async () => {
  const list = document.getElementById("history-list");
  list.innerHTML = "<p>加载中...</p>";
  try {
    const data = await getJSON("/api/v1/articles?offset=0&limit=20");
    const items = (data.data && data.data.items) || [];
    list.innerHTML = "";
    if (items.length === 0) {
      list.innerHTML = "<p>暂无历史记录</p>";
      return;
    }
    for (const a of items) {
      const div = document.createElement("div");
      div.className = "item";
      div.innerHTML =
        '<div class="title">' + escapeHTML(a.title || "(无标题)") + "</div>" +
        '<div class="summary">状态：' + escapeHTML(a.status) + " · ID：" + escapeHTML(a.id) + "</div>";
      list.appendChild(div);
    }
  } catch (err) {
    list.innerHTML = "<p>加载失败：" + escapeHTML(err.message) + "</p>";
  }
});

// 健康检查
(async function checkHealth() {
  const el = document.getElementById("health");
  try {
    const data = await getJSON("/health");
    el.textContent = (data.data && data.data.status) || "up";
    el.style.color = "#1e8e3e";
  } catch (err) {
    el.textContent = "不可用";
    el.style.color = "#c0392b";
  }
})();
